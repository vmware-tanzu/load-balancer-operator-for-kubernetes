// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig

import (
	"context"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	bootstrap_cluster "gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/bootstrap-cluster"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/phases"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *AKODeploymentConfigReconciler) initBootstrapCluster(log logr.Logger) {
	// Lazily initialize bootstrap cluster Reconciler
	if r.BootstrapClusterReconciler == nil {
		r.BootstrapClusterReconciler = bootstrap_cluster.NewReconciler(r.Client, r.Log, r.Scheme)
		log.Info("Bootstrap Cluster reconciler initialized")
	}
}

// reconcileBootstrapCluster reconciles bootstrap cluster akodeploymentconfig
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileBootstrapCluster(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	r.initBootstrapCluster(log)

	return phases.ReconcilePhases(ctx, log, obj,
		[]phases.ReconcilePhase{
			r.BootstrapClusterReconciler.DeployAKO,
			r.BootstrapClusterReconciler.DeployAKOSecret,
		},
	)
}

// reconcileBootstrapClusterDelete reconciles bootstrap cluster deletion
// It's a reconcilePhase function
func (r *AKODeploymentConfigReconciler) reconcileBootstrapClusterDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	r.initBootstrapCluster(log)

	return phases.ReconcilePhases(ctx, log, obj,
		[]phases.ReconcilePhase{
			r.BootstrapClusterReconciler.DeleteAKO,
		},
	)
}
