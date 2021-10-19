// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type akoDeploymentConfigForCluster struct {
	client.Client
	log logr.Logger
}

func (r *akoDeploymentConfigForCluster) AkoDeploymentConfigForCluster(o client.Object) []reconcile.Request {
	ctx := context.Background()

	cluster, ok := o.(*clusterv1.Cluster)
	if !ok {
		r.log.Error(errors.New("invalid type"),
			"Expected to receive Cluster resource",
			"actualType", fmt.Sprintf("%T", o.(client.Object)))
		return nil
	}

	logger := r.log.WithValues("cluster", cluster.Namespace+"/"+cluster.Name)

	if SkipCluster(cluster) {
		logger.Info("Skipping cluster in handler")
		return []reconcile.Request{}
	}
	// is cluster selected by non-default akodeploymentconfig
	_, selected := cluster.Labels[akoov1alpha1.AviClusterSelectedLabel]

	logger.V(3).Info("Getting all akodeploymentconfig")
	var akoDeploymentConfigs akoov1alpha1.AKODeploymentConfigList
	if err := r.Client.List(ctx, &akoDeploymentConfigs, []client.ListOption{}...); err != nil {
		return []reconcile.Request{}
	}
	// Create a reconcile request for every label matched akodeploymentconfig
	requests := []ctrl.Request{}
	for _, akoDeploymentConfig := range akoDeploymentConfigs.Items {
		if selector, err := metav1.LabelSelectorAsSelector(&akoDeploymentConfig.Spec.ClusterSelector); err != nil {
			logger.Error(err, "Failed to convert label sector to selector")
			continue
		} else if selector.Empty() && selected {
			logger.V(3).Info("Cluster selected by non-default AKODeploymentConfig, skip default one")
			continue
		} else if selector.Matches(labels.Set(cluster.GetLabels())) {
			logger.V(3).Info("Found matching AKODeploymentConfig", akoDeploymentConfig.Namespace+"/"+akoDeploymentConfig.Name)
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: akoDeploymentConfig.Namespace,
					Name:      akoDeploymentConfig.Name,
				},
			})
		}
	}
	logger.V(3).Info("Generating requests", "requests", requests)

	// Return reconcile requests for the AKODeploymentConfig resources.
	return requests
}

// AkoDeploymentConfigForClusterMapFunc returns a handler.Mapper for mapping Cluster
// resources to the AkoDeploymentConfig of this cluster
func AkoDeploymentConfigForClusterMapFunc(c client.Client, log logr.Logger) handler.MapFunc {
	akoDeploymentConfigForClusterMapper := &akoDeploymentConfigForCluster{
		Client: c,
		log:    log,
	}
	return akoDeploymentConfigForClusterMapper.AkoDeploymentConfigForCluster
}
