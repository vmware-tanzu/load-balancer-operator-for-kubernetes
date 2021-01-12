// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ako

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("AKO", func() {
	var (
		ctx      context.Context
		fclient  client.Client
		logger   logr.Logger
		ss       *appv1.StatefulSet
		finished bool
		err      error
		createSS bool
	)
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(appv1.AddToScheme(scheme)).NotTo(HaveOccurred())
		fclient = fakeClient.NewFakeClientWithScheme(scheme)
		logger = log.Log
		log.SetLogger(zap.New())
		ss = &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      akoStatefulSetName,
				Namespace: akoov1alpha1.AviNamespace,
			},
		}
		createSS = true
	})

	JustBeforeEach(func() {
		if createSS {
			Expect(fclient.Create(ctx, ss)).ToNot(HaveOccurred())
		}
		finished, err = CleanupFinished(ctx, fclient, logger)
	})

	When("StatefulSet does not exist", func() {
		BeforeEach(func() {
			createSS = false
		})
		It("should claim finished", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(finished).To(BeTrue())
		})
	})

	When("no status is in StatefulSet", func() {
		It("should not claim finished", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(finished).To(BeFalse())
		})
	})
	When("InProgress status is True", func() {
		BeforeEach(func() {
			ss.Status = appv1.StatefulSetStatus{
				Conditions: []appv1.StatefulSetCondition{
					appv1.StatefulSetCondition{
						Type:   akoConditionType,
						Status: corev1.ConditionTrue,
					},
				},
			}
		})
		It("should not claim finished", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(finished).To(BeFalse())
		})
	})
	When("InProgress status is False", func() {
		BeforeEach(func() {
			ss.Status = appv1.StatefulSetStatus{
				Conditions: []appv1.StatefulSetCondition{
					appv1.StatefulSetCondition{
						Type:   akoConditionType,
						Status: corev1.ConditionFalse,
					},
				},
			}
		})
		It("should claim finished", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(finished).To(BeTrue())
		})
	})
})
