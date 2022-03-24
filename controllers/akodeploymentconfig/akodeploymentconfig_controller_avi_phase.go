// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig

import (
	"bytes"
	"context"
	"errors"

	"net"
	"sort"

	"github.com/avinetworks/sdk/go/models"
	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/phases"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/user"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/haprovider"

	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	akov1alpha1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
)

func getAviCAFromADC(c client.Client, ctx context.Context,
	log logr.Logger, obj *akoov1alpha1.AKODeploymentConfig) (string, error) {
	aviControllerCA := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{
		Name:      obj.Spec.CertificateAuthorityRef.Name,
		Namespace: obj.Spec.CertificateAuthorityRef.Namespace,
	}, aviControllerCA); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cannot find referenced secret " + obj.Spec.CertificateAuthorityRef.Name + "/" + obj.Spec.CertificateAuthorityRef.Namespace + " requeue the request")
		} else {
			log.Error(err, "Failed to find referenced "+obj.Spec.CertificateAuthorityRef.Name+"/"+obj.Spec.CertificateAuthorityRef.Namespace+" Secret")
		}
		return "", err
	}

	return string(aviControllerCA.Data["certificateAuthorityData"][:]), nil
}

func (r *AKODeploymentConfigReconciler) initAVI(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	var currentCa, newCa string

	if r.aviClient != nil {
		currentCa, _ = r.aviClient.AviCertificateConfig()
	}
	newCa, err := getAviCAFromADC(r.Client, ctx, log, obj)
	if err != nil {
		return res, err
	}
	reInit := currentCa != newCa

	// Lazily initialize aviClient so we don't skip other reconciliations
	if r.aviClient == nil || reInit {
		var err error
		r.aviClient, err = aviclient.NewAviClientFromSecrets(r.Client, ctx, log, obj.Spec.Controller,
			obj.Spec.AdminCredentialRef.Name, obj.Spec.AdminCredentialRef.Namespace,
			obj.Spec.CertificateAuthorityRef.Name, obj.Spec.CertificateAuthorityRef.Namespace)

		if err != nil {
			log.Error(err, "Cannot init AVI clients from secrets")
			return res, err
		}
		log.Info("AVI Client initialized successfully")
	}

	if r.userReconciler == nil || reInit {
		r.userReconciler = user.NewProvider(r.Client, r.aviClient, r.Log, r.Scheme)
		log.Info("Ako User Reconciler initialized")
	}

	return res, nil
}

// reconcileAVI reconciles every cluster that matches the
// AKODeploymentConfig's selector by conducting AVI related operations
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileAVI(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if res, err := r.initAVI(ctx, log, obj); err != nil {
		log.Error(err, "Failed to initialize avi related clients")
		return res, err
	}

	return phases.ReconcilePhases(ctx, log, obj, []phases.ReconcilePhase{
		r.reconcileNetworkSubnets,
		r.reconcileAviInfraSetting,
		func(ctx context.Context, log logr.Logger, obj *akoov1alpha1.AKODeploymentConfig) (ctrl.Result, error) {
			return phases.ReconcileClustersPhases(ctx, r.Client, log, obj,
				[]phases.ReconcileClusterPhase{
					r.userReconciler.ReconcileAviUser,
				},
				[]phases.ReconcileClusterPhase{
					r.userReconciler.ReconcileAviUserDelete,
				},
			)
		},
	})
}

// reconcileAVIDelete reconciles every cluster that matches the
// AKODeploymentConfig's selector by conducting AVI related operations
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileAVIDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if res, err := r.initAVI(ctx, log, obj); err != nil {
		log.Error(err, "Failed to initialize avi related clients")
		return res, err
	}

	return phases.ReconcilePhases(ctx, log, obj, []phases.ReconcilePhase{
		r.reconcileAviInfraSettingDelete,
		func(ctx context.Context, log logr.Logger, obj *akoov1alpha1.AKODeploymentConfig) (ctrl.Result, error) {
			return phases.ReconcileClustersPhases(ctx, r.Client, log, obj,
				[]phases.ReconcileClusterPhase{
					r.userReconciler.ReconcileAviUserDelete,
				},
				[]phases.ReconcileClusterPhase{
					// TODO(fangyuanl): handle the data network configuration
					// deletion
					r.userReconciler.ReconcileAviUserDelete,
				},
			)
		},
	})

}

