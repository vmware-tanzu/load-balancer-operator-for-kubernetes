// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	ako_operator "gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako-operator"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	testutil "gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func intgTestEnsureClusterHAProvider() {

	Context("EnsureHAService", func() {
		var (
			ctx           *builder.IntegrationTestContext
			cluster       *clusterv1.Cluster
			staticCluster *clusterv1.Cluster
			serviceName   string
			testNamespace *corev1.Namespace
		)

		staticCluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ha-cluster",
				Namespace: akoov1alpha1.TKGSystemNamespace,
			},
			Spec: clusterv1.ClusterSpec{},
		}

		BeforeEach(func() {
			ctx = suite.NewIntegrationTestContext()
			cluster = staticCluster.DeepCopy()
			serviceName = cluster.Namespace + "-" + cluster.Name + "-" + akoov1alpha1.HAServiceName
		})
		AfterEach(func() {
			ctx.AfterEach()
			ctx = nil
		})

		When("Avi is not HA provider", func() {
			BeforeEach(func() {
				err := os.Setenv(ako_operator.IsControlPlaneHAProvider, "False")
				Expect(err).ShouldNot(HaveOccurred())
				testutil.CreateObjects(ctx, cluster)
			})
			AfterEach(func() {
				testutil.DeleteObjects(ctx, cluster)
			})
			It("should not create service or endpoint", func() {
				testutil.EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{
					Name:      serviceName,
					Namespace: akoov1alpha1.TKGSystemNamespace,
				}, &corev1.Service{}, testutil.NOTFOUND)
				testutil.EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{
					Name:      serviceName,
					Namespace: akoov1alpha1.TKGSystemNamespace,
				}, &corev1.Endpoints{}, testutil.NOTFOUND)
			})
		})

		When("Avi is HA provider", func() {
			When("HA service and endpoint not exist", func() {
				BeforeEach(func() {
					err := os.Setenv(ako_operator.IsControlPlaneHAProvider, "True")
					Expect(err).ShouldNot(HaveOccurred())
					testNamespace = &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: akoov1alpha1.TKGSystemNamespace,
						}}
					testutil.CreateObjects(ctx, testNamespace, cluster)
					// add an ip to service since ako is absent
					service := &corev1.Service{}
					testutil.EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{
						Name:      serviceName,
						Namespace: akoov1alpha1.TKGSystemNamespace,
					}, &corev1.Service{}, testutil.EXIST)

					err = ctx.Client.Get(ctx, client.ObjectKey{Name: serviceName, Namespace: akoov1alpha1.TKGSystemNamespace}, service)
					Expect(err).ShouldNot(HaveOccurred())

					service.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{
						IP:       "10.0.0.1",
						Hostname: "intg-test",
					}}
					err = ctx.Client.Status().Update(ctx, service)
					Expect(err).To(BeNil())
				})

				AfterEach(func() {
					testutil.DeleteObjects(ctx, cluster, testNamespace)
				})
				It("should create service and endpoint", func() {
					testutil.EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{
						Name:      serviceName,
						Namespace: akoov1alpha1.TKGSystemNamespace,
					}, &corev1.Endpoints{}, testutil.EXIST)

				})
			})
		})
	})
}
