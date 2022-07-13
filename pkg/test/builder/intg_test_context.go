// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"context"

	//nolint
	. "github.com/onsi/ginkgo"
	uuid "github.com/satori/go.uuid"

	//nolint
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IntegrationTestContext is used for integration testing. Each
// IntegrationTestContext contains one separate namespace
type IntegrationTestContext struct {
	context.Context
	Client    client.Client
	AviClient aviclient.FakeAviClient
	Namespace string
	suite     *TestSuite
}

func (*IntegrationTestContext) GetLogger() logr.Logger {
	// return logr.DiscardLogger{}
	return logr.Discard()
}

// AfterEach should be invoked by ginkgo.AfterEach to destroy the test namespace
func (ctx *IntegrationTestContext) AfterEach() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ctx.Namespace,
		},
	}
	By("Destroying integration test namespace")
	Expect(ctx.suite.integrationTestClient.Delete(ctx, namespace)).To(Succeed())
}

// NewIntegrationTestContext should be invoked by ginkgo.BeforeEach
//
// This function creates a namespace with a random name to separate integration
// test cases
//
// This function returns a TestSuite context
// The resources created by this function may be cleaned up by calling AfterEach
// with the IntegrationTestContext returned by this function
func (s *TestSuite) NewIntegrationTestContext() *IntegrationTestContext {
	ctx := &IntegrationTestContext{
		Context:   context.Background(),
		AviClient: *FakeAvi,
		Client:    s.integrationTestClient,
		suite:     s,
	}

	By("Creating a temporary namespace", func() {
		nonce := uuid.NewV4().String()[0:6]
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-" + nonce, // do not start with digits nor too long, otherwise dns will complain
			},
		}
		Expect(ctx.Client.Create(s, namespace)).To(Succeed())

		ctx.Namespace = namespace.Name
	})

	return ctx
}
