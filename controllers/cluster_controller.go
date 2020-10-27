// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "gitlab.eng.vmware.com/fangyuanl/akoo/api/v1alpha1"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	controllerruntime "gitlab.eng.vmware.com/fangyuanl/akoo/pkg/controller-runtime"
	"gitlab.eng.vmware.com/fangyuanl/akoo/pkg/controller-runtime/patch"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capiutil "sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SetupWithManager adds this reconciler to a new controller then to the
// provided manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch Cluster API v1alpha3 Cluster resources.
		For(&clusterv1.Cluster{}).
		Complete(r)
}

type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *ClusterReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	log := r.Log.WithValues("Cluster", req.NamespacedName)

	// Get the resource for this request.
	obj := &clusterv1.Cluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cluster not found, will not reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Always Patch when exiting this function so changes to the resource are updated on the API server.
	patchHelper, err := patch.NewHelper(obj, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to init patch helper for %s %s",
			obj.GroupVersionKind(), req.NamespacedName)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, obj); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "patch failed")
		}
	}()

	// Handle deleted resources.
	if !obj.GetDeletionTimestamp().IsZero() {
		if err := r.reconcileDelete(ctx, log, obj); err != nil {
			log.Error(err, "failed to reconcile Cluster deletion")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Handle non-deleted resources.
	if res, err := r.reconcileNormal(ctx, log, obj); err != nil {
		log.Error(err, "failed to reconcile Cluster")
		return res, err
	}
	return ctrl.Result{}, nil
}

// reconcileDelete removes the finalizer on Cluster once AKO finishes its
// cleanup work
func (r *ClusterReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) error {
	if controllerruntime.ContainsFinalizer(obj, akoov1alpha1.ClusterFinalizer) {
		log.Info("Handling deleted Cluster")

		if err := r.cleanup(ctx, log, obj); err != nil {
			log.Error(err, "Error cleaning up")
			return err
		}

		// The resources are deleted so remove the finalizer.
		// TODO(fangyuanl): comment out the removal of finalizer to help
		// with manual test. should be removed in future when cleanup is
		// implemented
		//ctrlutil.RemoveFinalizer(obj, akoov1alpha1.ClusterFinalizer)
		log.Info("Removing finalizer", "finalizer", akoov1alpha1.ClusterFinalizer)
	}

	return nil
}

func (r *ClusterReconciler) cleanup(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) error {
	// TODO(fangyuanl): add the logic to trigger the in cluster AKO self destruction
	//time.Sleep(time.Second * 3600)
	return nil
}

func (r *ClusterReconciler) reconcileNormal(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) (ctrl.Result, error) {
	log.V(1).Info("Start reconciling")

	res := ctrl.Result{}
	if _, exist := obj.Labels[akoov1alpha1.AviClusterLabel]; !exist {
		log.Info("cluster doesn't have AVI enabled, skip reconciling")
		return res, nil
	}

	if !controllerruntime.ContainsFinalizer(obj, akoov1alpha1.ClusterFinalizer) {
		log.Info("Add finalizer", "finalizer", akoov1alpha1.ClusterFinalizer)
		// The finalizer must be present before proceeding in order to ensure that any IPs allocated
		// are released when the interface is destroyed. Return immediately after here to let the
		// patcher helper update the object, and then proceed on the next reconciliation.
		ctrlutil.AddFinalizer(obj, akoov1alpha1.ClusterFinalizer)
	}

	reconcileFuns := []func(context.Context, logr.Logger, *clusterv1.Cluster) (ctrl.Result, error){
		r.reconcileCRS,
	}

	errs := []error{}
	for _, reconcileFunc := range reconcileFuns {
		// Call the inner reconciliation methods.
		reconcileResult, err := reconcileFunc(ctx, log, obj)
		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			continue
		}
		res = capiutil.LowestNonZeroResult(res, reconcileResult)
	}
	return res, kerrors.NewAggregate(errs)
}

// reconcileCRS creates the CRS for AKO deployment in workload clusters
func (r *ClusterReconciler) reconcileCRS(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
