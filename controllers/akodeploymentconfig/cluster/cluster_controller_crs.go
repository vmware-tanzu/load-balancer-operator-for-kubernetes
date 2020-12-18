// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"text/template"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clustereaddonv1alpha3 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileCRS creates the CRS for AKO deployment in workload clusters
func (r *ClusterReconciler) reconcileCRS(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (res ctrl.Result, reterr error) {
	log.Info("Starts reconciling ClusterResourceSet")

	var errs []error
	s := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      akoDeploymentSecretName(obj),
		Namespace: cluster.Namespace,
	}, s); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AKO Deployment Secret doesn't exist, start creating")
			if s, err = akoDeploymentSecret(obj, cluster.Namespace, cluster); err != nil {
				log.Error(err, "Failed to generate AKO Deployment Secret")
				errs = append(errs, err)
			} else if err = r.Create(ctx, s); err != nil {
				log.Error(err, "Failed to create AKO Deployment Secret, requeue")
				errs = append(errs, err)
				return res, kerrors.NewAggregate(errs)
			} else if err = r.Get(ctx, client.ObjectKey{
				Name:      akoDeploymentSecretName(obj),
				Namespace: cluster.Namespace,
			}, s); err != nil {
				log.Error(err, "Failed to get AKO Deployment Secret after creation, requeue")
				errs = append(errs, err)
				return res, kerrors.NewAggregate(errs)
			}
		}
	}

	// Always patch the Secret at the end
	patchHelper, err := patch.NewHelper(s, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to init patch helper for %s %s",
			s.GroupVersionKind(), cluster.Namespace+"/"+cluster.Name)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, s); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "Failed to patch AKO Deployment Secret")
		}
	}()

	s, err = akoDeploymentSecret(obj, cluster.Namespace, cluster)
	if err != nil {
		log.Error(err, "Failed to generate AKO Deployment Secret")
		errs = append(errs, err)
	}

	crs := &clustereaddonv1alpha3.ClusterResourceSet{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      akoDeploymentCRSName(obj),
		Namespace: cluster.Namespace,
	}, crs); err != nil {
		if apierrors.IsNotFound(err) {
			crs = akoDeploymentCRS(obj, cluster)
			if err := r.Create(ctx, crs); err != nil {
				log.Error(err, "Failed to create AKO Deployment ClusterResourceSet")
				errs = append(errs, err)
			} else {
				log.Info("Created AKO Deployment ClusterResourceSet successfully")
				if err := r.Get(ctx, client.ObjectKey{
					Name:      akoDeploymentCRSName(obj),
					Namespace: cluster.Namespace,
				}, crs); err != nil {
					log.Error(err, "Failed to get AKO Deployment ClusterResourceSet after creation, requeue")
					errs = append(errs, err)
					return res, kerrors.NewAggregate(errs)
				}
			}
		}
	}

	// Always patch the ClusterResourceSet at the end
	patchHelper, err = patch.NewHelper(crs, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to init patch helper for %s %s",
			crs.GroupVersionKind(), cluster.Namespace+"/"+cluster.Name)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, crs); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "Failed to patch AKO Deployment ClusterResourceSet")
		}
	}()

	crs = akoDeploymentCRS(obj, cluster)

	return res, kerrors.NewAggregate(errs)
}

func akoDeploymentSecretName(obj *akoov1alpha1.AKODeploymentConfig) string {
	return obj.Name + "-ako-deployment"
}

var akoDeploymentCRSName = akoDeploymentSecretName

func akoDeploymentCRS(obj *akoov1alpha1.AKODeploymentConfig, cluster *clusterv1.Cluster) *clustereaddonv1alpha3.ClusterResourceSet {
	res := &clustereaddonv1alpha3.ClusterResourceSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      akoDeploymentCRSName(obj),
			Namespace: cluster.Namespace,
		},
		Spec: clustereaddonv1alpha3.ClusterResourceSetSpec{
			ClusterSelector: obj.Spec.ClusterSelector,
			Resources: []clustereaddonv1alpha3.ResourceRef{
				{
					Name: akoDeploymentSecretName(obj),
					Kind: "Secret",
				},
			},
		},
	}
	return res
}

func akoDeploymentSecret(obj *akoov1alpha1.AKODeploymentConfig, namespace string, cluster *clusterv1.Cluster) (*corev1.Secret, error) {
	res := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      akoDeploymentSecretName(obj),
			Namespace: namespace,
		},
		Type: clustereaddonv1alpha3.ClusterResourceSetSecretType,
		Data: make(map[string][]byte),
	}
	spec, err := AKODeploymentYaml(obj, cluster)
	if err != nil {
		return nil, err
	} else {
		res.Data["ako-deployment"] = []byte(spec)
	}
	return res, nil
}

