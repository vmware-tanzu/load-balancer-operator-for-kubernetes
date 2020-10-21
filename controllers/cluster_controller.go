// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ClusterReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	log := r.Log.WithValues("Cluster", req.NamespacedName)

	// Get the resource for this request.
	obj := &clusterv1alpha3.Cluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cluster not found, will not reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	// Handle deleted resources.
	if !obj.GetDeletionTimestamp().IsZero() {
		if err := r.reconcileDelete(ctx, log, obj); err != nil {
			log.Error(err, "failed to reconcile Cluster deletion")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Handle non-deleted resources.
	if err := r.reconcileNormal(ctx, log, obj); err != nil {
		log.Error(err, "failed to reconcile Cluster")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// We're not watching delete event for the workload cluster
func (r *ClusterReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1alpha3.Cluster,
) error {
	return nil
}

// SetupWithManager adds this reconciler to a new controller then to the
// provided manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch Cluster API v1alpha3 Cluster resources.
		For(&clusterv1alpha3.Cluster{}).
		Complete(r)
}

func (r *ClusterReconciler) reconcileNormal(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1alpha3.Cluster,
) error {
	r.Log.Info("Start reconciling", "cluster", obj.Name)
	var errs []error
	if err := r.reconcileCRS(ctx, log, obj); err != nil {
		errs = append(errs, err)
	}

	return kerrors.NewAggregate(errs)
}
