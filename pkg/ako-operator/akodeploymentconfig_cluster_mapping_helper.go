// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"context"

	"github.com/go-logr/logr"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListAkoDeplymentConfigSelectClusters list all clusters enabled current akodeploymentconfig
func ListAkoDeplymentConfigSelectClusters(
	ctx context.Context,
	kclient client.Client,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig) (*clusterv1.ClusterList, error) {
	// get all clusters can be selected by this akodeploymentconfig's cluster selector
	selector, err := metav1.LabelSelectorAsSelector(&obj.Spec.ClusterSelector)
	if err != nil {
		return nil, err
	}
	var clusters clusterv1.ClusterList
	if err := kclient.List(ctx, &clusters, []client.ListOption{
		client.MatchingLabelsSelector{Selector: selector},
	}...); err != nil {
		return nil, err
	}
	// remove clusters that:
	// 1. not ready
	// 2. management cluster
	// 3. previously selected by other adc objects
	var newItems []clusterv1.Cluster
	var allErrs []error
	for _, cluster := range clusters.Items {
		if !SkipCluster(&cluster) {
			// management cluster can't be selected by other adc objects
			// except the management cluster AKODeploymentConfig
			if cluster.Namespace == akoov1alpha1.TKGSystemNamespace &&
				obj.Name != akoov1alpha1.ManagementClusterAkoDeploymentConfig {
				continue
			}
			adcName, exist := cluster.Labels[akoov1alpha1.AviClusterLabel]
			// if cluster is already selected by other customized adc objects, skip
			// only clusters selected by default adc with empty selector object can be overrided
			if exist && adcName != obj.Name {
				if !isDefaultADC(adcName) || !defaultADCHasEmptySelector(ctx, kclient) {
					continue
				}
			}
			// update cluster adc label
			if !exist || adcName != obj.Name {
				applyClusterLabel(log, &cluster, obj)
				if err := kclient.Update(ctx, &cluster); err != nil {
					allErrs = append(allErrs, err)
				}
			}
			newItems = append(newItems, cluster)
		}
	}
	clusters.Items = newItems
	return &clusters, kerrors.NewAggregate(allErrs)
}

// UpdateClusterAKODeploymentConfigLabel updates the cluster's networking.tkg.tanzu.vmware.com/avi
// label to the akodeploymentconfig object currently selects this cluster
func UpdateClusterAKODeploymentConfigLabel(
	ctx context.Context,
	kclient client.Client,
	log logr.Logger,
	cluster *clusterv1.Cluster) (adc *akoov1alpha1.AKODeploymentConfig, err error) {
	// get akodeploymentconfig object for this cluster
	adcForCluster, err := getAKODeploymentConfigForCluster(ctx, kclient, log, cluster)
	// update label
	if err != nil || adcForCluster == nil {
		removeClusterLabel(log, cluster)
	} else {
		applyClusterLabel(log, cluster, adcForCluster)
	}
	return adcForCluster, err
}

// SkipCluster checks if akodeploymentconfig controller should skip reconciling this cluster or not
func SkipCluster(cluster *clusterv1.Cluster) bool {
	// if condition.ready is false
	// and cluster is not being deleted
	// and cluster is not a bootstrap cluster, skip
	if conditions.IsFalse(cluster, clusterv1.ReadyCondition) &&
		cluster.DeletionTimestamp.IsZero() &&
		!IsBootStrapCluster() {
		return true
	}
	return false
}

// isDefaultADC check if akodeploymentconfig object is default one
func isDefaultADC(adcName string) bool {
	return adcName == akoov1alpha1.WorkloadClusterAkoDeploymentConfig
}

// defaultADCHasEmptySelector checks if default akodeploymentconfig has empty selector
func defaultADCHasEmptySelector(ctx context.Context, kclient client.Client) bool {
	var defaultAdc akoov1alpha1.AKODeploymentConfig
	if err := kclient.Get(ctx, client.ObjectKey{
		Name: akoov1alpha1.WorkloadClusterAkoDeploymentConfig},
		&defaultAdc); err != nil {
		return false
	}
	selector, err := metav1.LabelSelectorAsSelector(&defaultAdc.Spec.ClusterSelector)
	if err != nil {
		return false
	}
	return selector.Empty()
}

// getAKODeploymentConfigForCluster return the akodeloymentconfig object which selects
// current cluster
func getAKODeploymentConfigForCluster(
	ctx context.Context,
	kclient client.Client,
	log logr.Logger,
	cluster *clusterv1.Cluster) (*akoov1alpha1.AKODeploymentConfig, error) {
	// list all the akodeploymentconfig objects
	var akoDeploymentConfigs akoov1alpha1.AKODeploymentConfigList
	if err := kclient.List(ctx, &akoDeploymentConfigs, []client.ListOption{}...); err != nil {
		log.Error(err, "Failed to list all AKODeploymentConfig objects")
		return nil, err
	}
	// find which adc matches current cluster
	var defaultAdc akoov1alpha1.AKODeploymentConfig
	for _, akoDeploymentConfig := range akoDeploymentConfigs.Items {
		if selector, err := metav1.LabelSelectorAsSelector(&akoDeploymentConfig.Spec.ClusterSelector); err != nil {
			log.Error(err, "Failed to convert label sector to selector")
		} else if selector.Empty() {
			if akoDeploymentConfig.Name == akoov1alpha1.WorkloadClusterAkoDeploymentConfig {
				defaultAdc = akoDeploymentConfig
			}
		} else if selector.Matches(labels.Set(cluster.GetLabels())) {
			log.Info("cluster is selected by akodeploymentconfig", "adc", akoDeploymentConfig.Name)
			return &akoDeploymentConfig, nil
		}
	}
	// only default adc with empty selector can select all clusters and return
	if defaultAdc.Name == akoov1alpha1.WorkloadClusterAkoDeploymentConfig {
		log.Info("cluster is selected by akodeploymentconfig", "adc", defaultAdc.Name)
		return &defaultAdc, nil
	}
	log.Info("cluster is not selected by any akodeploymentconfig objects")
	return nil, nil
}

// applyClusterLabel applies the networking.tkg.tanzu.vmware.com/avi label to a Cluster
func applyClusterLabel(log logr.Logger, cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig) {
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	log.Info("Adding label to cluster", "label", akoov1alpha1.AviClusterLabel, "adc", obj.Name)
	cluster.Labels[akoov1alpha1.AviClusterLabel] = obj.Name
}

// removeClusterLabel removes the networking.tkg.tanzu.vmware.com/avi label from a Cluster
func removeClusterLabel(log logr.Logger, cluster *clusterv1.Cluster) {
	if _, exists := cluster.Labels[akoov1alpha1.AviClusterLabel]; exists {
		log.Info("Removing label from cluster", "label", akoov1alpha1.AviClusterLabel)
	}
	delete(cluster.Labels, akoov1alpha1.AviClusterLabel)
}
