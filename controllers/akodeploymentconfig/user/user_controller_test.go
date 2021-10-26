// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AkoUserReconcilerTest() {
	var (
		err                 error
		ctx                 context.Context
		userReconciler      *AkoUserReconciler
		akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
		aviUsername         string
		aviPwd              string
		aviCA               string
		managmentSecretName string
		workloadSecretName  string
	)
	BeforeEach(func() {
		ctx = context.Background()
		mgr := suite.GetManager()
		testClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
		Expect(err).ShouldNot(HaveOccurred())

		userReconciler = NewProvider(testClient,
			nil,
			ctrl.Log.WithName("reconciler").WithName("AviUser"),
			mgr.GetScheme())

		akoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ako-deployment-config",
				Namespace: "default",
			},
			Spec: akoov1alpha1.AKODeploymentConfigSpec{
				ClusterSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "test",
					},
				},
				DataNetwork: akoov1alpha1.DataNetwork{
					Name:    "test",
					CIDR:    "1.1.1.1/20",
					IPPools: []akoov1alpha1.IPPool{},
				},
				CertificateAuthorityRef: &akoov1alpha1.SecretRef{
					Name:      "test-ca-secret",
					Namespace: "default",
				},
				AdminCredentialRef: &akoov1alpha1.SecretRef{},
			},
		}

		aviUsername = "fake-username"
		aviPwd = "fake-pwd"
		aviCA = "fake-ca"

		managmentSecretName = "fake-management-secret"
		workloadSecretName = "fake-workload-secret"

		err = userReconciler.Create(ctx, akoDeploymentConfig)
		Expect(err).ShouldNot(HaveOccurred())
	})
	AfterEach(func() {
		err = userReconciler.Delete(ctx, akoDeploymentConfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Context("Should be able to create secret spec", func() {
		It("Should be able to get correct secret secret name and namespace", func() {
			secretName, secretNamespace := userReconciler.mcAVISecretNameNameSpace("test-cluster", "default")
			Expect(secretName).To(Equal("test-cluster-avi-credentials"))
			Expect(secretNamespace).To(Equal("default"))
		})

		It("Should be able to create correct secret in management cluster", func() {
			secret := userReconciler.createAviUserSecret(managmentSecretName, "default", aviUsername, aviPwd, aviCA, akoDeploymentConfig, false)
			Expect(secret.Name).To(Equal(managmentSecretName))
			Expect(secret.Namespace).To(Equal("default"))
			Expect(len(secret.Data)).To(Equal(3))
		})

		It("Should be able to create correct secret spec in workload cluster", func() {
			secret := userReconciler.createAviUserSecret(workloadSecretName, akoov1alpha1.AviNamespace, aviUsername, aviPwd, aviCA, akoDeploymentConfig, true)
			Expect(secret.Name).To(Equal(workloadSecretName))
			Expect(secret.Namespace).To(Equal(akoov1alpha1.AviNamespace))
			Expect(len(secret.Data)).To(Equal(3))
		})
	})

	Context("Should be able to get avi controller ca", func() {
		var caSecret *corev1.Secret

		BeforeEach(func() {
			aviControllerCA := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ca-secret",
					Namespace: "default",
				},
			}
			err = userReconciler.Create(ctx, aviControllerCA)
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			err = userReconciler.Delete(ctx, caSecret)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("get avi controller ca", func() {
			caSecret, err = userReconciler.getAVIControllerCA(ctx, akoDeploymentConfig)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(caSecret.Name).To(Equal("test-ca-secret"))
			Expect(caSecret.Namespace).To(Equal("default"))
		})
	})
}
