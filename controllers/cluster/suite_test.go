// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"path/filepath"
	"testing"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/cluster"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	testutil "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

// suite is used for unit and integration testing this controller.
var suite = builder.NewTestSuiteForController(
	func(mgr ctrlmgr.Manager) error {

		builder.FakeAvi = aviclient.NewFakeAviClient()

		if err := (&cluster.ClusterReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Cluster"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			return err
		}
		return nil
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
		if err != nil {
			return err
		}
		return nil
	},
	filepath.Join(testutil.FindModuleDir("sigs.k8s.io/cluster-api"), "config", "crd", "bases"),
)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Operator Cluster Controller", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func intgTests() {
	Describe("ClusterController Test", intgTestEnsureClusterHAProvider)
}

func unitTests() {
}
