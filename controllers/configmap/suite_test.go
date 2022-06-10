// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package configmap_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/configmap"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/builder"
	testutil "github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/util"
)

// suite is used for unit and integration testing this controller.
var suite = builder.NewTestSuiteForController(
	func(mgr ctrlmgr.Manager) error {
		builder.FakeAvi = aviclient.NewFakeAviClient()
		rec := &configmap.ConfigMapReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("ConfigMap"),
			Scheme: mgr.GetScheme(),
		}
		rec.SetAviClient(builder.FakeAvi)
		if err := rec.SetupWithManager(mgr); err != nil {
			return err
		}

		// always create a namespace tkg-system for testing
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "tkg-system"},
		}
		err := rec.Client.Create(context.Background(), namespace)
		return err
	},
	func(scheme *runtime.Scheme) (err error) {
		err = corev1.AddToScheme(scheme)
		if err != nil {
			return err
		}
		err = clusterv1.AddToScheme(scheme)
		if err != nil {
			return err
		}
		err = akoov1alpha1.AddToScheme(scheme)
		return err
	},
	filepath.Join(testutil.FindModuleDir("sigs.k8s.io/cluster-api"), "config", "crd", "bases"),
)

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Operator Cluster Controller", intgTests, unitTests)
}

func intgTests() {
	Describe("ClusterController Test", intgTestEnsureUsableNetworkAddedInBootstrapCluster)
}

func unitTests() {
}
