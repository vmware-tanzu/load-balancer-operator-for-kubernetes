// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package funcs

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	networkv1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig"
	adccluster "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/cluster"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/cluster"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"

	akov1alpha1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
)

var AddAllToSchemeFunc builder.AddToSchemeFunc = func(scheme *runtime.Scheme) (err error) {
	err = networkv1alpha1.AddToScheme(scheme)
	if err != nil {
		return err
	}
	err = akoov1alpha1.AddToScheme(scheme)
	if err != nil {
		return err
	}
	err = akov1alpha1.AddToScheme(scheme)
	if err != nil {
		return err
	}
	err = corev1.AddToScheme(scheme)
	if err != nil {
		return err
	}
	err = clusterv1.AddToScheme(scheme)
	if err != nil {
		return err
	}

	return nil
}

var AddAKODeploymentConfigAndClusterControllerToMgrFunc builder.AddToManagerFunc = func(mgr ctrlmgr.Manager) error {
	rec := &akodeploymentconfig.AKODeploymentConfigReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AKODeploymentConfig"),
		Scheme: mgr.GetScheme(),
	}

	builder.FakeAvi = aviclient.NewFakeAviClient()
	rec.SetAviClient(builder.FakeAvi)

	adcClusterReconciler := adccluster.NewReconciler(rec.Client, rec.Log, rec.Scheme)
	adcClusterReconciler.GetRemoteClient = adccluster.GetFakeRemoteClient
	rec.ClusterReconciler = adcClusterReconciler

	if err := rec.SetupWithManager(mgr); err != nil {
		return err
	}

	// involve the cluster controller as well for the resetting skip-default-adc label test
	if err := (&cluster.ClusterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Cluster"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
