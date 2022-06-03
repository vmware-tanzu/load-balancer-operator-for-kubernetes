// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	ako_operator "github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// AkoDeploymentConfigForCluster returns a handler map function for mapping Cluster
// resources to the AkoDeploymentConfig of this cluster
func AkoDeploymentConfigForCluster(c client.Client, log logr.Logger) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		ctx := context.Background()
		cluster, ok := o.(*clusterv1.Cluster)
		if !ok {
			log.Error(errors.New("invalid type"),
				"Expected to receive Cluster resource",
				"actualType", fmt.Sprintf("%T", o))
			return nil
		}
		logger := log.WithValues("cluster", cluster.Namespace+"/"+cluster.Name)
		if ako_operator.SkipCluster(cluster) {
			logger.Info("Skipping cluster in handler")
			return []reconcile.Request{}
		}

		adcForCluster, err := ako_operator.GetAKODeploymentConfigForCluster(ctx, c, logger, cluster)

		if err != nil || adcForCluster == nil {
			logger.V(3).Info("cluster is not selected by any ako deploymentconfig")
			removeClusterLabel(logger, cluster)
			return []reconcile.Request{}
		} else {
			logger.V(3).Info("cluster is selected by adc", "akodeploymentconfig", adcForCluster)
			applyClusterLabel(logger, cluster, adcForCluster)
			return []ctrl.Request{{NamespacedName: types.NamespacedName{Name: adcForCluster.Name}}}
		}
	}
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
