// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"

	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/funcs"
	testutil "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/util"
)

// suite is used for unit and integration testing this controller.
var suite = builder.NewTestSuiteForController(
	funcs.AddAKODeploymentConfigAndClusterControllerToMgrFunc,
	funcs.AddAllToSchemeFunc,
	filepath.Join(testutil.FindModuleDir("sigs.k8s.io/cluster-api"), "config", "crd", "bases"),
	filepath.Join(testutil.FindModuleDir("github.com/vmware/load-balancer-and-ingress-services-for-kubernetes"), "helm", "ako", "crds"),
)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Operator", intgTests, unitTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func intgTests() {
	Describe("AkoDeploymentConfigController Test", intgTestAkoDeploymentConfigController)
}

func unitTests() {
	Describe("Ensure static ranges Test", unitTestEnsureStaticRanges)
}
