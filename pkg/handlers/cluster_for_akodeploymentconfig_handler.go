// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
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
		// get akodeploymentconfig object for this cluster
		adcForCluster, err := ako_operator.GetAKODeploymentConfigForCluster(ctx, c, logger, cluster)
		if err != nil {
			logger.Error(err, "failed to get cluster matched akodeploymentconfig object")
			return []reconcile.Request{}
		}

		requests := []reconcile.Request{}
		if adcForCluster == nil {
			logger.Info("cluster is not selected by any akodeploymentconfig, skip")
		} else {
			logger.Info("cluster is selected by akodeploymentconfig", "akodeploymentconfig", adcForCluster)
			requests = append(requests, ctrl.Request{NamespacedName: types.NamespacedName{Name: adcForCluster.Name}})
		}
		return requests
	}
}
