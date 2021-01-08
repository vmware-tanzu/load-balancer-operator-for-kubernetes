// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
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

const (
	requeueAfterForAKODeletion = time.Second * 1
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
		} else {
			log.Info("AKO deletion is in progress, requeue", "after", requeueAfterForAKODeletion.String())
			return ctrl.Result{Requeue: true, RequeueAfter: requeueAfterForAKODeletion}, nil
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

	var cleanupFinished bool

	// We then retrieve the AKO ConfigMap from the workload cluster and
	// update the `deleteConfig` field to trigger AKO's cleanup
	akoConfigMap := &corev1.ConfigMap{}
	akoConfigMapKey := client.ObjectKey{
		Name:      akoov1alpha1.AkoConfigMapName,
		Namespace: akoov1alpha1.AviNamespace,
	}
	err = remoteClient.Get(ctx, akoConfigMapKey, akoConfigMap)
	if err == nil {
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
	} else {
		if apierrors.IsNotFound(err) {
			log.Info("Cannot find AKO ConfigMap, consider the cleanup finished", "configMap", akoConfigMapKey.Namespace+"/"+akoConfigMapKey.Name)
			cleanupFinished = true
		} else {
			return false, err
		}
	}

	if !cleanupFinished {
		cleanupFinished, err = ako.CleanupFinished(ctx, remoteClient, log)
		if err != nil {
			log.Error(err, "Failed to retrieve AKO cleanup status")
			return false, err
		}
	}
	if cleanupFinished {
		log.Info("AKO finished cleanup, updating Cluster condition")
		conditions.MarkTrue(obj, akoov1alpha1.AviResourceCleanupSucceededCondition)
		return true, nil
	}

	return false, nil
}
