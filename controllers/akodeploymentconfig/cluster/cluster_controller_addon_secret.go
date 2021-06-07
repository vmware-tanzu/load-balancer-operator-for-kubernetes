// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"bytes"
	"context"
	"text/template"

	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	akoAddonSecretValues = `
#@data/values
#@overlay/match-child-defaults missing_ok=True
---
loadBalancerAndIngressService:
  name: {{ .Values.Name }}
  namespace: {{ .Values.Namespace }}
  config:
    is_cluster_service: {{ .Values.IsClusterService }}
    replica_count: {{ .Values.ReplicaCount }}
    image_settings:
      repository: {{ .Values.Image.Repository }}
      pull_policy: {{ .Values.Image.PullPolicy }}
      version: {{ .Values.Image.Version }}
    ako_settings:
      log_level: {{ .Values.AKOSettings.LogLevel }}
      full_sync_frequency: {{ .Values.AKOSettings.FullSyncFrequency }}
      api_server_port: {{ .Values.AKOSettings.ApiServerPort }}
      delete_config: {{ .Values.AKOSettings.DeleteConfig }}
      disable_static_route_sync:  {{ .Values.AKOSettings.DisableStaticRouteSync }}
      cluster_name: {{ .Values.AKOSettings.ClusterName }}
      cni_plugin: {{ .Values.AKOSettings.CniPlugin }}
      sync_namespace: {{ .Values.AKOSettings.SyncNamespace }}
    network_settings:
      subnet_ip: {{ .Values.NetworkSettings.SubnetIP }}
      subnet_prefix: {{ .Values.NetworkSettings.SubnetPrefix }}
      network_name: {{ .Values.NetworkSettings.NetworkName }}
      node_network_list: {{ .Values.NetworkSettings.NodeNetworkListJson }}
      vip_network_list: "[]"  
    l7_settings:
      disable_ingress_class: {{ .Values.L7Settings.DisableIngressClass }}
      default_ing_controller: {{ .Values.L7Settings.DefaultIngController }}
      l7_sharding_scheme: {{ .Values.L7Settings.L7ShardingScheme }}
      service_type: {{ .Values.L7Settings.ServiceType }}
      shard_vs_size: {{ .Values.L7Settings.ShardVSSize }}   
      pass_through_shardsize: {{ .Values.L7Settings.PassthroughShardSize }}
    l4_settings:
      default_domain: {{ .Values.L4Settings.DefaultDomain }}
    controller_settings: 
      service_engine_group_name: {{ .Values.ControllerSettings.ServiceEngineGroupName }}
      controller_version: {{ .Values.ControllerSettings.ControllerVersion }}
      cloud_name: {{ .Values.ControllerSettings.CloudName }}
      controller_ip: {{ .Values.ControllerSettings.ControllerIP }}
    nodeport_selector:
      key: {{ .Values.NodePortSelector.Key }}
      value: {{ .Values.NodePortSelector.Value }}
    resources:
      limits:
        cpu: {{ .Values.Resources.Limits.Cpu }}
        memory: {{ .Values.Resources.Limits.Memory }}
      request:
        cpu: {{ .Values.Resources.Requests.Cpu }}
        memory: {{ .Values.Resources.Requests.Memory }}
    rbac:
      psp_enabled: {{ .Values.Rbac.PspEnabled }}
      psp_policy_api_version: {{ .Values.Rbac.PspPolicyApiVersion }}
    persistent_volume_claim: {{ .Values.PersistentVolumeClaim }}
    mount_path: {{ .Values.MountPath }}
    log_file: {{ .Values.LogFile }}
`
)

func (r *ClusterReconciler) ReconcileAddonSecret(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Starts reconciling add on secret")
	res := ctrl.Result{}
	newSecret, err := r.createAKOAddonSecret(cluster, obj)
	if err != nil {
		log.Info("Failed to convert AKO Deployment Config to add-on secret, requeue the request")
		return res, err
	}
	secret := &corev1.Secret{}
	if err = r.Get(ctx, client.ObjectKey{
		Name:      r.akoAddonSecretName(cluster),
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AKO add on secret doesn't exist, start creating it")
			return res, r.Create(ctx, newSecret)
		}
		log.Error(err, "Failed to get AKO Deployment Secret, requeue")
		return res, err
	}
	secret = newSecret.DeepCopy()
	return res, r.Update(ctx, secret)
}

func (r *ClusterReconciler) ReconcileAddonSecretDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Starts reconciling add on secret deletion")
	res := ctrl.Result{}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      r.akoAddonSecretName(cluster),
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AKO add on secret already deleted")
			return res, nil
		}
		log.Error(err, "Failed to get AKO Deployment Secret, requeue")
		return res, err
	}
	return res, r.Delete(ctx, secret)
}

func (r *ClusterReconciler) akoAddonSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-addon"
}

func (r *ClusterReconciler) createAKOAddonSecret(cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig) (*corev1.Secret, error) {
	secretStringData, err := AkoAddonSecretYaml(cluster, obj)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.akoAddonSecretName(cluster),
			Namespace: cluster.Namespace,
			Annotations: map[string]string{
				akoov1alpha1.TKGAddonAnnotationKey: "networking/load-balancer-and-ingress-service",
			},
			Labels: map[string]string{
				akoov1alpha1.TKGAddOnLabelAddonNameKey:   "load-balancer-and-ingress-service",
				akoov1alpha1.TKGAddOnLabelClusterNameKey: cluster.Name,
				akoov1alpha1.TKGAddOnLabelClusterctlKey:  "",
			},
		},
		Type: akoov1alpha1.TKGAddOnSecretType,
		StringData: map[string]string{
			akoov1alpha1.TKGAddOnSecretDataKey: secretStringData,
		},
	}
	return secret, nil
}

func AkoAddonSecretYaml(cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig) (string, error) {
	tmpl, err := template.New("deployment").Parse(akoAddonSecretValues)
	if err != nil {
		return "", err
	}
	values, err := ako.PopulateValues(obj, cluster.Namespace+"-"+cluster.Name)
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
