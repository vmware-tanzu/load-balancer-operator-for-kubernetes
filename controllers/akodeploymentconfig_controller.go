/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"net"
	"sort"

	"github.com/avinetworks/sdk/go/models"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/aviclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (r *AKODeploymentConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&akoov1alpha1.AKODeploymentConfig{}).
		Complete(r)
}

// AKODeploymentConfigReconciler reconciles a AKODeploymentConfig object
type AKODeploymentConfigReconciler struct {
	client.Client
	aviClient *aviclient.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=network.tanzu.vmware.com,resources=akodeploymentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.tanzu.vmware.com,resources=akodeploymentconfigs/status,verbs=get;update;patch

func (r *AKODeploymentConfigReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	log := r.Log.WithValues("AKODeploymentConfig", req.NamespacedName)
	res := ctrl.Result{}

	// Get the resource for this request.
	obj := &akoov1alpha1.AKODeploymentConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Machine not found, will not reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Always Patch when exiting this function so changes to the resource are updated on the API server.
	patchHelper, err := patch.NewHelper(obj, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to init patch helper for %s %s",
			obj.GroupVersionKind(), req.NamespacedName)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, obj); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "patch failed")
		}
	}()

	if r.aviClient == nil {
		adminCredential := &corev1.Secret{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      obj.Spec.AdminCredentialRef.Name,
			Namespace: obj.Spec.AdminCredentialRef.Namespace,
		}, adminCredential); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Cannot find referenced AdminCredential Secret, requeue the request")
			} else {
				log.Error(err, "Failed to find referenced AdminCredential Secret")
			}
			return res, err
		}

		aviControllerCA := &corev1.Secret{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      obj.Spec.CertificateAuthorityRef.Name,
			Namespace: obj.Spec.CertificateAuthorityRef.Namespace,
		}, aviControllerCA); err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("Cannot find referenced CertificateAuthorityRef Secret, requeue the request")
			} else {
				log.Error(err, "Failed to find referenced CertificateAuthorityRef Secret")
			}
			return res, err
		}
		aviClient, err := aviclient.NewAviClient(&aviclient.AviClientConfig{
			ServerIP: obj.Spec.Controller,
			Username: string(adminCredential.Data["username"][:]),
			Password: string(adminCredential.Data["password"][:]),
			// CA:         string(aviControllerCA.Data["certificateAuthorityData"][:]),
		})
		if err != nil {
			log.Error(err, "Failed to initialize AVI Controller Client, requeue the request")
			return res, err
		}

		r.aviClient = aviClient
		log.Info("AVI Client initialized successfully")
	}

	// Handle deleted cluster resources.
	if !obj.GetDeletionTimestamp().IsZero() {
		res, err := r.reconcileDelete(ctx, log, obj)
		if err != nil {
			log.Error(err, "failed to reconcile AKODeploymentConfig deletion")
			return res, err
		}
		return res, nil
	}

	// Handle non-deleted resources.
	if res, err := r.reconcileNormal(ctx, log, obj); err != nil {
		log.Error(err, "failed to reconcile AKODeploymentConfig")
		return res, err
	}
	return res, nil
}

func (r *AKODeploymentConfigReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *AKODeploymentConfigReconciler) reconcileNormal(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	phases := []func(ctx context.Context, log logr.Logger, obj *akoov1alpha1.AKODeploymentConfig) (ctrl.Result, error){
		r.reconcileNetworkSubnets,
	}
	errs := []error{}
	for _, phase := range phases {
		// Call the inner reconciliation methods.
		phaseResult, err := phase(ctx, log, obj)
		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			continue
		}
		res = util.LowestNonZeroResult(res, phaseResult)
	}
	return res, kerrors.NewAggregate(errs)
}

func (r *AKODeploymentConfigReconciler) reconcileNetworkSubnets(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	log.Info("Start reconciling AVI Network Subnets")

	network, err := r.aviClient.Network.GetByName(obj.Spec.DataNetwork.Name)
	if err != nil {
		log.Error(err, "Failed to get the Data Network from AVI Controller")
		return res, nil
	}

	// TODO(fangyuanl): move validation to webhook
	// We also need to make sure IPPools are not overlapping
	addr, cidr, err := net.ParseCIDR(obj.Spec.DataNetwork.CIDR)
	if err != nil {
		log.Error(err, "Failed to parse the Data Network CIDR")
		return res, nil
	}
	ones, _ := cidr.Mask.Size()
	mask := int32(ones)

	addrType := "V4"
	if addr.To4() == nil {
		addrType = "V6"
	}

	modified := EnsureAviNetwork(network, addrType, cidr, mask, obj.Spec.DataNetwork.IPPools, log)

	if modified {
		log.V(3).Info("Change detected, updating Network", "network", obj.Spec.DataNetwork.Name)
		_, err := r.aviClient.Network.Update(network)
		if err != nil {
			log.Error(err, "Failed to update Network, requeue the request", "network", network)
			return res, err
		}
		log.Info("Successfully updated Network", "subnets", network.ConfiguredSubnets)
	} else {
		log.Info("No change detected for Network", "network", obj.Spec.DataNetwork.Name)
	}

	return res, nil
}

func GetAddr(addr string, addrType string) *models.IPAddr {
	return &models.IPAddr{
		Addr: &addr,
		Type: &addrType,
	}
}

