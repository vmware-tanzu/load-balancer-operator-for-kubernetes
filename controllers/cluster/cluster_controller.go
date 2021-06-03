// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"github.com/pkg/errors"
	ako_operator "gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako-operator"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime/handlers"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/haprovider"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
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
		Watches(
			&source.Kind{Type: &corev1.Service{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handlers.ClusterForService(r.Client, r.Log),
			}).
		Complete(r)
}

type ClusterReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Haprovider *haprovider.HAProvider
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

	// Always Patch when exiting this function so changes to the resource are updated on the API server.
	patchHelper, err := patch.NewHelper(cluster, r.Client)
	if err != nil {
		return res, errors.Wrapf(err, "failed to init patch helper for %s %s",
			cluster.GroupVersionKind(), req.NamespacedName)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, cluster); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "patch failed")
		}
	}()

	if ako_operator.IsHAProvider() {
		log.Info("AVI is control plane HA provider")
		r.Haprovider = haprovider.NewProvider(r.Client, r.Log)
		if err = r.Haprovider.CreateOrUpdateHAService(ctx, cluster); err != nil {
			log.Error(err, "Fail to reconcile HA service")
			return res, err
		}
		if ako_operator.IsBootStrapCluster() {
			return res, nil
		}
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

	// Removing add on secret and its associated resources for a AKO
	if _, err := r.deleteAddonSecret(ctx, log, cluster); err != nil {
		log.Error(err, "Failed to remove secret", cluster.Name)
		return res, err
	}

	// Removing avi label after deleting all the resources
	delete(cluster.Labels, akoov1alpha1.AviClusterLabel)

	return res, nil
}

// deleteAddonSecret delete cluster related add on secret
func (r *ClusterReconciler) deleteAddonSecret(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
) (ctrl.Result, error) {
	log.Info("Starts reconciling add on secret deletion")
	res := ctrl.Result{}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      r.akoAddonSecretName(cluster),
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(3).Info("add on secret is already deleted")
			return res, nil
		}
		log.Error(err, "Failed to get add on secret, requeue")
		return res, err
	}

	if err := r.Delete(ctx, secret); err != nil {
		log.Error(err, "Failed to delete add on secret, requeue")
		return res, err
	}
	return res, nil
}

func (r *ClusterReconciler) akoAddonSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-addon"
}
