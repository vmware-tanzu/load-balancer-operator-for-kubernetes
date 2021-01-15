// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/cluster"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/machine"
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
	if err := (&akodeploymentconfig.AKODeploymentConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AKODeploymentConfig"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
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
