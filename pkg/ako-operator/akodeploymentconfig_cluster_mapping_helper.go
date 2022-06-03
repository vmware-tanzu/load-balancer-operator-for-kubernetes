// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListAkoDeplymentConfigSelectClusters list all clusters enabled current akodeploymentconfig
func ListAkoDeplymentConfigSelectClusters(ctx context.Context, kclient client.Client, obj *akoov1alpha1.AKODeploymentConfig) (*clusterv1.ClusterList, error) {
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
	var newItems []clusterv1.Cluster
	for _, c := range clusters.Items {
		if !SkipCluster(&c) {
			adcName, selected := c.Labels[akoov1alpha1.AviClusterLabel]
			if !selected {
				continue
			}
			// if cluster previously is selected by default adc then it can be override
			// no matter default adc has selector or not, it has lower priority than customer
			// newly created adc
			if adcName != obj.Name && adcName != akoov1alpha1.WorkloadClusterAkoDeploymentConfig {
				continue
			}
			// management cluster can't be selected by other AKODeploymentConfig
			// instead of management cluster AKODeploymentConfig
			if c.Namespace == akoov1alpha1.TKGSystemNamespace &&
				obj.Name != akoov1alpha1.ManagementClusterAkoDeploymentConfig {
				continue
			}
			newItems = append(newItems, c)
		}
	}
	clusters.Items = newItems
	return &clusters, nil
}

func GetAKODeploymentConfigForCluster(ctx context.Context, kclient client.Client, log logr.Logger, cluster *clusterv1.Cluster) (adc *akoov1alpha1.AKODeploymentConfig, err error) {
	var akoDeploymentConfigs akoov1alpha1.AKODeploymentConfigList
	if err := kclient.List(ctx, &akoDeploymentConfigs, []client.ListOption{}...); err != nil {
		log.Error(err, "Failed to list all AKODeploymentConfig objects")
		return nil, err
	}
	for _, akoDeploymentConfig := range akoDeploymentConfigs.Items {
		if selector, err := metav1.LabelSelectorAsSelector(&akoDeploymentConfig.Spec.ClusterSelector); err != nil {
			log.Error(err, "Failed to convert label sector to selector")
		} else if selector.Empty() {
			// TODO:(xudongl) add webhook to provent customer to create adc without selector
			if akoDeploymentConfig.Name == akoov1alpha1.WorkloadClusterAkoDeploymentConfig {
				log.Info("this is default ako deployment config, it can select all non-selected clusters")
				adc = &akoDeploymentConfig
			} else {
				err = errors.New("non default AKODeploymentConfig cluster selector must not be empty")
				log.Error(err, "selector must not be empty")
				return nil, err
			}
		} else if selector.Matches(labels.Set(cluster.GetLabels())) {
			log.Info("Found matching AKODeploymentConfig", "adc", akoDeploymentConfig.Namespace+"/"+akoDeploymentConfig.Name)
			return &akoDeploymentConfig, nil
		}
	}
	return adc, nil
}

func UpdateClusterSelectedADCInfo(ctx context.Context, kclient client.Client, log logr.Logger, cluster *clusterv1.Cluster) (adc *akoov1alpha1.AKODeploymentConfig, err error) {
	adcForCluster, err := GetAKODeploymentConfigForCluster(ctx, kclient, log, cluster)
	if err != nil || adcForCluster == nil {
		removeClusterLabel(log, cluster)
	} else {
		applyClusterLabel(log, cluster, adcForCluster)
	}
	err = kclient.Update(ctx, cluster)
	return adcForCluster, err
}

func SkipCluster(cluster *clusterv1.Cluster) bool {
	// if condition.ready is false and cluster is not being deleted and not bootstrap cluster, skip
	if conditions.IsFalse(cluster, clusterv1.ReadyCondition) &&
		cluster.DeletionTimestamp.IsZero() &&
		!IsBootStrapCluster() {
		println("skip cluster")
		return true
	}
	return false
}

// applyClusterLabel is a reconcileClusterPhase. It applies the AVI label to a Cluster
func applyClusterLabel(log logr.Logger, cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig) {
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	if _, exists := cluster.Labels[akoov1alpha1.AviClusterLabel]; !exists {
		log.Info("Adding label to cluster", "label", akoov1alpha1.AviClusterLabel)
	} else {
		log.Info("Label already applied to cluster", "label", akoov1alpha1.AviClusterLabel)
	}
	// Always set avi label on managed cluster
	cluster.Labels[akoov1alpha1.AviClusterLabel] = obj.Name
}

// removeClusterLabel is a reconcileClusterPhase. It removes the AVI label from a Cluster
func removeClusterLabel(log logr.Logger, cluster *clusterv1.Cluster) {
	if _, exists := cluster.Labels[akoov1alpha1.AviClusterLabel]; exists {
		log.Info("Removing label from cluster", "label", akoov1alpha1.AviClusterLabel)
	}
	// Always deletes avi label on managed cluster
	delete(cluster.Labels, akoov1alpha1.AviClusterLabel)
}
