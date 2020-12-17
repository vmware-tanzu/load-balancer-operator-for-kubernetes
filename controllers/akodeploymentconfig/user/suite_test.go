// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"path/filepath"
	"testing"

	networkv1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	testutil "gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clustereaddonv1alpha3 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha3"
	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	. "github.com/onsi/ginkgo"
)

// suite is used for unit and integration testing this controller.
var suite = builder.NewTestSuiteForReconciler(
	func(mgr ctrlmgr.Manager) error {
		return nil
	},
	func(scheme *runtime.Scheme) (err error) {
		err = networkv1alpha1.AddToScheme(scheme)
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
		err = clustereaddonv1alpha3.AddToScheme(scheme)
		if err != nil {
			return err
		}
		return nil
	},
	filepath.Join(testutil.FindModuleDir("sigs.k8s.io/cluster-api"), "config", "crd", "bases"),
)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Reconciler", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func intgTests() {
	Describe("ako user reconciler test", AkoUserReconcilerTest)
}

func unitTests() {
}
