// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package machine_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
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
		setMachineFinalizers := func(machine *clusterv1.Machine, finalizers []string) {
			Eventually(func() error {
				err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(machine), machine)
				if apierrors.IsNotFound(err) {
					return nil
				}
				if err != nil {
					return err
				}
				machine.SetFinalizers(finalizers)
				return ctx.Client.Update(ctx.Context, machine)
			}).Should(Succeed())
		}

		BeforeEach(func() {
			cluster.Labels = testLabels
			testutil.CreateObjects(ctx, cluster, machine)
			// Set Machine finalizers to mimic CAPI
			setMachineFinalizers(machine, []string{clusterv1.MachineFinalizer})
		})

		AfterEach(func() {
			testutil.DeleteObjects(ctx, machine)
			// Remove Machine finalizers to mimic CAPI
			setMachineFinalizers(machine, []string{})

			Eventually(func() bool {
				err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(machine), &clusterv1.Machine{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())

			testutil.DeleteObjects(ctx, cluster)
		})

		When("AVI is HA Provider", func() {
			BeforeEach(func() {
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

			AfterEach(func() {
				err = os.Setenv(ako_operator.IsControlPlaneHAProvider, "False")
				Expect(err).ShouldNot(HaveOccurred())
			})

			assertEndpointsForCluster := func(assertionFunc func(interface{}, ...interface{}) AsyncAssertion, cluster *clusterv1.Cluster, expectedIPs []string) {
				assertionFunc(func() []string {
					ep := &corev1.Endpoints{}
					err := ctx.Client.Get(ctx.Context, client.ObjectKey{Name: cluster.Namespace + "-" + cluster.Name + "-control-plane", Namespace: cluster.Namespace}, ep)
					if err != nil {
						return nil
					}
					if len(ep.Subsets) == 0 {
						return nil
					}
					actualIPs := []string{}
					for _, addr := range ep.Subsets[0].Addresses {
						actualIPs = append(actualIPs, addr.IP)
					}
					return actualIPs
				}).Should(ConsistOf(expectedIPs))
			}

			It("Corresponding Endpoints should be created", func() {
				assertEndpointsForCluster(Eventually, cluster, []string{"1.1.1.1"})
			})

			Context("when another Machine is added", func() {
				var secondMachine *clusterv1.Machine

				BeforeEach(func() {
					secondMachine = staticMachine.DeepCopy()
					secondMachine.Name = "test-machine-2"
					secondMachine.Namespace = cluster.Namespace
					testutil.CreateObjects(ctx, secondMachine)
					// Set Machine finalizers to mimic CAPI
					setMachineFinalizers(secondMachine, []string{clusterv1.MachineFinalizer})
					secondMachine.Status = clusterv1.MachineStatus{
						Addresses: []clusterv1.MachineAddress{{
							Address: "1.1.1.2",
							Type:    clusterv1.MachineExternalIP,
						}},
					}
					testutil.UpdateObjectsStatus(ctx, secondMachine)
				})

				AfterEach(func() {
					testutil.DeleteObjects(ctx, secondMachine)
					// Remove Machine finalizers to mimic CAPI
					setMachineFinalizers(secondMachine, []string{})

					assertEndpointsForCluster(Eventually, cluster, []string{"1.1.1.1"})
					Eventually(func() bool {
						err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(secondMachine), &clusterv1.Machine{})
						return apierrors.IsNotFound(err)
					}).Should(BeTrue())
				})

				It("the corresponding Endpoint addresses are updated", func() {
					assertEndpointsForCluster(Eventually, cluster, []string{"1.1.1.1", "1.1.1.2"})
				})
			})

			Context("when reconciliation is paused", func() {
				BeforeEach(func() {
					assertEndpointsForCluster(Eventually, cluster, []string{"1.1.1.1"})
				})

				It("by setting the cluster spec, Endpoints should not change after a machine is deleted", func() {
					cluster.Spec.Paused = true
					Expect(ctx.Client.Update(ctx.Context, cluster)).To(Succeed())

					testutil.DeleteObjects(ctx, machine)
					setMachineFinalizers(machine, []string{})
					Eventually(func() bool {
						err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(machine), &clusterv1.Machine{})
						return apierrors.IsNotFound(err)
					}).Should(BeTrue())

					assertEndpointsForCluster(Consistently, cluster, []string{"1.1.1.1"})
				})

				It("by setting the machine paused annotation, Endpoints should not change after a machine is deleted", func() {
					machine.Annotations[clusterv1.PausedAnnotation] = ""
					Expect(ctx.Client.Update(ctx.Context, machine)).To(Succeed())

					testutil.DeleteObjects(ctx, machine)
					setMachineFinalizers(machine, []string{})
					Eventually(func() bool {
						err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(machine), &clusterv1.Machine{})
						return apierrors.IsNotFound(err)
					}).Should(BeTrue())

					assertEndpointsForCluster(Consistently, cluster, []string{"1.1.1.1"})
				})
			})
		})

		It("the pre-terminate Machine annotation should be removed when a Machine is deleted before its parent Cluster", func() {
			testutil.DeleteObjects(ctx, machine)
			Eventually(func() map[string]string {
				err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(machine), machine)
				if err != nil {
					return nil
				}
				return machine.Annotations
			}).ShouldNot(ContainElement(akoov1alpha1.PreTerminateAnnotation))

			setMachineFinalizers(machine, []string{})
			Eventually(func() bool {
				err := ctx.Client.Get(ctx.Context, client.ObjectKeyFromObject(machine), &clusterv1.Machine{})
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})
}
