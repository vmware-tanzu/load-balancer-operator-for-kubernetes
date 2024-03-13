// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package machine_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testutil "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
			Namespace: "default",
		},
		Spec: clusterv1.ClusterSpec{},
	}

	staticMachine = &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name":  "test",
				"cluster.x-k8s.io/control-plane": "",
			},
			Annotations: map[string]string{
				"pre-terminate.delete.hook.machine.cluster.x-k8s.io/avi-cleanup": "ako-operator",
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: "test",
		},
	}

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
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
	})

	When("A Cluster is created", func() {
		BeforeEach(func() {
			cluster.Labels = testLabels
			testutil.CreateObjects(ctx, cluster, machine)
		})

		AfterEach(func() {
			testutil.DeleteObjects(ctx, cluster, machine)
		})

		When("AVI is HA Provider", func() {
			JustBeforeEach(func() {
				err = os.Setenv(ako_operator.IsControlPlaneHAProvider, "True")
				Expect(err).ShouldNot(HaveOccurred())
				machine.Status = clusterv1.MachineStatus{
					Addresses: []clusterv1.MachineAddress{{
						Address: "1.1.1.1",
						Type:    clusterv1.MachineExternalIP,
					}},
				}
				testutil.UpdateObjectsStatus(ctx, machine)
			})
			It("Corresponding Endpoints should be created", func() {
				ep := &corev1.Endpoints{}
				Eventually(func() int {
					err := ctx.Client.Get(ctx.Context, client.ObjectKey{Name: cluster.Namespace + "-" + cluster.Name + "-control-plane", Namespace: cluster.Namespace}, ep)
					if err != nil {
						return 0
					}
					if len(ep.Subsets) == 0 {
						return 0
					}
					return len(ep.Subsets[0].Addresses)
				}).Should(Equal(1))
				Expect(ep.Subsets[0].Addresses[0].IP).Should(Equal("1.1.1.1"))
			})
			It("Should add one more machine", func() {
				secondMachine := staticMachine.DeepCopy()
				secondMachine.Name = "test-machine-2"
				secondMachine.Namespace = cluster.Namespace
				testutil.CreateObjects(ctx, secondMachine)
				secondMachine.Status = clusterv1.MachineStatus{
					Addresses: []clusterv1.MachineAddress{{
						Address: "1.1.1.2",
						Type:    clusterv1.MachineExternalIP,
					}},
				}
				testutil.UpdateObjectsStatus(ctx, secondMachine)

				ep := &corev1.Endpoints{}
				Eventually(func() bool {
					err := ctx.Client.Get(ctx.Context, client.ObjectKey{Name: cluster.Namespace + "-" + cluster.Name + "-control-plane", Namespace: cluster.Namespace}, ep)
					return err == nil
				}).Should(BeTrue())
				Expect(ep.Subsets).ShouldNot(BeNil())
				Expect(ep.Subsets[0].Addresses).ShouldNot(BeNil())
				testutil.DeleteObjects(ctx, secondMachine)
			})
		})

		It("Delete one machine directly", func() {
			testutil.DeleteObjects(ctx, machine)
			Eventually(func() bool {
				err := ctx.Client.Get(ctx.Context, client.ObjectKey{Name: machine.Name, Namespace: machine.Namespace}, &clusterv1.Machine{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})
}
