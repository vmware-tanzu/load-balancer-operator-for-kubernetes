// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package machine_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func intgTestMachineController() {
	var (
		ctx           *builder.IntegrationTestContext
		cluster       *clusterv1.Cluster
		staticCluster *clusterv1.Cluster
		staticMachine *clusterv1.Machine
		machine       *clusterv1.Machine
		testLabels    map[string]string
		err           error
	)

	staticCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: clusterv1.ClusterSpec{},
	}

	staticMachine = &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "test",
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name": "test",
			},
			Annotations: map[string]string{
				"pre-terminate.delete.hook.machine.cluster.x-k8s.io/avi-cleanup": "ako-operator",
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: "test",
		},
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

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
		staticCluster.Namespace = ctx.Namespace
		cluster = staticCluster.DeepCopy()
		machine = staticMachine.DeepCopy()
		testLabels = map[string]string{
			"networking.tkg.tanzu.vmware.com/avi": "",
			"tkg.tanzu.vmware.com/cluster-name":   "test",
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
			createObjects(cluster, machine)
		})
		It("Delete one machine directly", func() {
			deleteObjects(machine)
			Eventually(func() bool {
				err := ctx.Client.Get(ctx.Context, client.ObjectKey{Name: "test-machine", Namespace: "test"}, &clusterv1.Machine{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
		AfterEach(func() {
			deleteObjects(cluster)
		})

	})
}
