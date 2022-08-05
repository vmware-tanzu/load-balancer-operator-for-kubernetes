// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/cluster"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/phases"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
)

func (r *AKODeploymentConfigReconciler) initCluster(log logr.Logger) {
	// Lazily initialize clusterReconciler
	if r.ClusterReconciler == nil {
		r.ClusterReconciler = cluster.NewReconciler(r.Client, r.Log, r.Scheme)
		log.Info("Cluster reconciler initialized")
	}
}

// reconcileClusters reconciles every cluster that matches the
// AKODeploymentConfig's selector
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileClusters(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	r.initCluster(log)

	return phases.ReconcileClustersPhases(ctx, r.Client, log, obj,
		[]phases.ReconcileClusterPhase{
			r.addClusterFinalizer,
			r.ClusterReconciler.ReconcileAddonSecret,
			r.ClusterReconciler.ReconcileClusterBootstrap,
		},
		[]phases.ReconcileClusterPhase{
			r.ClusterReconciler.ReconcileAddonSecretDelete,
			r.ClusterReconciler.ReconcileClusterBootstrapDelete,
			r.ClusterReconciler.ReconcileDelete,
		},
	)
}

// reconcileClustersDelete reconciles every cluster that matches the
// AKODeploymentConfig's selector when a AKODeploymentConfig is being deleted
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileClustersDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	r.initCluster(log)

	return phases.ReconcileClustersPhases(ctx, r.Client, log, obj,
		// When AKODeploymentConfig is being deleted and the target
		// cluster is in normal state, remove the label and finalizer to
		// stop managing it
		[]phases.ReconcileClusterPhase{
			r.removeClusterFinalizer,
			r.ClusterReconciler.ReconcileAddonSecretDelete,
		},
		[]phases.ReconcileClusterPhase{
			r.ClusterReconciler.ReconcileAddonSecretDelete,
			r.ClusterReconciler.ReconcileDelete,
		},
	)
}

// addClusterFinalizer is a reconcileClusterPhase. It adds the AVI
// finalizer to a Cluster.
func (r *AKODeploymentConfigReconciler) addClusterFinalizer(
	_ context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	_ *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if !ctrlutil.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) &&
		cluster.Namespace != akoov1alpha1.TKGSystemNamespace {
		log.Info("Add finalizer to cluster", "finalizer", akoov1alpha1.ClusterFinalizer)
		ctrlutil.AddFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
	}
	return ctrl.Result{}, nil
}

// removeClusterFinalizer is a reconcileClusterPhase. It removes the AVI
// finalizer from a Cluster. This can only be called when the cluster is not in
// deletion state and AKODeploymentConfig is being deleted.
func (r *AKODeploymentConfigReconciler) removeClusterFinalizer(
	_ context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	_ *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if ctrlutil.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) {
		log.Info("Removing finalizer from cluster", "finalizer", akoov1alpha1.ClusterFinalizer)
	}
	ctrlutil.RemoveFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
	return ctrl.Result{}, nil
}
