// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
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
	)
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(clusterv1.AddToScheme(scheme)).NotTo(HaveOccurred())
		fclient = fakeClient.NewFakeClientWithScheme(scheme)
		logger = log.Log
		log.SetLogger(zap.New())
	})

	JustBeforeEach(func() {
		machineClusterHandler = MachinesForCluster(fclient, logger)
		requests = machineClusterHandler.Map(input)
	})
	When("the cluster is from the system namespace", func() {
		BeforeEach(func() {
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: v1alpha1.TKGSystemNamespace,
				},
			}
			input = handler.MapObject{
				Object: cluster,
			}
		})
		It("should not create any request", func() {
			Expect(len(requests)).To(Equal(0))
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

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
			}
			input = handler.MapObject{
				Object: cluster,
			}
		})
		It("should create two request", func() {
			Expect(len(requests)).To(Equal(2))
		})
	})

})
