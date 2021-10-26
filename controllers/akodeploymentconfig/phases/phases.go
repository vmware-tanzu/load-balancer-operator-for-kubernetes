// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package phases

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	handlers "github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/controller-runtime/handlers"
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
	clusters, err := ListAkoDeplymentConfigDeployedClusters(ctx, client, obj)
	if err != nil {
		log.Error(err, "Fail to list clusters deployed by current AKODeploymentConfig")
		return res, err
	}

	if len(clusters.Items) == 0 {
		log.Info("No cluster matches the selector, skip")
	}

	var allErrs []error
	// For each cluster managed by the AKODeploymentConfig, run each phase
	// function
	for _, cluster := range clusters.Items {
		var errs []error

		clog := log.WithValues("cluster", cluster.Namespace+"/"+cluster.Name)

		// Always Patch for each cluster when exiting this function so changes to the resource are updated on the API server.
		patchHelper, err := patch.NewHelper(&cluster, client)
		if err != nil {
			return res, errors.Wrapf(err, "failed to init patch helper for %s %s",
				cluster.GroupVersionKind(), cluster.Namespace+"/"+cluster.Name)
		}

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

// ListAkoDeplymentConfigDeployedClusters list all clusters enabled current akodeploymentconfig
func ListAkoDeplymentConfigDeployedClusters(ctx context.Context, kclient client.Client, obj *akoov1alpha1.AKODeploymentConfig) (*clusterv1.ClusterList, error) {
	selector, err := metav1.LabelSelectorAsSelector(&obj.Spec.ClusterSelector)
	if err != nil {
		return nil, err
	}
	listOptions := []client.ListOption{
		client.MatchingLabelsSelector{Selector: selector},
	}
	var clusters clusterv1.ClusterList
	if err := kclient.List(ctx, &clusters, listOptions...); err != nil {
		return nil, err
	}

	var newItems []clusterv1.Cluster
	for _, c := range clusters.Items {
		if !handlers.SkipCluster(&c) {
			_, selected := c.Labels[akoov1alpha1.AviClusterSelectedLabel]
			// when cluster selected by non-default AKODeploymentConfig,
			// skip default select all AKODeploymentConfig
			if selector.Empty() && selected {
				continue
			}
			// management cluster can't be selected by other AKODeploymentConfig
			// instead of management cluster AKODeploymentConfig
			if c.Namespace == akoov1alpha1.TKGSystemNamespace &&
				obj.Name != akoov1alpha1.ManagementClusterAkoDeploymentConfig {
				continue
			}
			newItems = append(newItems, c)
		}
	}
	clusters.Items = newItems

	return &clusters, nil
}
