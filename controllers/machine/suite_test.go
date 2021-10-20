// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package machine_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/controllers/machine"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/builder"
	testutil "gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clustereaddonv1alpha4 "sigs.k8s.io/cluster-api/exp/addons/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

// suite is used for unit and integration testing this controller.
var suite = builder.NewTestSuiteForController(
	func(mgr ctrlmgr.Manager) error {

		builder.FakeAvi = aviclient.NewFakeAviClient()

		if err := (&machine.MachineReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("Machine"),
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
		err = clustereaddonv1alpha4.AddToScheme(scheme)
		if err != nil {
			return err
		}
		return nil
	},
	filepath.Join(testutil.FindModuleDir("sigs.k8s.io/cluster-api"), "config", "crd", "bases"),
)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Operator Machine Controller", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func intgTests() {
	Describe("MachineController Test", intgTestMachineController)
}

func unitTests() {
}
