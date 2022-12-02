// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
)

// ReconcilePhase defines a function that reconciles one aspect of
// AKODeploymentConfig
type ReconcilePhase func(context.Context, logr.Logger, *akoov1alpha1.AKODeploymentConfig) (ctrl.Result, error)

// reconcilePhases runs each phase regardless of its error status.
// The aggregated error will be returned
func ReconcilePhases(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
	phases []ReconcilePhase,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	var errs []error
	for _, phase := range phases {
		// Call the inner reconciliation methods.
		phaseResult, err := phase(ctx, log, obj)
		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			continue
		}
		res = util.LowestNonZeroResult(res, phaseResult)
	}
	return res, kerrors.NewAggregate(errs)
}

// ReconcilePhase defines a per-cluster function that reconciles one aspect of
// AKODeploymentConfig
type ReconcileClusterPhase func(context.Context, logr.Logger, *clusterv1.Cluster, *akoov1alpha1.AKODeploymentConfig) (ctrl.Result, error)

// reconcileClusters reconcile every cluster that matches the
// AKODeploymentConfig's selector by running through an array of phases
func ReconcileClustersPhases(
	ctx context.Context,
	client client.Client,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
	normalPhases []ReconcileClusterPhase,
	deletePhases []ReconcileClusterPhase,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	// Get the list of clusters managed by the AKODeploymentConfig
	clusters, err := ako_operator.ListAkoDeploymentConfigSelectClusters(ctx, client, log, obj)
	if err != nil {
		log.Error(err, "Fail to list clusters deployed by current AKODeploymentConfig")
		return res, err
	}

	if len(clusters.Items) == 0 {
		log.Info("No cluster matches the selector, skip")
		return res, nil
	}

	var allErrs []error
	// For each cluster managed by the AKODeploymentConfig, run each phase
	// function
	for _, cluster := range clusters.Items {
		var errs []error

		clog := log.WithValues("cluster", cluster.Namespace+"/"+cluster.Name)

		// skip reconcile if cluster is using kube-vip to provide load balancer service
		if isLBProvider, err := ako_operator.IsLoadBalancerProvider(&cluster); err != nil {
			log.Error(err, "can't unmarshal cluster variables")
			return res, err
		} else if !isLBProvider {
			log.Info(fmt.Sprintf("cluster uses kube-vip to provide load balancer type of service, skip reconciling for cluster %s/%s", cluster.Namespace, cluster.Name))
			return res, nil
		}

		// Always Patch for each cluster when exiting this function so changes to the resource are updated on the API server.
		patchHelper, err := patch.NewHelper(&cluster, client)
		if err != nil {
			return res, errors.Wrapf(err, "failed to init patch helper for %s %s",
				cluster.GroupVersionKind(), cluster.Namespace+"/"+cluster.Name)
		}

		// update cluster avi label before run any phase functions
		ako_operator.ApplyClusterLabel(log, &cluster, obj)

		phases := normalPhases
		if !cluster.GetDeletionTimestamp().IsZero() {
			phases = deletePhases
		}
		for _, phase := range phases {
			// Call the inner reconciliation methods regardless of
			// the error status
			phaseResult, err := phase(ctx, clog, &cluster, obj)
			if err != nil {
				errs = append(errs, err)
			}
			if len(errs) > 0 {
				continue
			}
			res = util.LowestNonZeroResult(res, phaseResult)
		}

		clusterErr := kerrors.NewAggregate(errs)
		patchOpts := []patch.Option{}
		if clusterErr == nil {
			patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})
		} else {
			allErrs = append(allErrs, clusterErr)
		}

		if err := patchHelper.Patch(ctx, &cluster, patchOpts...); err != nil {
			clusterErr = kerrors.NewAggregate([]error{clusterErr, err})
			if clusterErr != nil {
				log.Error(clusterErr, "patch failed")
			}
		}
	}

	return res, kerrors.NewAggregate(allErrs)
}