// reconcileNetworkSubnets ensures the Datanetwork configuration is in sync with
// AVI Controller configuration
func (r *AKODeploymentConfigReconciler) reconcileNetworkSubnets(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	log.Info("Start reconciling AVI Network Subnets")

	if r.aviClient == nil {
		log.Info("AVI client not initialized, requeue")
		return res, errors.New("AVI client not initialized")
	}

	network, err := r.aviClient.NetworkGetByName(obj.Spec.DataNetwork.Name)
	if err != nil {
		log.Info("[WARN] Failed to get the Data Network from AVI Controller")
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
		_, err := r.aviClient.NetworkUpdate(network)
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

// func (r *AKODeploymentConfigReconciler) reconcileCloudUsableNetwork(
// 	ctx context.Context,
// 	log logr.Logger,
// 	obj *akoov1alpha1.AKODeploymentConfig,
// ) (ctrl.Result, error) {
// 	log = log.WithValues("cloud", obj.Spec.CloudName)
// 	log.Info("Start reconciling AVI cloud usable network")

// 	added, err := r.AddUsableNetwork(r.aviClient, obj.Spec.CloudName, obj.Spec.DataNetwork.Name)
// 	if err != nil {
// 		log.Error(err, "Failed to add usable network", "network", obj.Spec.DataNetwork.Name)
// 		return ctrl.Result{}, err
// 	}
// 	if added {
// 		log.Info("Added Usable Network", "network", obj.Spec.DataNetwork.Name)
// 	} else {
// 		log.Info("Network is already one of the cloud's usable network", "network", obj.Spec.DataNetwork.Name)
// 	}
// 	return ctrl.Result{}, nil
// }

func (r *AKODeploymentConfigReconciler) reconcileAviInfraSetting(
	ctx context.Context,
	log logr.Logger,
	adc *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	log.Info("Start reconciling AVIInfraSetting")

	if adc.Spec.ControlPlaneNetwork.Name == "" {
		log.Info("ControlPlaneNetwork is empty in akoDeploymentConfig, skip creating AVIInfraSetting")
		return res, nil
	}

	newAviInfraSetting := r.createAviInfraSetting(adc)
	aviInfraSetting := &akov1alpha1.AviInfraSetting{}

	if err := r.Get(ctx, client.ObjectKey{
		Name: haprovider.GetAviInfraSettingName(adc),
	}, aviInfraSetting); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AVIInfraSetting doesn't exist, start creating it")
			return res, r.Create(ctx, newAviInfraSetting)
		}
		log.Error(err, "Failed to get AVIInfraSetting, requeue")
		return res, err
	}
	newAviInfraSetting.Spec.DeepCopyInto(&aviInfraSetting.Spec)
	return res, r.Update(ctx, aviInfraSetting)
}

func (r *AKODeploymentConfigReconciler) createAviInfraSetting(adc *akoov1alpha1.AKODeploymentConfig) *akov1alpha1.AviInfraSetting {
	// ShardVSSize describes ingress shared virtual service size, default value is SMALL
	shardSize := "SMALL"
	if adc.Spec.ExtraConfigs.IngressConfigs.ShardVSSize != "" {
		shardSize = adc.Spec.ExtraConfigs.IngressConfigs.ShardVSSize
	}

	return &akov1alpha1.AviInfraSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name: haprovider.GetAviInfraSettingName(adc),
		},
		Spec: akov1alpha1.AviInfraSettingSpec{
			SeGroup: akov1alpha1.AviInfraSettingSeGroup{
				Name: adc.Spec.ServiceEngineGroup,
			},
			Network: akov1alpha1.AviInfraSettingNetwork{
				VipNetworks: []akov1alpha1.AviInfraSettingVipNetwork{{
					NetworkName: adc.Spec.ControlPlaneNetwork.Name,
					Cidr:        adc.Spec.ControlPlaneNetwork.CIDR,
				}},
			},
			L7Settings: akov1alpha1.AviInfraL7Settings{
				ShardSize: shardSize,
			},
		},
	}
}

