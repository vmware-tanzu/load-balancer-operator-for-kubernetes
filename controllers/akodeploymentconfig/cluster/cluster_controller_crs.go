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
	"time"

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

const (
	requeueAfterCreation = time.Millisecond * 100
)

// ReconcileCRS ensures the CRS and the resources associated with it for a AKO
// deployment in the target workload cluster exist
func (r *ClusterReconciler) ReconcileCRS(
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
		errs = append(errs, err)
		if apierrors.IsNotFound(err) {
			log.Info("AKO Deployment Secret doesn't exist, start creating it")
			s, err = akoDeploymentSecret(nil, obj, cluster.Namespace, cluster)
			if err != nil {
				log.Error(err, "Failed to generate AKO Deployment Secret, requeue")
				return res, kerrors.NewAggregate(append(errs, err))
			}
			// Always requeue to trigger a reconcile,
			// hopefully the Create succeeds then the next
			// Get would succeed as well
			return ctrl.Result{Requeue: true, RequeueAfter: requeueAfterCreation}, r.Create(ctx, s)
		}
		log.Error(err, "Failed to get AKO Deployment Secret, requeue")
		return res, kerrors.NewAggregate(errs)
	}

	defer func() {
		// patch package doesn't support Service object because it
		// doesn't have neither spec nor status. So Update is used
		if err := r.Client.Update(ctx, s); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "Failed to update AKO Deployment Secret")
		}
	}()

	// Reconstruct the secret
	s, err := akoDeploymentSecret(s, obj, cluster.Namespace, cluster)
	if err != nil {
		log.Error(err, "Failed to generate AKO Deployment Secret, requeue")
		return res, kerrors.NewAggregate(append(errs, err))
	}

	crs := &clustereaddonv1alpha3.ClusterResourceSet{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      akoDeploymentCRSName(obj),
		Namespace: cluster.Namespace,
	}, crs); err != nil {
		errs = append(errs, err)
		if apierrors.IsNotFound(err) {
			log.Info("AKO Deployment ClusterResourceSet doesn't exist, start creating it")
			crs = akoDeploymentCRS(nil, obj, cluster)
			// Always requeue to trigger a reconcile,hopefully the
			// Create succeeds then the next Get would succeed as
			// well
			return ctrl.Result{Requeue: true, RequeueAfter: requeueAfterCreation}, r.Create(ctx, crs)
		}
		log.Error(err, "Failed to get AKO Deployment ClusterResourceSet, requeue")
		return res, kerrors.NewAggregate(errs)
	}

	// Always patch the ClusterResourceSet at the end
	crsPatchHelper, err := patch.NewHelper(crs, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to init patch helper for %s %s",
			crs.GroupVersionKind(), cluster.Namespace+"/"+cluster.Name)
	}
	defer func() {
		if err := crsPatchHelper.Patch(ctx, crs); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "Failed to patch AKO Deployment ClusterResourceSet")
		}
	}()

	// Reconstruct the CRS
	crs = akoDeploymentCRS(crs, obj, cluster)

	return res, kerrors.NewAggregate(errs)
}

// ReconcileCRSDelete ensures the CRS and its associated resources for a AKO
// deployment in the target workload cluster is deleted
func (r *ClusterReconciler) ReconcileCRSDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (res ctrl.Result, reterr error) {
	log.Info("Starts reconciling ClusterResourceSet")

	crs := &clustereaddonv1alpha3.ClusterResourceSet{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      akoDeploymentCRSName(obj),
		Namespace: cluster.Namespace,
	}, crs); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(3).Info("ClusterResourceSet is already deleted")
			return res, nil
		}
		log.Error(err, "Failed to get ClusterResourceSet, requeue")
		return res, err
	}

	// CAPI CRS controller will remove ClusterResourceBinding and CRS's
	// associated resources on our behalf, so deleting CRS is enough
	if err := r.Delete(ctx, crs); err != nil {
		log.Error(err, "Failed to delete ClusterResourceSet, requeue")
		return res, err
	}

	return ctrl.Result{}, nil
}

func akoDeploymentSecretName(obj *akoov1alpha1.AKODeploymentConfig) string {
	return "ako-deployment-" + obj.Name
}

var akoDeploymentCRSName = akoDeploymentSecretName

func akoDeploymentCRS(base *clustereaddonv1alpha3.ClusterResourceSet, obj *akoov1alpha1.AKODeploymentConfig, cluster *clusterv1.Cluster) *clustereaddonv1alpha3.ClusterResourceSet {
	var res *clustereaddonv1alpha3.ClusterResourceSet
	if base != nil {
		res = base
	} else {
		res = &clustereaddonv1alpha3.ClusterResourceSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      akoDeploymentCRSName(obj),
				Namespace: cluster.Namespace,
			},
		}
	}
	res.Spec.ClusterSelector = obj.Spec.ClusterSelector
	res.Spec.Resources = []clustereaddonv1alpha3.ResourceRef{
		{
			Name: akoDeploymentSecretName(obj),
			Kind: "Secret",
		},
	}
	// TODO(fangyuanl): add defaulting webhook
	if res.Spec.ClusterSelector.MatchExpressions == nil {
		res.Spec.ClusterSelector.MatchExpressions = []metav1.LabelSelectorRequirement{}
	}
	// TODO(fangyuanl): CAPI doesn't allow empty ClusterSelector so manually
	// add AVI cluster label
	// See https://github.com/kubernetes-sigs/cluster-api/pull/4036 for more
	// details.
	if res.Spec.ClusterSelector.MatchLabels == nil {
		res.Spec.ClusterSelector.MatchLabels = map[string]string{
			akoov1alpha1.AviClusterLabel: "",
		}
	}
	return res
}

func akoDeploymentSecret(base *corev1.Secret, obj *akoov1alpha1.AKODeploymentConfig, namespace string, cluster *clusterv1.Cluster) (*corev1.Secret, error) {
	var res *corev1.Secret
	if base != nil {
		res = base
	} else {
		res = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      akoDeploymentSecretName(obj),
				Namespace: namespace,
			},
		}
	}
	res.Type = clustereaddonv1alpha3.ClusterResourceSetSecretType
	if res.Data == nil {
		res.Data = make(map[string][]byte)
	}
	spec, err := AKODeploymentYaml(obj, cluster)
	if err != nil {
		return res, err
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
	values.Namespace = "avi-system"
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
