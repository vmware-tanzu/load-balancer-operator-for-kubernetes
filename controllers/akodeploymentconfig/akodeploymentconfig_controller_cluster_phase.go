// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig

import (
	"context"

	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/cluster"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/phases"

	"github.com/go-logr/logr"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"

	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
)

func (r *AKODeploymentConfigReconciler) initCluster(log logr.Logger) {
	// Lazily initialize clusterReconciler
	if r.clusterReconciler == nil {
		r.clusterReconciler = cluster.NewReconciler(r.Client, r.Log, r.Scheme)
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

	return phases.ReconcileClustersPhases(ctx, r.Client, log, obj, []phases.ReconcileClusterPhase{
		r.applyClusterLabel,
		r.clusterReconciler.ReconcileNormal,
	})
}

// reconcileClustersDelete reconciles every cluster that matches the
// AKODeploymentConfig's selector
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileClustersDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	r.initCluster(log)

	return phases.ReconcileClustersPhases(ctx, r.Client, log, obj, []phases.ReconcileClusterPhase{
		r.clusterReconciler.ReconcileDelete,
	})
}

// applyClusterLabel is a reconcileClusterPhase. It applies the AVI label to a
// Cluster
func (r *AKODeploymentConfigReconciler) applyClusterLabel(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	// Always set avi label on managed cluster
	cluster.Labels[akoov1alpha1.AviClusterLabel] = ""
	return ctrl.Result{}, nil
}
