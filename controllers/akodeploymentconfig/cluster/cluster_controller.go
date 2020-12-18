// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/phases"
	corev1 "k8s.io/api/core/v1"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako"
	controllerruntime "gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewReconciler initializes a ClusterReconciler
func NewReconciler(c client.Client, log logr.Logger, scheme *runtime.Scheme) *ClusterReconciler {
	return &ClusterReconciler{
		Client: c,
		Log:    log,
		Scheme: scheme,
	}
}

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create;list;watch
// +kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=clusterresourcesets;clusterresourcesets/status,verbs=get;list;watch;create;update;patch;delete

type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// ReconcileDelete removes the finalizer on Cluster once AKO finishes its
// cleanup work
func (r *ClusterReconciler) ReconcileDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	if controllerruntime.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) {
		log.Info("Handling deleted Cluster")

		finished, err := r.cleanup(ctx, log, cluster)
		if err != nil {
			log.Error(err, "Error cleaning up")
			return res, err
		}

		// The resources are deleted so remove the finalizer.
		if finished {
			log.Info("Removing finalizer", "finalizer", akoov1alpha1.ClusterFinalizer)
			ctrlutil.RemoveFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
		}
	}

	return res, nil
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

	if finished, err := ako.CleanupFinished(ctx, remoteClient, log); err != nil {
		log.Error(err, "Failed to retrieve AKO cleanup status")
		return false, err
	} else if finished {
		log.Info("AKO finished cleanup, updating Cluster condition")
		conditions.MarkTrue(obj, akoov1alpha1.AviResourceCleanupSucceededCondition)
		return true, nil
	}

	return false, nil
}

func (r *ClusterReconciler) ReconcileNormal(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Start reconciling")

	res := ctrl.Result{}
	if _, exist := cluster.Labels[akoov1alpha1.AviClusterLabel]; !exist {
		log.Info("cluster doesn't have AVI enabled, skip reconciling")
		return res, nil
	}

	if !controllerruntime.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) {
		log.Info("Add finalizer", "finalizer", akoov1alpha1.ClusterFinalizer)
		// The finalizer must be present before proceeding in order to ensure that any IPs allocated
		// are released when the interface is destroyed. Return immediately after here to let the
		// patcher helper update the object, and then proceed on the next reconciliation.
		ctrlutil.AddFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
	}

	return phases.ReconcileClustersPhases(ctx, r.Client, log, obj, []phases.ReconcileClusterPhase{
		r.reconcileCRS,
	})
}
