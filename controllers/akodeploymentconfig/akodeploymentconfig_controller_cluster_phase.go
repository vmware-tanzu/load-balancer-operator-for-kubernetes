// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig

import (
	"context"

	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/cluster"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/phases"
	controllerruntime "gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime"

	"github.com/go-logr/logr"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
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
			r.applyClusterLabel,
			r.addClusterFinalizer,
			r.ClusterReconciler.ReconcileAddonSecret,
		},
		[]phases.ReconcileClusterPhase{
			r.ClusterReconciler.ReconcileAddonSecretDelete,
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
			r.removeClusterLabel,
			r.removeClusterFinalizer,
			r.ClusterReconciler.ReconcileAddonSecretDelete,
		},
		[]phases.ReconcileClusterPhase{
			r.ClusterReconciler.ReconcileAddonSecretDelete,
			r.ClusterReconciler.ReconcileDelete,
		},
	)
}

// applyClusterLabel is a reconcileClusterPhase. It applies the AVI label to a
// Cluster
func (r *AKODeploymentConfigReconciler) applyClusterLabel(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	if _, exists := cluster.Labels[akoov1alpha1.AviClusterLabel]; !exists {
		log.Info("Adding label to cluster", "label", akoov1alpha1.AviClusterLabel)
	} else {
		log.Info("Label already applied to cluster", "label", akoov1alpha1.AviClusterLabel)

	}
	// Always set avi label on managed cluster
	cluster.Labels[akoov1alpha1.AviClusterLabel] = ""
	return ctrl.Result{}, nil
}

// removeClusterLabel is a reconcileClusterPhase. It removes the AVI label from a
// Cluster
func (r *AKODeploymentConfigReconciler) removeClusterLabel(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if _, exists := cluster.Labels[akoov1alpha1.AviClusterLabel]; exists {
		log.Info("Removing label from cluster", "label", akoov1alpha1.AviClusterLabel)
	}
	// Always deletes avi label on managed cluster
	delete(cluster.Labels, akoov1alpha1.AviClusterLabel)
	return ctrl.Result{}, nil
}

// addClusterFinalizer is a reconcileClusterPhase. It adds the AVI
// finalizer to a Cluster.
func (r *AKODeploymentConfigReconciler) addClusterFinalizer(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if !controllerruntime.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) {
		log.Info("Add finalizer to cluster", "finalizer", akoov1alpha1.ClusterFinalizer)
		ctrlutil.AddFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
	}
	return ctrl.Result{}, nil
}

// removeClusterFinalizer is a reconcileClusterPhase. It removes the AVI
// finalizer from a Cluster. This can only be called when the cluster is not in
// deletion state and AKODeploymentConfig is being deleted.
func (r *AKODeploymentConfigReconciler) removeClusterFinalizer(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if controllerruntime.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) {
		log.Info("Removing finalizer from cluster", "finalizer", akoov1alpha1.ClusterFinalizer)
	}
	ctrlutil.RemoveFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
	return ctrl.Result{}, nil
}
