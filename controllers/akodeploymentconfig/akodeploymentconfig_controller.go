// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig

import (
	"context"

	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/cluster"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/phases"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/user"

	controllerruntime "gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	akoov1alpha1 "gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/controller-runtime/handlers"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (r *AKODeploymentConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&akoov1alpha1.AKODeploymentConfig{}).
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(handlers.AkoDeploymentConfigForCluster(r.Client, r.Log)),
		).
		Complete(r)
}

type AKODeploymentConfigReconciler struct {
	client.Client
	aviClient         aviclient.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	userReconciler    *user.AkoUserReconciler
	ClusterReconciler *cluster.ClusterReconciler
}

func (r *AKODeploymentConfigReconciler) SetAviClient(client aviclient.Client) {
	r.aviClient = client
}

// AKODeploymentConfigReconciler reconciles a AKODeploymentConfig object

// +kubebuilder:rbac:groups=networking.tkg.tanzu.vmware.com,resources=akodeploymentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.tkg.tanzu.vmware.com,resources=akodeploymentconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create;list;watch;update;delete

func (r *AKODeploymentConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("AKODeploymentConfig", req.NamespacedName)
	res := ctrl.Result{}
	var err error

	// Get the resource for this request.
	obj := &akoov1alpha1.AKODeploymentConfig{}
	if err = r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AKODeploymentConfig not found, will not reconcile")
			return res, nil
		}
		return res, err
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

	// Handle deleted cluster resources.
	if !obj.GetDeletionTimestamp().IsZero() {
		res, err := r.reconcileDelete(ctx, log, obj)
		if err != nil {
			log.Error(err, "failed to reconcile AKODeploymentConfig deletion")
			return res, err
		}
		return res, nil
	}

	// Handle non-deleted resources.
	res, err = r.reconcileNormal(ctx, log, obj)
	if err != nil {
		log.Error(err, "failed to reconcile AKODeploymentConfig")
		return res, err
	}
	return res, nil
}

func (r *AKODeploymentConfigReconciler) reconcileNormal(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if !controllerruntime.ContainsFinalizer(obj, akoov1alpha1.AkoDeploymentConfigFinalizer) {
		log.Info("Add finalizer", "finalizer", akoov1alpha1.AkoDeploymentConfigFinalizer)
		// The finalizer must be present before proceeding in order to ensure that all avi user account
		// resources are released when the interface is destroyed. Return immediately after here to let the
		// patcher helper update the object, and then proceed on the next reconciliation.
		ctrlutil.AddFinalizer(obj, akoov1alpha1.AkoDeploymentConfigFinalizer)
	}
	return phases.ReconcilePhases(ctx, log, obj,
		[]phases.ReconcilePhase{r.reconcileAVI, r.reconcileClusters})
}

func (r *AKODeploymentConfigReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (res ctrl.Result, reterr error) {
	// Directly return if there is no finalizer
	if !controllerruntime.ContainsFinalizer(obj, akoov1alpha1.AkoDeploymentConfigFinalizer) {
		return res, nil
	}

	log.Info("AkoDeploymentConfig is being deleted. Start cleaning up")

	defer func() {
		if reterr == nil && !res.Requeue {
			// remove finalizer when clean up finishes successfully
			log.Info("Removing finalizer", "finalizer", akoov1alpha1.AkoDeploymentConfigFinalizer)
			ctrlutil.RemoveFinalizer(obj, akoov1alpha1.AkoDeploymentConfigFinalizer)
		}
	}()
	return phases.ReconcilePhases(ctx, log, obj,
		[]phases.ReconcilePhase{r.reconcileClustersDelete, r.reconcileAVIDelete})
}
