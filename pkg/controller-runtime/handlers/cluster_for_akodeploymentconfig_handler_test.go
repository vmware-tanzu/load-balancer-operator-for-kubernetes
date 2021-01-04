// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
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

var (
	ctx     context.Context
	fclient client.Client
	logger  logr.Logger
)

var _ = Describe("AKODeploymentConfig Cluster Handler", func() {
	var (
		akoDeploymentConfighandler handler.Mapper
		requests                   []reconcile.Request
		input                      handler.MapObject
	)
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(akoov1alpha1.AddToScheme(scheme)).NotTo(HaveOccurred())
		fclient = fakeClient.NewFakeClientWithScheme(scheme)
		logger = log.Log
		log.SetLogger(zap.New())
	})

	JustBeforeEach(func() {
		akoDeploymentConfighandler = AkoDeploymentConfigForCluster(fclient, logger)
		requests = akoDeploymentConfighandler.Map(input)
	})
	When("no AKODeploymentConfig exists", func() {
		BeforeEach(func() {
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
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
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
			}
			input = handler.MapObject{
				Object: cluster,
			}
		})
		It("should create one request", func() {
			Expect(len(requests)).To(Equal(1))
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
			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
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

			cluster := &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
					Labels: map[string]string{
						"test1": "hey",
						"test2": "wow",
					},
				},
			}
			input = handler.MapObject{
				Object: cluster,
			}
		})
		It("should create only one request", func() {
			Expect(len(requests)).To(Equal(2))
		})
	})

})
