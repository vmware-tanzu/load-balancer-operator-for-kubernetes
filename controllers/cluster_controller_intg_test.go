// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

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
		ctx           *builder.IntegrationTestContext
		cluster       *clusterv1.Cluster
		staticCluster *clusterv1.Cluster
		err           error
	)

	staticCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				akoov1alpha1.AviClusterLabel: "",
			},
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

	checkClusterFinalizerExitence := func(key client.ObjectKey, finalizer string, checkExistence bool) {
		Eventually(func() bool {
			obj := &clusterv1.Cluster{}
			err := ctx.Client.Get(ctx.Context, key, obj)
			if err != nil {
				return false
			}
			if checkExistence {
				return controllerruntime.ContainsFinalizer(obj, finalizer)
			} else {
				return !controllerruntime.ContainsFinalizer(obj, finalizer)
			}
		}).Should(BeTrue())
	}

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
		staticCluster.Namespace = ctx.Namespace
	})
	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
		staticCluster.Namespace = ""
	})

	When("Non AVI Cluster is created", func() {
		BeforeEach(func() {
			cluster = staticCluster.DeepCopy()
			delete(cluster.Labels, akoov1alpha1.AviClusterLabel)
			createObjects(cluster)
		})
		AfterEach(func() {
			deleteObjects(cluster)
		})
		It("Should not trigger the reconcile", func() {
			checkClusterFinalizerExitence(client.ObjectKey{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}, akoov1alpha1.ClusterFinalizer, false)
		})
	})
	When("AVI Cluster is created", func() {
		BeforeEach(func() {
			cluster = staticCluster.DeepCopy()
			createObjects(cluster)
		})
		AfterEach(func() {
			deleteObjects(cluster)
		})
		It("Should have the finalizer", func() {
			checkClusterFinalizerExitence(client.ObjectKey{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}, akoov1alpha1.ClusterFinalizer, true)
		})
	})
}
