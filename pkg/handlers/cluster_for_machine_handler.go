// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ako_operator "github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MachinesForCluster returns a handler map function for mapping Cluster
// resources to the Machines of this cluster
func MachinesForCluster(c client.Client, log logr.Logger) handler.MapFunc {

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

		listOptions := []client.ListOption{
			client.InNamespace(cluster.Namespace),
			client.MatchingLabels(map[string]string{clusterv1.ClusterLabelName: cluster.Name}),
		}

		log.V(3).Info("Start listing machines for cluster", "cluster", cluster.Namespace+"/"+cluster.Name)

		var machines clusterv1.MachineList
		if err := c.List(ctx, &machines, listOptions...); err != nil {
			return []reconcile.Request{}
		}

		log.V(3).Info("Finished listing machines for cluster", "cluster", cluster.Namespace+"/"+cluster.Name, "machines-count", len(machines.Items))

		// Create a reconcile request for each machine resource.
		requests := []ctrl.Request{}
		for _, machine := range machines.Items {
			requests = append(requests, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: machine.Namespace,
					Name:      machine.Name,
				},
			})
		}
		log.V(3).Info("Generating requests", "requests", requests)
		// Return reconcile requests for the Machine resources.
		return requests
	}
}