// EnsureAviNetwork brings network to the intented state by ensuring there is
// one subnet in network that has the specified cidr/mask and ipPools
func EnsureAviNetwork(network *models.Network, addrType string, cidr *net.IPNet, mask int32, ipPools []akoov1alpha1.IPPool, log logr.Logger) bool {
	var foundSubnet, modified bool
	for i, subnet := range network.ConfiguredSubnets {
		if *subnet.Prefix.IPAddr.Addr == cidr.IP.String() && *subnet.Prefix.Mask == mask {
			foundSubnet = true
			log.V(3).Info("Found matching subnet", "subnet", network.ConfiguredSubnets[i])
			modified = EnsureStaticRanges(subnet, ipPools, addrType)
			if modified {
				// Update the original subnet in network
				network.ConfiguredSubnets[i] = subnet
				log.V(3).Info("Found matching subnet, after merging", "subnet", network.ConfiguredSubnets[i])
			}
			break
		}
	}
	// If there is no such subnet, create one
	if !foundSubnet {
		subnet := &models.Subnet{
			Prefix: &models.IPAddrPrefix{
				IPAddr: &models.IPAddr{},
				Mask:   &mask,
			},
		}
		// Add static IPs as static ranges in the subnet
		for _, ipPool := range ipPools {
			subnet.StaticRanges = append(subnet.StaticRanges, &models.IPAddrRange{
				Begin: GetAddr(ipPool.Start, ipPool.Type),
				End:   GetAddr(ipPool.End, ipPool.Type),
			})
		}
		// Add subnet into the network
		network.ConfiguredSubnets = append(network.ConfiguredSubnets, subnet)
	}
	return !foundSubnet || modified
}

// ensureStaticRanges creates or updates the subnet's static ranges to ensure IP
// ranges in IPPools are reflected in the subnet. It does so by firstly doing a
// sort on the static ranges, then try to extend an exisitng range or fill in
// the hole.
// Note: ippools are guaranteed to be non-overlapping by validation
func EnsureStaticRanges(subnet *models.Subnet, ipPools []akoov1alpha1.IPPool, addrType string) bool {
	SortSubnetStaticRanges(subnet)
	SortIPPools(ipPools)

	newStaticRanges := []*models.IPAddrRange{}
	i, j := 0, 0
	var start string
	// merging intervals
	for i < len(subnet.StaticRanges) && j < len(ipPools) {
		// seeing a gap, we create a new range
		if isIPLessThan(*subnet.StaticRanges[i].End.Addr, ipPools[j].Start) {
			if start != "" {
				newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
					Begin: GetAddr(start, addrType),
					End:   subnet.StaticRanges[i].End,
				})
			} else {
				newStaticRanges = append(newStaticRanges, subnet.StaticRanges[i])
			}
			start = ipPools[j].Start
			i++
		} else {
			// seeing a gap, we create a new range
			if isIPLessThan(ipPools[j].End, *subnet.StaticRanges[i].Begin.Addr) {
				if start != "" {
					newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
						Begin: GetAddr(start, addrType),
						End:   GetAddr(ipPools[j].End, addrType),
					})
				} else {
					newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
						Begin: GetAddr(ipPools[j].Start, addrType),
						End:   GetAddr(ipPools[j].End, addrType),
					})
				}
				start = *subnet.StaticRanges[i].Begin.Addr
				j++
			} else {
				if start == "" {
					if isIPLessThan(ipPools[j].Start, *subnet.StaticRanges[i].Begin.Addr) {
						start = ipPools[j].Start
					} else {
						start = *subnet.StaticRanges[i].Begin.Addr
					}
				}
				if isIPLessThan(ipPools[j].End, *subnet.StaticRanges[i].End.Addr) {
					j++
				} else {
					i++
				}
			}
		}
	}

	for i < len(subnet.StaticRanges) {
		if start != "" {
			newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
				Begin: GetAddr(start, addrType),
				End:   subnet.StaticRanges[i].End,
			})
			start = ""
		} else {
			newStaticRanges = append(newStaticRanges, subnet.StaticRanges[i])
		}
		i++
	}

	for j < len(ipPools) {
		if start != "" {
			newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
				Begin: GetAddr(start, addrType),
				End:   GetAddr(ipPools[j].End, addrType),
			})
			start = ""
		} else {
			newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
				Begin: GetAddr(ipPools[j].Start, addrType),
				End:   GetAddr(ipPools[j].End, addrType),
			})
		}
		j++
	}

	res := !IsStaticRangeEqual(newStaticRanges, subnet.StaticRanges)
	if res {
		subnet.StaticRanges = newStaticRanges
	}
	return res
}

func IsStaticRangeEqual(r1, r2 []*models.IPAddrRange) bool {
	if len(r1) != len(r2) {
		return false
	}
	for i := 0; i < len(r1); i++ {
		if (*(r1[i].Begin.Addr) != *(r2[i].Begin.Addr)) || (*(r1[i].End.Addr) != *(r2[i].End.Addr)) {
			return false
		}
	}
	return true
}

func SortIPPools(pools []akoov1alpha1.IPPool) {
	sort.Slice(pools, func(i, j int) bool {
		return isIPLessThan(pools[i].Start, pools[j].Start)
	})
}

func SortSubnetStaticRanges(subnet *models.Subnet) {
	if subnet == nil {
		return
	}
	sort.Slice(subnet.StaticRanges, func(i, j int) bool {
		return isIPLessThan(*subnet.StaticRanges[i].Begin.Addr, *subnet.StaticRanges[j].Begin.Addr)
	})
}

func isIPLessThan(a, b string) bool {
	aIP := net.ParseIP(a)
	bIP := net.ParseIP(b)
	return bytes.Compare(aIP, bIP) < 0
}
