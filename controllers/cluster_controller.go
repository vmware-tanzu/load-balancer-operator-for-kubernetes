// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	controllerruntime "gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/remote"
	capiutil "sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
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

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=clusterresourcesets;clusterresourcesets/status,verbs=get;list;watch;create;update;patch;delete

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
		patchOpts := []patch.Option{}
		if reterr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		}

		if err := patchHelper.Patch(ctx, obj, patchOpts...); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, err})
			if reterr != nil {
				log.Error(err, "patch failed")
			}
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

		finished, err := r.cleanup(ctx, log, obj)
		if err != nil {
			log.Error(err, "Error cleaning up")
			return err
		}

		// The resources are deleted so remove the finalizer.
		if finished {
			log.Info("Removing finalizer", "finalizer", akoov1alpha1.ClusterFinalizer)
			ctrlutil.RemoveFinalizer(obj, akoov1alpha1.ClusterFinalizer)
		}
	}

	return nil
}

func (r *ClusterReconciler) cleanup(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) (bool, error) {
	// Firstly we check if there is a cleanup condition in the Cluster
	// status , if not, we update it
	if conditions.Get(obj, akoov1alpha1.AviResourceCleanupSucceededCondition) == nil {
		conditions.MarkFalse(obj, akoov1alpha1.AviResourceCleanupSucceededCondition, akoov1alpha1.AviResourceCleanupReason, clusterv1.ConditionSeverityInfo, "Cleaning up the AVI load balancing resources before deletion")
		log.Info("Trigger the AKO cleanup in the target Cluster and set Cluster condition", "condition", akoov1alpha1.AviResourceCleanupSucceededCondition)
	}

	remoteClient, err := remote.NewClusterClient(ctx, r.Client, client.ObjectKey{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}, r.Scheme)
	if err != nil {
		log.Info("Failed to create remote client for cluster, requeue the request")
		return false, err
	}

	// We then retrieve the AKO ConfigMap from the workload cluster and
	// update the `deleteConfig` field to trigger AKO's cleanup
	akoConfigMap := &corev1.ConfigMap{}
	akoConfigMapKey := client.ObjectKey{
		Name:      "avi-k8s-config",
		Namespace: "avi-system",
	}
	if err := remoteClient.Get(ctx, akoConfigMapKey, akoConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cannot find AKO ConfigMap, requeue the request", "configMap", "avi-system/test")
		}
		return false, err
	}
	cleanupSetting := akoConfigMap.Data["deleteConfig"]
	log.Info("Found AKO Configmap", "deleteConfig", cleanupSetting)

	// true is the only accepted string in AKO
	if cleanupSetting != "true" {
		log.Info("Updating deleteConfig in AKO's ConfigMap")
		akoConfigMap.Data["deleteConfig"] = "true"

		if err := remoteClient.Update(ctx, akoConfigMap); err != nil {
			log.Info("Failed to update AKO ConfigMap, requeue the request")
			return false, err
		}
	}

	// TODO(fangyuanl): use the real finalizer value by importing from AKO's
	// repo
	if !controllerruntime.ContainsFinalizer(obj, "finalizer-placeholder") {
		log.Info("AKO finished cleanup, updating Cluster condition")
		conditions.MarkTrue(obj, akoov1alpha1.AviResourceCleanupSucceededCondition)
		return true, nil
	}

	return false, nil
}

func (r *ClusterReconciler) reconcileNormal(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) (ctrl.Result, error) {
	log.Info("Start reconciling")

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
