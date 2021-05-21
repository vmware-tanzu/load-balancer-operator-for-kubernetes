// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"
	"strings"

	"gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type clusterForService struct {
	client.Client
	log logr.Logger
}

func (r *clusterForService) Map(o handler.MapObject) []reconcile.Request {
	ctx := context.Background()
	service, ok := o.Object.(*corev1.Service)
	if !ok {
		r.log.Error(errors.New("invalid type"),
			"Expected to receive service resource",
			"actualType", fmt.Sprintf("%T", o.Object))
		return nil
	}
	logger := r.log.WithValues("service", service.Namespace+"/"+service.Name)
	if SkipService(service) {
		return []reconcile.Request{}
	}
	var cluster clusterv1.Cluster
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      service.Annotations[v1alpha1.TKGClusterNameLabel],
		Namespace: service.Annotations[v1alpha1.TKGClusterNameSpaceLabel],
	}, &cluster); err != nil {
		return []reconcile.Request{}
	}
	// Create a reconcile request for cluster resource.
	requests := []ctrl.Request{ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: cluster.Namespace,
			Name:      cluster.Name,
		}}}
	logger.V(3).Info("Generating requests", "requests", requests)
	// Return reconcile requests for the cluster resources.
	return requests
}

// ClusterForService returns a handler.Mapper for mapping Service
// resources to the cluster
func ClusterForService(c client.Client, log logr.Logger) handler.Mapper {
	return &clusterForService{
		Client: c,
		log:    log,
	}
}

func SkipService(service *corev1.Service) bool {
	return service.Spec.Type != corev1.ServiceTypeLoadBalancer || !strings.Contains(service.Name, v1alpha1.HAServiceName)
}