func (r *AKODeploymentConfigReconciler) reconcileAviInfraSettingDelete(
	ctx context.Context,
	log logr.Logger,
	adc *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	log.Info("Start reconciling AVIInfraSetting Delete")

	// Get the list of clusters managed by the AKODeploymentConfig
	clusters, err := phases.ListAkoDeplymentConfigDeployedClusters(ctx, r.Client, adc)
	if err != nil {
		log.Error(err, "Fail to list clusters deployed by current AKODeploymentConfig")
		return res, err
	}

	if len(clusters.Items) != 0 {
		log.Info("There are clusters managed by current AKODeploymentConfig, skip AviInfraSetting deletion")
		return res, nil
	}

	aviInfraSetting := &akov1alpha1.AviInfraSetting{}
	if err := r.Get(ctx, client.ObjectKey{
		Name: haprovider.GetAviInfraSettingName(adc),
	}, aviInfraSetting); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AVIInfraSetting doesn't exist, skip deletion")
			return res, nil
		}
		log.Error(err, "Failed to get AVIInfraSetting, requeue")
		return res, err
	}
	return res, r.Delete(ctx, aviInfraSetting)
}

// EnsureAviNetwork brings network to the intented state by ensuring there is
// one subnet in network that has the specified cidr/mask and ipPools
func EnsureAviNetwork(network *models.Network, addrType string, cidr *net.IPNet, mask int32, ipPools []akoov1alpha1.IPPool, log logr.Logger) bool {
	var foundSubnet, modified bool
	var index int
	if index, foundSubnet = AviNetworkContainsSubnet(network, cidr.IP.String(), mask); foundSubnet {
		log.V(3).Info("Found matching subnet", "subnet", network.ConfiguredSubnets[index])
		subnet := network.ConfiguredSubnets[index]
		modified = EnsureStaticRanges(subnet, ipPools, addrType)
		if modified {
			// Update the original subnet in network
			network.ConfiguredSubnets[index] = subnet
			log.V(3).Info("Found matching subnet, after merging", "subnet", network.ConfiguredSubnets[index])
		}
	} else {
		// If there is no such subnet, create one
		subnet := &models.Subnet{
			Prefix: &models.IPAddrPrefix{
				IPAddr: GetAddr(cidr.IP.String(), addrType),
				Mask:   &mask,
			},
			// Add IP pools as static ranges in the subnet
			StaticRanges: CreateStaticRangeFromIPPools(ipPools),
		}
		// Add subnet into the network
		network.ConfiguredSubnets = append(network.ConfiguredSubnets, subnet)
		modified = true
	}
	return modified
}

// ensureStaticRanges creates or updates the subnet's static ranges to ensure IP
// ranges in IPPools are reflected in the subnet. It does so by firstly doing a
// sort on the static ranges, then try to extend an exisitng range or fill in
// the hole.
// Note: ippools are guaranteed to be non-overlapping by validation
func EnsureStaticRanges(subnet *models.Subnet, ipPools []akoov1alpha1.IPPool, addrType string) bool {
	// no ip pools specified in akodeploymentconfig, don't update subnet settings
	if ipPools == nil {
		return false
	}
	newStaticRanges := CreateStaticRangeFromIPPools(ipPools)
	res := !IsStaticRangeEqual(newStaticRanges, subnet.StaticRanges)
	if res {
		subnet.StaticRanges = newStaticRanges
	}
	return res
}

func IsStaticRangeEqual(r1, r2 []*models.IPAddrRange) bool {
	SortStaticRanges(r1)
	SortStaticRanges(r2)
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

func SortStaticRanges(staticRanges []*models.IPAddrRange) {
	sort.Slice(staticRanges, func(i, j int) bool {
		return isIPLessThan(*staticRanges[i].Begin.Addr, *staticRanges[j].Begin.Addr)
	})
}

func isIPLessThan(a, b string) bool {
	aIP := net.ParseIP(a)
	bIP := net.ParseIP(b)
	return bytes.Compare(aIP, bIP) < 0
}

func AviNetworkContainsSubnet(network *models.Network, startAddr string, mask int32) (int, bool) {
	for i, subnet := range network.ConfiguredSubnets {
		if *(subnet.Prefix.IPAddr.Addr) == startAddr && *(subnet.Prefix.Mask) == mask {
			return i, true
		}
	}
	return -1, false
}

func CreateStaticRangeFromIPPools(ipPools []akoov1alpha1.IPPool) []*models.IPAddrRange {
	newStaticRanges := []*models.IPAddrRange{}
	for _, ipPool := range ipPools {
		newStaticRanges = append(newStaticRanges, &models.IPAddrRange{
			Begin: GetAddr(ipPool.Start, ipPool.Type),
			End:   GetAddr(ipPool.End, ipPool.Type),
		})
	}
	return newStaticRanges
}

func GetAddr(addr string, addrType string) *models.IPAddr {
	return &models.IPAddr{
		Addr: &addr,
		Type: &addrType,
	}
}
