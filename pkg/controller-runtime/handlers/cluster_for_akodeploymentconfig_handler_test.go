// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	akov1alpha1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("AKODeploymentConfig Cluster Handler", func() {
	var (
		akoDeploymentConfigMapFunc handler.MapFunc
		requests                   []reconcile.Request
		input                      client.Object
		ctx                        context.Context
		fclient                    client.Client
		logger                     logr.Logger
		cluster                    *clusterv1.Cluster
	)
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(akoov1alpha1.AddToScheme(scheme)).NotTo(HaveOccurred())
		Expect(akov1alpha1.AddToScheme(scheme)).NotTo(HaveOccurred())
		fclient = fakeClient.NewClientBuilder().WithScheme(scheme).Build()
		logger = log.Log
		log.SetLogger(zap.New())
		cluster = &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
			},
		}
		conditions.MarkTrue(cluster, clusterv1.ReadyCondition)
	})

	JustBeforeEach(func() {
		akoDeploymentConfigMapFunc = AkoDeploymentConfigForCluster(fclient, logger)
		requests = akoDeploymentConfigMapFunc(input)
	})
	When("no AKODeploymentConfig exists", func() {
		BeforeEach(func() {
			input = cluster
		})
		It("should not create any request", func() {
			Expect(len(requests)).To(Equal(0))
		})
	})

	When("one matching AKODeploymentConfig exists", func() {
		BeforeEach(func() {
			akodeploymentconfigForAll := &akoov1alpha1.AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: akoov1alpha1.AKODeploymentConfigSpec{
					ClusterSelector: metav1.LabelSelector{},
				},
			}

			Expect(fclient.Create(ctx, akodeploymentconfigForAll)).NotTo(HaveOccurred())
			input = cluster
		})
		It("should create one request", func() {
			Expect(len(requests)).To(Equal(1))
		})
		When("the cluster is from the system namespace", func() {
			BeforeEach(func() {
				cluster.Namespace = akoov1alpha1.TKGSystemNamespace
				input = cluster
			})
			// After Dakar, ako would also be deployed in management cluster.
			It("should create 1 request", func() {
				Expect(len(requests)).To(Equal(1))
			})
		})
		When("the cluster is not ready", func() {
			When("cluster is not being deleted", func() {
				BeforeEach(func() {
					conditions.MarkFalse(cluster, clusterv1.ReadyCondition, "test-reason", clusterv1.ConditionSeverityInfo, "test-msg")
					input = cluster
				})
				It("should not create any request", func() {
					Expect(len(requests)).To(Equal(0))
				})
			})
			When("cluster is being deleted", func() {
				BeforeEach(func() {
					conditions.MarkFalse(cluster, clusterv1.ReadyCondition, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "")
					deletionTime := metav1.NewTime(time.Now())
					cluster.SetDeletionTimestamp(&deletionTime)
					input = cluster
				})
				It("should create 1 request", func() {
					Expect(len(requests)).To(Equal(1))
				})
			})
		})
	})

	When("more than one non-matching AKODeploymentConfig exists", func() {
		BeforeEach(func() {
			akodeploymentconfig1 := &akoov1alpha1.AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: akoov1alpha1.AKODeploymentConfigSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "hey",
						},
					},
				},
			}
			akodeploymentconfig2 := &akoov1alpha1.AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
				},
				Spec: akoov1alpha1.AKODeploymentConfigSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "wow",
						},
					},
				},
			}

			Expect(fclient.Create(ctx, akodeploymentconfig1)).NotTo(HaveOccurred())
			Expect(fclient.Create(ctx, akodeploymentconfig2)).NotTo(HaveOccurred())
			input = cluster
		})
		It("should not create any request", func() {
			Expect(len(requests)).To(Equal(0))
		})
	})

	When("two AKODeploymentConfigs match one Cluster", func() {
		BeforeEach(func() {
			akodeploymentconfig1 := &akoov1alpha1.AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test1",
				},
				Spec: akoov1alpha1.AKODeploymentConfigSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test1": "hey",
						},
					},
				},
			}
			akodeploymentconfig2 := &akoov1alpha1.AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test2",
				},
				Spec: akoov1alpha1.AKODeploymentConfigSpec{
					ClusterSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test2": "wow",
						},
					},
				},
			}

			Expect(fclient.Create(ctx, akodeploymentconfig1)).NotTo(HaveOccurred())
			Expect(fclient.Create(ctx, akodeploymentconfig2)).NotTo(HaveOccurred())

			cluster.Labels = map[string]string{
				"test1": "hey",
				"test2": "wow",
			}
			input = cluster
		})
		It("should create only one request", func() {
			Expect(len(requests)).To(Equal(2))
		})
	})

})
