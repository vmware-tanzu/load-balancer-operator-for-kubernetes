// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	controllerruntime "gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func intgTestMachineDeletionHook() {
	var (
		ctx                 *builder.IntegrationTestContext
		cluster             *clusterv1.Cluster
		staticCluster       *clusterv1.Cluster
		akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
		testLabels          map[string]string
		err                 error
	)

	staticCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: clusterv1.ClusterSpec{},
	}
	createObjects := func(objs ...runtime.Object) {
		for _, o := range objs {
			err = ctx.Client.Create(ctx.Context, o)
			Expect(err).To(BeNil())
		}
	}
	deleteObjects := func(objs ...runtime.Object) {
		for _, o := range objs {
			// ignore error
			_ = ctx.Client.Delete(ctx.Context, o)
		}
	}

	ensureClusterReconciliationMatchExpectation := func(key client.ObjectKey, expectReconciled bool) {
		Eventually(func() bool {
			obj := &clusterv1.Cluster{}
			err := ctx.Client.Get(ctx.Context, key, obj)
			if err != nil {
				return false
			}
			finalizer := akoov1alpha1.ClusterFinalizer
			clusterLabel := akoov1alpha1.AviClusterLabel
			if expectReconciled {
				if !controllerruntime.ContainsFinalizer(obj, finalizer) {
					return false
				}
				if _, exist := obj.Labels[clusterLabel]; !exist {
					return false
				}
			} else {
				if controllerruntime.ContainsFinalizer(obj, finalizer) {
					return false
				}
				if _, exist := obj.Labels[clusterLabel]; exist {
					return false
				}
			}
			return true
		}).Should(BeTrue())
	}

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
		staticCluster.Namespace = ctx.Namespace
		cluster = staticCluster.DeepCopy()
		akoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: akoov1alpha1.AKODeploymentConfigSpec{
				DataNetwork: akoov1alpha1.DataNetwork{
					Name: "test",
					CIDR: "10.0.0.0/24",
					IPPools: []akoov1alpha1.IPPool{
						akoov1alpha1.IPPool{
							Start: "10.0.0.1",
							End:   "10.0.0.10",
							Type:  "V4",
						},
					},
				},
			},
		}
		testLabels = map[string]string{
			"test": "true",
		}
	})
	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
		staticCluster.Namespace = ""
	})

	When("A Cluster is created", func() {
		JustBeforeEach(func() {
			cluster.Labels = testLabels
			createObjects(cluster, akoDeploymentConfig)
		})
		AfterEach(func() {
			deleteObjects(cluster, akoDeploymentConfig)
		})
		When("there is no matching AKODeploymentConfig", func() {
			BeforeEach(func() {
				akoDeploymentConfig.Labels = testLabels
			})
			It("should not reconcile the cluster", func() {
				ensureClusterReconciliationMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, false)
			})
		})
		When("there is a matching AKODeploymentConfig", func() {
			It("should reconcile the cluster", func() {
				ensureClusterReconciliationMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, true)
			})
		})
	})
}
