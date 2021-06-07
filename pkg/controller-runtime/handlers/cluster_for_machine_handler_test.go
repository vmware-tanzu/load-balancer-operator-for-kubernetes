// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Machine Cluster Handler", func() {
	var (
		machineClusterHandler handler.Mapper
		requests              []reconcile.Request
		input                 handler.MapObject
		ctx                   context.Context
		fclient               client.Client
		logger                logr.Logger
		cluster               *clusterv1.Cluster
	)
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(clusterv1.AddToScheme(scheme)).NotTo(HaveOccurred())
		fclient = fakeClient.NewFakeClientWithScheme(scheme)
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
		machineClusterHandler = MachinesForCluster(fclient, logger)
		requests = machineClusterHandler.Map(input)
	})
	When("the cluster is from the system namespace", func() {
		BeforeEach(func() {
			cluster.Namespace = akoov1alpha1.TKGSystemNamespace
			input = handler.MapObject{
				Object: cluster,
			}
		})
		It("should not create any request", func() {
			Expect(len(requests)).To(Equal(0))
		})
	})

	When("the cluster is not ready", func() {
		BeforeEach(func() {
			machine1 := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine1",
					Namespace: "test",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "test",
					},
				},
			}
			machine2 := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine2",
					Namespace: "test",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "test",
					},
				},
			}

			Expect(fclient.Create(ctx, machine1)).NotTo(HaveOccurred())
			Expect(fclient.Create(ctx, machine2)).NotTo(HaveOccurred())

			cluster.Namespace = "test"
		})
		When("cluster is not being deleted", func() {
			BeforeEach(func() {
				conditions.MarkFalse(cluster, clusterv1.ReadyCondition, "test-reason", clusterv1.ConditionSeverityInfo, "test-msg")
				input = handler.MapObject{
					Object: cluster,
				}
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
				input = handler.MapObject{
					Object: cluster,
				}
			})
			It("should create 2 requests", func() {
				Expect(len(requests)).To(Equal(2))
			})
		})
	})

	When("the cluster has two machines", func() {
		BeforeEach(func() {
			machine1 := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine1",
					Namespace: "test",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "test",
					},
				},
			}
			machine2 := &clusterv1.Machine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine2",
					Namespace: "test",
					Labels: map[string]string{
						clusterv1.ClusterLabelName: "test",
					},
				},
			}

			Expect(fclient.Create(ctx, machine1)).NotTo(HaveOccurred())
			Expect(fclient.Create(ctx, machine2)).NotTo(HaveOccurred())

			cluster.Namespace = "test"
			input = handler.MapObject{
				Object: cluster,
			}
		})
		It("should create two request", func() {
			Expect(len(requests)).To(Equal(2))
		})
	})

})
