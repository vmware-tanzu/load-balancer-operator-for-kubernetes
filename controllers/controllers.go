// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/cluster"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/configmap"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/machine"
	akoo "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupReconcilers(mgr ctrl.Manager) error {
	if err := (&machine.MachineReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Machine"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	if !akoo.IsBootStrapCluster() {
		// if the ako-operator manager is deployed on management cluster
		// it will reconcile against any AKODeploymentConfig
		if err := (&akodeploymentconfig.AKODeploymentConfigReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("AKODeploymentConfig"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			return err
		}
	} else {
		// else if the ako-operator is deployed on the boostrap cluster
		// we don't need to reconcile any AKODeploymentConfig,
		// but we still have to add the network to IPAM specified in AKO's configmap
		if err := (&configmap.ConfigMapReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ConfigMap"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			return err
		}
	}
	if err := (&cluster.ClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Cluster"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}
	return nil
}
