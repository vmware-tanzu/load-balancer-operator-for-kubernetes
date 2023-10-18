// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/haprovider"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.serviceToCluster(r.Client, r.Log)),
		).
		Complete(r)
}

type ClusterReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Haprovider *haprovider.HAProvider
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
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

	log = log.WithValues("Cluster", cluster.Namespace+"/"+cluster.Name)

	isVIPProvider, err := ako_operator.IsControlPlaneVIPProvider(cluster)
	if err != nil {
		log.Error(err, "can't unmarshal cluster variables")
		return res, err
	}

	if isVIPProvider {
		log.Info("AVI is control plane HA provider")
		r.Haprovider = haprovider.NewProvider(r.Client, r.Log)
		if err = r.Haprovider.CreateOrUpdateHAService(ctx, cluster); err != nil {
			log.Error(err, "Fail to reconcile HA service")
			return res, err
		}
	}

	// skip reconcile if cluster is using kube-vip to provide load balancer service
	if isLBProvider, err := ako_operator.IsLoadBalancerProvider(cluster); err != nil {
		log.Error(err, "can't unmarshal cluster variables")
		return res, err
	} else if !isLBProvider {
		log.Info("cluster uses kube-vip to provide load balancer type of service, skip reconciling")
		return res, nil
	}

	akoDeploymentConfig, err := ako_operator.GetAKODeploymentConfigForCluster(ctx, r.Client, log, cluster)
	if err != nil {
		log.Error(err, "failed to get cluster matched akodeploymentconfig")
		return res, err
	}

	// Removing finalizer and avi label if current cluster can't be selected by any akoDeploymentConfig
	if akoDeploymentConfig == nil {
		log.Info("Not find cluster matched akodeploymentconfig, skip Cluster reconciling, removing finalizer and avi labels", "finalizer", akoov1alpha1.ClusterFinalizer)
		ako_operator.RemoveClusterLabel(log, cluster)
		ctrlutil.RemoveFinalizer(cluster, akoov1alpha1.ClusterFinalizer)
	} else {
		log.Info("cluster has AVI enabled", "akodeploymentconfig", akoDeploymentConfig)
		ako_operator.ApplyClusterLabel(log, cluster, akoDeploymentConfig)
	}

	return res, nil
}

// serviceToCluster returns a handler map function for mapping Service
// resources to the cluster
func (r *ClusterReconciler) serviceToCluster(c client.Client, log logr.Logger) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		service, ok := o.(*corev1.Service)
		if !ok {
			log.Error(errors.New("invalid type"),
				"Expected to receive service resource",
				"actualType", fmt.Sprintf("%T", o))
			return nil
		}
		logger := log.WithValues("service", service.Namespace+"/"+service.Name)
		if r.skipService(service) {
			return []reconcile.Request{}
		}
		// in bootstrap kind cluster, ensure ako deletion before delete service
		if ako_operator.IsBootStrapCluster() && !service.DeletionTimestamp.IsZero() {
			if err := r.deleteAKOStatefulSet(ctx, c, v1alpha1.AkoStatefulSetName, v1alpha1.TKGSystemNamespace); err != nil {
				log.Error(err, "Fail to delete AKO statefulset before service in bootstrap cluster")
			}
			return []reconcile.Request{}
		}
		var cluster clusterv1.Cluster
		if err := c.Get(ctx, client.ObjectKey{
			Name:      service.Annotations[v1alpha1.TKGClusterNameLabel],
			Namespace: service.Annotations[v1alpha1.TKGClusterNameSpaceLabel],
		}, &cluster); err != nil {
			return []reconcile.Request{}
		}
		// Create a reconcile request for cluster resource.
		requests := []ctrl.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: cluster.Namespace,
				Name:      cluster.Name,
			}}}
		logger.V(3).Info("Generating requests", "requests", requests)
		// Return reconcile requests for the cluster resources.
		return requests
	}
}

func (r *ClusterReconciler) skipService(service *corev1.Service) bool {
	return service.Spec.Type != corev1.ServiceTypeLoadBalancer || !strings.Contains(service.Name, v1alpha1.HAServiceName)
}

// deleteAKOStatefulSet deletes the stateful set with specified name and namespace
func (r *ClusterReconciler) deleteAKOStatefulSet(ctx context.Context, c client.Client, name string, namespace string) error {
	akoStatefulSet := &v1.StatefulSet{}
	if err := c.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace},
		akoStatefulSet); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}
	return c.Delete(ctx, akoStatefulSet)
}
