// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package default_adc_non_empty_selectors

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	testutil "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/util"
)

func intgTestCanSelectedByDefaultADCWithNonEmptySelectors() {
	var (
		ctx *builder.IntegrationTestContext

		labels                           map[string]string
		staticCluster                    *clusterv1.Cluster
		staticAkoDeploymentConfig        *akoov1alpha1.AKODeploymentConfig
		staticDefaultAkoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
	)

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()

		labels = map[string]string{
			"foo": "bar",
		}
		staticCluster = testutil.GetDefaultCluster()
		staticAkoDeploymentConfig = testutil.GetCustomizedADC(labels)
		staticDefaultAkoDeploymentConfig = testutil.GetDefaultADC()
	})

	When("install-ako-for-all has non-empty cluster selector", func() {
		BeforeEach(func() {
			staticDefaultAkoDeploymentConfig.Spec.ClusterSelector.MatchLabels = labels
			testutil.CreateObjects(ctx, staticDefaultAkoDeploymentConfig.DeepCopy())
		})

		// default adc with non-empty selector ->
		// create a cluster selected by default adc ->
		// create a new adc also can select cluster -> expect cluster should not change the adc
		It("labels the cluster", func() {
			By("create a cluster selected by default ADC")
			staticCluster.Labels = labels
			testutil.CreateObjects(ctx, staticCluster.DeepCopy())

			By("labels with 'networking.tkg.tanzu.vmware.com/avi: install-ako-for-all'", func() {
				testutil.EnsureClusterAviLabelMatchExpectation(ctx, client.ObjectKey{
					Name:      staticCluster.Name,
					Namespace: staticCluster.Namespace,
				}, akoov1alpha1.AviClusterLabel, staticDefaultAkoDeploymentConfig.Name)
			})

			By("create another ADC with same selector")
			testutil.CreateObjects(ctx, staticAkoDeploymentConfig.DeepCopy())

			By("cluster should keep its labels, since the default ADC cannot be override if it has non-empty selector", func() {
				Consistently(func() bool {
					obj := &clusterv1.Cluster{}
					err := ctx.Client.Get(ctx.Context, client.ObjectKey{
						Name:      staticCluster.Name,
						Namespace: staticCluster.Namespace,
					}, obj)
					if err != nil {
						return false
					}
					val, ok := obj.Labels[akoov1alpha1.AviClusterLabel]
					return ok && val == staticDefaultAkoDeploymentConfig.Name
				})
			})
		})
	})
}
