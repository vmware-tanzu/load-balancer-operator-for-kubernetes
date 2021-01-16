// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"

	testenv "gitlab.eng.vmware.com/core-build/ako-operator/e2e/pkg/env"
)

var _ = Describe("AKODeploymentConfig with selector", func() {
	var (
		testName = "AKODeploymentConfig with selector"
		testcase *testenv.E2ETestCase
		skip     bool
	)

	BeforeEach(func() {
		By("checking if test should be skipped")
		skip, testcase = testenv.LoadTestCase(testName)
		if skip {
			Skip(fmt.Sprintf("Skip test case :[%s]", testName))
		} else {
			GinkgoT().Logf("Running test case: [%s]", testName)
		}
		By("ensuring AKODeploymentConfig is applied")
		testcase.EnsureYamlsApplied([]string{
			testcase.AKODeploymentConfig.Path,
		})
		By("running sanity checks")
		testcase.SanityCheck()
		By("creating test case specific namespace")
		testcase.Init()
	})

	AfterEach(func() {
		if skip {
			Skip(fmt.Sprintf("Skip cleaning up test case :[%s]", testName))
		} else {
			GinkgoT().Logf("Cleaning up test case: [%s]", testName)
		}
		By("ensuring AKODeploymentConfig is deleted")
		testcase.EnsureYamlsRemoved([]string{
			testcase.AKODeploymentConfig.Path,
		})
		By("tearing down test case specific namespace")
		testcase.Teardown()
	})

	When("one cluster is newly created without the specified label", func() {
		var (
			clusterName string
		)

		BeforeEach(func() {
			By("creating a TKG Workload Cluster")
			clusterName = testenv.GenerateRandomName()
			testcase.EnsureClusterCreated(clusterName)
		})

		AfterEach(func() {
			By("deleting a TKG Workload Cluster")
			testcase.EnsureClusterDeleted(clusterName)
		})

		When("it's later applied with the specified label", func() {
			BeforeEach(func() {
				By("ensuring cluster has the correct labels applied")
				testcase.EnsureClusterLabelApplied(
					clusterName, testenv.AKODeploymentConfigLabelsGetter(testcase),
				)
			})
			It("should be managed by AKO Operator", func() {
				By("ensuring AKO is successfully installed in the workload cluster")
				testenv.EnsureAKO(testcase, clusterName)
				By("ensuring Load Balancer Type SVC works as expected")
				testenv.EnsureLoadBalancerService(testcase, clusterName)
			})
		})
	})
})
