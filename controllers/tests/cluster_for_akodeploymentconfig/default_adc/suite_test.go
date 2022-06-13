// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package default_adc_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/builder"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/funcs"
	testutil "github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/util"
)

// suite is used for testing the interactions between the controllers
var suite = builder.NewTestSuiteForController(
	funcs.AddAKODeploymentConfigAndClusterControllerToMgrFunc,
	funcs.AddAllToSchemeFunc,
	filepath.Join(testutil.FindModuleDir("sigs.k8s.io/cluster-api"), "config", "crd", "bases"),
	filepath.Join(testutil.FindModuleDir("github.com/vmware/load-balancer-and-ingress-services-for-kubernetes"), "helm", "ako", "crds"),
)

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

func TestController(t *testing.T) {
	suite.Register(t, "AKO Operator Controllers", intgTests, unitTests)
}

func intgTests() {
	Describe("Cluster not selected", intgTestClusterDisableAVIWithoutAnyADC)
	Describe("Cluster selected by ADC", intgTestClusterCanBeSelectedByADC)
}

func unitTests() {
}
