// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package haprovider

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("Control Plane HA provider", func() {
	var (
		ctx        context.Context
		haProvider HAProvider
		err        error
	)
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())
		Expect(clusterv1.AddToScheme(scheme)).NotTo(HaveOccurred())
		log.SetLogger(zap.New())
		fc := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
		logger := log.Log
		haProvider = *NewProvider(fc, logger)
	})

	Context("Test_CreateOrUpdateHAEndpoints", func() {
		var (
			mc      *clusterv1.Machine
			cluster *clusterv1.Cluster
			ep      *corev1.Endpoints
			key     client.ObjectKey
		)
		BeforeEach(func() {
			mc = &clusterv1.Machine{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-mc",
					Namespace: "default",
				},
				Spec: clusterv1.MachineSpec{},
			}
			cluster = &clusterv1.Cluster{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: clusterv1.ClusterSpec{},
			}
		})

		JustBeforeEach(func() {
			err = haProvider.CreateOrUpdateHAEndpoints(ctx, mc)
		})

		It("machine is not a control plane machine, should skip", func() {
			Expect(err).ShouldNot(HaveOccurred())
		})

		When("machine is a control plane machine", func() {
			BeforeEach(func() {
				mc.ObjectMeta.Labels = map[string]string{clusterv1.MachineControlPlaneLabelName: ""}
				mc.Spec.ClusterName = "test-cluster"
			})

			It("the cluster machine belong doesn't exist, should throw out cluster not found error", func() {
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			When("the cluster machine belongs to exist", func() {
				BeforeEach(func() {
					Expect(haProvider.Client.Create(ctx, cluster)).ShouldNot(HaveOccurred())
					mc.Status.Addresses = clusterv1.MachineAddresses{
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineExternalIP,
							Address: "1.1.1.1",
						},
					}
					ep = &corev1.Endpoints{}
					key = client.ObjectKey{Name: haProvider.getHAServiceName(cluster), Namespace: mc.Namespace}
				})

				AfterEach(func() {
					Expect(haProvider.Client.Delete(ctx, ep)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Delete(ctx, cluster)).ShouldNot(HaveOccurred())
				})

				It("Should create a endpoints object and machine to the endpoints", func() {
					Expect(err).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(1))
					Expect(ep.Subsets[0].Addresses[0].IP).Should(Equal("1.1.1.1"))
					Expect(ep.Subsets[0].Addresses[0].NodeName).Should(Equal(pointer.StringPtr("test-mc")))
				})

				It("should not add a duplicated machine", func() {
					mc2 := mc.DeepCopy()

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc2)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(1))
					Expect(ep.Subsets[0].Addresses[0].IP).Should(Equal("1.1.1.1"))
					Expect(ep.Subsets[0].Addresses[0].NodeName).Should(Equal(pointer.StringPtr("test-mc")))
				})

				It("should not add machine's other type IP", func() {
					mc2 := mc.DeepCopy()
					mc2.Name = "test-mc-2"
					mc2.Status.Addresses = clusterv1.MachineAddresses{
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineInternalIP,
							Address: "1.1.1.1",
						},
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineExternalIP,
							Address: "test123",
						},
					}

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc2)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(1))
					Expect(ep.Subsets[0].Addresses[0].IP).Should(Equal("1.1.1.1"))
					Expect(ep.Subsets[0].Addresses[0].NodeName).Should(Equal(pointer.StringPtr("test-mc")))
				})

				It("should update endpoints when machine ip changed", func() {
					mc.Status.Addresses = clusterv1.MachineAddresses{
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineExternalIP,
							Address: "1.1.1.2",
						},
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineInternalIP,
							Address: "1.1.1.3",
						},
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineExternalIP,
							Address: "test123",
						},
					}

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(1))
					Expect(ep.Subsets[0].Addresses[0].IP).Should(Equal("1.1.1.2"))
					Expect(ep.Subsets[0].Addresses[0].NodeName).Should(Equal(pointer.StringPtr("test-mc")))
				})

				It("should remove machine from endpoints when machine deleting", func() {
					time := v1.Now()
					mc.DeletionTimestamp = &time

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(ep.Subsets).Should(BeNil())
				})

				It("[two machines] should delete the second machine", func() {
					mc2 := mc.DeepCopy()
					mc2.Name = "test-mc-2"
					mc2.Status.Addresses = clusterv1.MachineAddresses{
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineExternalIP,
							Address: "1.1.1.2",
						},
					}

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc2)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(2))

					time := v1.Now()
					mc.DeletionTimestamp = &time

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(1))
					Expect(ep.Subsets[0].Addresses[0].IP).Should(Equal("1.1.1.2"))
					Expect(ep.Subsets[0].Addresses[0].NodeName).Should(Equal(pointer.StringPtr("test-mc-2")))

					mc2.DeletionTimestamp = &time

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc2)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets)).Should(Equal(0))

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets)).Should(Equal(0))
				})

				// Once dual-stack is supported, change this test case
				It("Should Only support IPV4 for now", func() {
					mc2 := mc.DeepCopy()
					mc2.Name = "test-mc-2"
					mc2.Status.Addresses = clusterv1.MachineAddresses{
						clusterv1.MachineAddress{
							Type:    clusterv1.MachineExternalIP,
							Address: "fd01:3:4:2877:250:56ff:feb4:adaf",
						},
					}

					Expect(haProvider.CreateOrUpdateHAEndpoints(ctx, mc2)).ShouldNot(HaveOccurred())
					Expect(haProvider.Client.Get(ctx, key, ep)).ShouldNot(HaveOccurred())
					Expect(len(ep.Subsets[0].Addresses)).Should(Equal(1))
				})
			})

		})

	})
})
