// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package configmap

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/netprovider"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SetupWithManager adds this reconciler to a new controller then to the
// provided manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Watch ConfigMap resources.
		For(&corev1.ConfigMap{}).
		Complete(r)
}

// ConfigMapReconciler reads the data network from avi-k8s-config ConfigMap and
// accordingly adds it as a usable network via the AVI Controller client.
type ConfigMapReconciler struct {
	client.Client
	aviClient aviclient.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	netprovider.UsableNetworkProvider
}

func (r *ConfigMapReconciler) SetAviClient(client aviclient.Client) {
	r.aviClient = client
}

var InvalidAKOConfigMapErr = errors.New("Invalid format of AKO ConfigMap")

func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := r.Log.WithValues("ConfigMap", req.NamespacedName)

	// Get the resource for this request.
	cm := &corev1.ConfigMap{}
	if err := r.Client.Get(ctx, req.NamespacedName, cm); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ConfigMap not found, will not reconcile")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if cm.Name != v1alpha1.AkoConfigMapName || cm.Namespace != v1alpha1.TKGSystemNamespace {
		return ctrl.Result{}, nil
	}

	log.Info("Start reconciling AVI cloud usable network in bootstrap cluster")

	cloudName, exist := cm.Data[v1alpha1.AkoConfigMapCloudNameKey]
	if !exist {
		log.Info("Key not found in ConfigMap: cloudName")
		return ctrl.Result{}, InvalidAKOConfigMapErr
	}
	vipNetworkListRaw, exist := cm.Data[v1alpha1.AkoConfigMapVipNetworkListKey]
	if !exist {
		log.Info("Key not found in ConfigMap: vipNetworkList")
		return ctrl.Result{}, InvalidAKOConfigMapErr
	}

	var vipNetworkList netprovider.UsableNetworks
	if err := json.Unmarshal([]byte(vipNetworkListRaw), &vipNetworkList); err != nil {
		log.Error(err, "Failed to unmarshal VIPNetworkList")
		return ctrl.Result{}, err
	}

	log.V(5).Info(fmt.Sprintf("ConfigMap %s found in %s", v1alpha1.AkoConfigMapName, v1alpha1.TKGSystemNamespace))

	for _, vipNetwork := range vipNetworkList {
		added, err := r.AddUsableNetwork(r.aviClient, cloudName, vipNetwork.NetworkName)
		if err != nil {
			log.Error(err, "Failed to add usable network", vipNetwork.NetworkName)
			return ctrl.Result{}, err
		}
		if added {
			log.Info("Added Usable Network", vipNetwork.NetworkName)
		} else {
			log.Info("Network is already one of the cloud's usable network", vipNetwork.NetworkName)
		}
	}
	return ctrl.Result{}, nil
}