func setDefaultValues(values *Values) {
	values.AKOSettings = AKOSettings{
		LogLevel:               "INFO",
		ApiServerPort:          8080,
		DeleteConfig:           false,
		DisableStaticRouteSync: true,
		// FullSyncFrequency: don't set, use default value in AKO
		// CniPlugin: don't set, use default value in AKO
		// SyncNamespace: don't set, use default value in AKO
		// ClusterName: populate in runtime
	}
	values.ReplicaCount = 1
	values.L7Settings = L7Settings{
		DefaultIngController: false,
		ServiceType:          "NodePort",
		// L7ShardingScheme: don't set, use default value in AKO
		// ShardVSSize: don't set, use default value in AKO
		// PassthroughShardSize: don't set, use default value in AKO
	}
	values.L4Settings = L4Settings{
		// DefaultDomain: don't set, use default value in AKO
	}
	values.ControllerSettings = ControllerSettings{
		// ServiceEngineGroupName: populate in runtime
		// CloudName: populate in runtime
		// ControllerIP: populate in runtime
		// ControllerVersion: don't set, depend on AKO to autodetect,
		// also because we don't consider version skew in Calgary
	}
	values.NodePortSelector = NodePortSelector{
		// Key: don't set, use default value in AKO
		// Value: don't set, use default value in AKO
	}
	values.Resources = Resources{
		Limits: Limits{
			Cpu:    "250m",
			Memory: "300Mi",
		},
		Requests: Requests{
			Cpu:    "100m",
			Memory: "200Mi",
		},
	}
	values.NetworkSettings = NetworkSettings{
		// SubnetIP: don't set, populate in runtime
		// SubnetPrefix: don't set, populate in runtime
		// NetworkName: don't set, populate in runtime
		// NodeNetworkList: don't set, use default value in AKO
		// NodeNetworkListJson: don't set, use default value in AKO
	}
	if len(values.NetworkSettings.NodeNetworkList) != 0 {
		// preprocessing
		nodeNetworkListJson, jsonerr := json.Marshal(values.NetworkSettings.NodeNetworkList)
		if jsonerr != nil {
			fmt.Println("Can't convert network setting into json. Error: ", jsonerr)
		}
		values.NetworkSettings.NodeNetworkListJson = string(nodeNetworkListJson)
	}
}

func PopluateValues(obj *akoov1alpha1.AKODeploymentConfig, cluster *clusterv1.Cluster) (Values, error) {
	values := Values{}

	setDefaultValues(&values)

	values.Image.Repository = obj.Spec.ExtraConfigs.Image.Repository
	values.Image.PullPolicy = obj.Spec.ExtraConfigs.Image.PullPolicy
	values.Image.Version = obj.Spec.ExtraConfigs.Image.Version

	values.AKOSettings.ClusterName = cluster.Name

	values.ControllerSettings.CloudName = obj.Spec.CloudName
	values.ControllerSettings.ControllerIP = obj.Spec.Controller
	values.ControllerSettings.ServiceEngineGroupName = obj.Spec.ServiceEngine

	network := obj.Spec.DataNetwork
	values.NetworkSettings.NetworkName = network.Name
	ip, ipnet, err := net.ParseCIDR(network.CIDR)
	if err != nil {
		return values, err
	}
	values.NetworkSettings.SubnetIP = ip.String()
	ones, _ := ipnet.Mask.Size()
	values.NetworkSettings.SubnetPrefix = strconv.Itoa(ones)

	values.PersistentVolumeClaim = obj.Spec.ExtraConfigs.Log.PersistentVolumeClaim
	values.MountPath = obj.Spec.ExtraConfigs.Log.MountPath
	values.LogFile = obj.Spec.ExtraConfigs.Log.LogFile

	values.DisableIngressClass = obj.Spec.ExtraConfigs.DisableIngressClass

	values.Name = "ako-" + cluster.Name

	values.Rbac = Rbac{
		PspEnabled:          obj.Spec.ExtraConfigs.Rbac.PspEnabled,
		PspPolicyApiVersion: obj.Spec.ExtraConfigs.Rbac.PspPolicyAPIVersion,
	}
	return values, nil
}

func AKODeploymentYaml(obj *akoov1alpha1.AKODeploymentConfig, cluster *clusterv1.Cluster) (string, error) {
	tmpl, err := template.New("deployment").Parse(akoDeploymentYamlTemplate)
	if err != nil {
		return "", err
	}

	values, err := PopluateValues(obj, cluster)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, map[string]interface{}{
		"Values": values,
	})

	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
