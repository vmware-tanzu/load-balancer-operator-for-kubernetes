// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster_bootstrap_test

import (
	. "github.com/onsi/ginkgo"
	runv1alpha3 "github.com/vmware-tanzu/tanzu-framework/apis/run/v1alpha3"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/controllers/akodeploymentconfig/cluster"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	testutil "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/util"
)

func bootstrapTest() {

	var (
		ctx                              *builder.IntegrationTestContext
		staticCluster                    *clusterv1.Cluster
		staticDefaultAkoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
		staticClusterBootstrap           *runv1alpha3.ClusterBootstrap
	)

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
		staticCluster = testutil.GetDefaultCluster()
		staticDefaultAkoDeploymentConfig = testutil.GetDefaultADC()
		staticClusterBootstrap = testutil.GetDefaultCB(staticCluster)
	})

	When("Cluster Bootstrap exists", func() {

		BeforeEach(func() {
			testutil.CreateObjects(ctx, staticDefaultAkoDeploymentConfig.DeepCopy())
			testutil.CreateObjects(ctx, staticCluster.DeepCopy())
			testutil.CreateObjects(ctx, staticClusterBootstrap.DeepCopy())
		})

		It("Tests packages", func() {
			By("making sure AKO is in CB's additional packages", func() {
				testutil.EnsureClusterBootstrapPackagesMatchExpectation(ctx, client.ObjectKey{
					Name:      staticCluster.Name,
					Namespace: staticCluster.Namespace,
				}, cluster.AkoPackageName, true)
			})

		})
	})
}
