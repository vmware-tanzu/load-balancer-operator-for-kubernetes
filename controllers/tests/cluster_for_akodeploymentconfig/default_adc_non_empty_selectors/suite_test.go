// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package default_adc_non_empty_selectors

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

func TestControllerWithNonEmptyDefaultADC(t *testing.T) {
	suite.Register(t, "AKO Operator Controllers with non-empty selector install-ako-for-all", intgTests, unitTests)
}

func intgTests() {
	Describe("Cluster selected by default ADC with non-empty selectors", intgTestCanSelectedByDefaultADCWithNonEmptySelectors)
}

func unitTests() {
}
