// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	akoControllers "gitlab.eng.vmware.com/core-build/ako-operator/controllers"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clustereaddonv1alpha3 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func intgTestCRS() {
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

	checkCRSExitence := func(key client.ObjectKey, checkExistence bool) {
		Eventually(func() bool {
			s := &v1.Secret{}
			err_s := ctx.Client.Get(ctx.Context, key, s)

			crs := &clustereaddonv1alpha3.ClusterResourceSet{}
			err_c := ctx.Client.Get(ctx.Context, key, crs)

			if err_s == nil && err_c == nil { // both secret and crs exist
				return checkExistence
			} else if err_s != nil && err_c != nil {
				if apierrors.IsNotFound(err_s) && apierrors.IsNotFound(err_c) { // neither secret nor crs exists
					return !checkExistence
				}
			}
			return false

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
		It("Should not have Secret and CRS", func() {
			checkCRSExitence(client.ObjectKey{
				Name:      akoControllers.CrsWorkloadClusterResourceName,
				Namespace: cluster.Namespace,
			}, false)
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
		It("Should have the Secret and CRS", func() {
			checkCRSExitence(client.ObjectKey{
				Name:      akoControllers.CrsWorkloadClusterResourceName,
				Namespace: cluster.Namespace,
			}, true)
		})
	})
}
