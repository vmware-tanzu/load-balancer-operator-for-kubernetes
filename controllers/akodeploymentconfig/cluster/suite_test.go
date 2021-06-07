// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	"k8s.io/apimachinery/pkg/runtime"

	ctrlmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

// suite is used for unit and integration testing this controller.
var suite = builder.NewTestSuiteForController(
	func(mgr ctrlmgr.Manager) error {
		return nil
	},
	func(scheme *runtime.Scheme) (err error) {
		return nil
	},
)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Operator AKODeploymentConfig controller Cluster reconciler", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func intgTests() {
}

func unitTests() {
	Describe("AKO Deployment Spec generation", unitTestAKODeploymentYaml)
}
