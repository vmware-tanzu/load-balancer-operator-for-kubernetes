// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes/scheme"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/ako"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
		Client:          c,
		Log:             log,
		Scheme:          scheme,
		GetRemoteClient: remote.NewClusterClient,
	}
}

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create;list;watch
// +kubebuilder:rbac:groups=addons.cluster.x-k8s.io,resources=clusterresourcesets;clusterresourcesets/status,verbs=get;list;watch;create;update;patch;delete

type ClusterReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	GetRemoteClient remote.ClusterClientGetter
}

// ReconcileDelete removes the finalizer on Cluster once AKO finishes its
// cleanup work
func (r *ClusterReconciler) ReconcileDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	_ *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	if ctrlutil.ContainsFinalizer(cluster, akoov1alpha1.ClusterFinalizer) {
		log.Info("Handling deleted Cluster")

		finished, err := r.cleanup(ctx, log, cluster)
		if err != nil {
			log.Error(err, "Error cleaning up")
			return res, err
		}
		// remove finalizer only after avi resources deleted && avi user deleted
		finished = finished && conditions.IsTrue(cluster, akoov1alpha1.AviUserCleanupSucceededCondition)

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
	if !conditions.Has(obj, akoov1alpha1.AviResourceCleanupSucceededCondition) {
		conditions.MarkFalse(obj, akoov1alpha1.AviResourceCleanupSucceededCondition, akoov1alpha1.AviResourceCleanupReason, clusterv1.ConditionSeverityInfo, "Cleaning up the AVI load balancing resources before deletion")
		log.Info("Trigger the AKO cleanup in the target Cluster and set Cluster condition", "condition", akoov1alpha1.AviResourceCleanupSucceededCondition)
	} else if conditions.IsTrue(obj, akoov1alpha1.AviResourceCleanupSucceededCondition) {
		log.Info("Avi resource cleanup is finished")
		return true, nil
	}

	remoteClient, err := r.GetRemoteClient(ctx, akoov1alpha1.AKODeploymentConfigControllerName, r.Client, client.ObjectKey{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	})
	if err != nil {
		log.Info("Failed to create remote client for cluster, requeue the request")
		return false, err
	}

	akoAddonSecret := &corev1.Secret{}
	if err := remoteClient.Get(ctx, client.ObjectKey{
		Name:      r.akoAddonDataValueName(),
		Namespace: akoov1alpha1.TKGSystemNamespace,
	}, akoAddonSecret); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		log.Error(err, "Failed to get AKO Addon Data Values, AKO clean up failed")
		return false, err
	}

	akoAddonSecretData := akoAddonSecret.Data["values.yaml"]

	values, err := ako.NewValuesFromBytes(akoAddonSecretData)
	if err != nil {
		log.Error(err, "Failed to unmarshal values from ako add-on data")
		return false, err
	}

	if values.LoadBalancerAndIngressService.Config.AKOSettings.DeleteConfig != "true" {
		values.LoadBalancerAndIngressService.Config.AKOSettings.DeleteConfig = "true"
		secretData, err := values.YttYaml()
		if err != nil {
			return false, errors.Errorf("workload cluster %s ako add-on data values marshal error", obj.Name)
		}
		akoAddonSecret.Data["values.yaml"] = []byte(secretData)
		if err := remoteClient.Update(ctx, akoAddonSecret); err != nil {
			log.Error(err, "Failed to update AKO Addon Data Values, AKO clean up failed")
			return false, err
		}
		log.Info("Updated `deleteConfig` field to true in AKO Addon Data Values, starting ako clean up")
	}

	cleanupFinished, err := ako.CleanupFinished(ctx, remoteClient, log)
	if err != nil {
		log.Error(err, "Failed to retrieve AKO cleanup status")
		return false, err
	}

	if cleanupFinished {
		log.Info("AKO finished cleanup, updating Cluster condition")
		conditions.MarkTrue(obj, akoov1alpha1.AviResourceCleanupSucceededCondition)
		return true, nil
	}
	return false, nil
}

func GetFakeRemoteClient(_ context.Context, _ string, _ client.Client, _ client.ObjectKey) (client.Client, error) {
	// return fake client
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(), nil
}

func (r *ClusterReconciler) akoAddonDataValueName() string {
	return "load-balancer-and-ingress-service-data-values"
}
