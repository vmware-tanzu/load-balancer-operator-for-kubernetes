// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SetupWithManager adds this reconciler to a new controller then to the
// provided manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch Cluster resources.
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

	res := ctrl.Result{}
	// Get the resource for this request.
	cluster := &clusterv1.Cluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cluster not found, will not reconcile")
			return res, nil
		}
		return res, err
	}

	log = log.WithValues("Cluster", cluster.Namespace+"/"+cluster.Name)

	if _, exist := cluster.Labels[akoov1alpha1.AviClusterLabel]; !exist {
		log.Info("Cluster doesn't have AVI enabled, skip Cluster reconciling")
		return res, nil
	}

	log.Info("Cluster has AVI enabled, start Cluster reconciling")

	// Getting all akodeploymentconfigs
	var akoDeploymentConfigs akoov1alpha1.AKODeploymentConfigList
	if err := r.Client.List(ctx, &akoDeploymentConfigs); err != nil {
		return res, err
	}

	// Matches current cluster with all the akoDeploymentConfigs
	clusterLabels := cluster.GetLabels()
	for _, akoDeploymentConfig := range akoDeploymentConfigs.Items {
		if selector, err := metav1.LabelSelectorAsSelector(&akoDeploymentConfig.Spec.ClusterSelector); err != nil {
			log.Error(err, "Failed to convert label sector to selector when matching ", cluster.Name, " with ", akoDeploymentConfig.Name)
		} else if selector.Matches(labels.Set(clusterLabels)) {
			log.Info("Cluster ", cluster.Name, " is selected by Akodeploymentconfig ", akoDeploymentConfig.Namespace+"/"+akoDeploymentConfig.Name, ", return")
			return res, nil
		}
	}

	// Removing finalizer if current cluster can't be selected by any akoDeploymentConfig
	log.Info("Removing finalizer", "finalizer", akoov1alpha1.ClusterFinalizer)
	ctrlutil.RemoveFinalizer(cluster, akoov1alpha1.ClusterFinalizer)

	// Removing pre-terminate hook if current cluster can't be selected by any akoDeploymentConfig
	if _, err := r.deleteHook(ctx, log, cluster); err != nil {
		log.Error(err, "Failed to remove pre-terminate hook", cluster.Name)
		return res, err
	}

	return res, nil
}

func (r *ClusterReconciler) deleteHook(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
) (ctrl.Result, error) {
	log.Info("Start deleting pre-terminate hooks from all the machines of Cluster", cluster.Name)

	res := ctrl.Result{}
	listOptions := []client.ListOption{
		client.MatchingLabels(map[string]string{clusterv1.ClusterLabelName: cluster.Name}),
	}

	// List machines of current cluster
	var machines clusterv1.MachineList
	if err := r.Client.List(ctx, &machines, listOptions...); err != nil {
		return res, err
	}

	// Removing the pre-terminate hook from each machine
	for _, machine := range machines.Items {
		if annotations.HasWithPrefix(clusterv1.PreTerminateDeleteHookAnnotationPrefix, machine.ObjectMeta.Annotations) {
			delete(machine.Annotations, akoov1alpha1.PreTerminateAnnotation)
			log.Info("Removing pre-terminate hook")
		}
	}

	return res, nil
}
