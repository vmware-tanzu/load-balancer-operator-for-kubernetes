// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware/alb-sdk/go/models"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func beforeEach(t *testing.T) (staticADC AKODeploymentConfig, g *WithT) {
	runTest = true
	kclient = fake.NewClientBuilder().Build()
	aviClient = aviclient.NewFakeAviClient()

	staticADC = AKODeploymentConfig{}
	g = NewWithT(t)
	return staticADC, g
}

func TestCreateNewAKODeploymentConfig(t *testing.T) {
	_, g := beforeEach(t)

	testcases := []struct {
		name              string
		adminSecret       corev1.Secret
		certificateSecret corev1.Secret
		adc               AKODeploymentConfig
		expectErr         bool
		expectMsg         string
	}{
		{
			name: "test",
			adminSecret: corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-avi-credentials",
					Namespace: "default",
				},
				StringData: map[string]string{
					"username": "admin",
					"password": "test123",
				},
			},
			certificateSecret: corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test-avi-certificate",
					Namespace: "default",
				},
				StringData: map[string]string{
					"certificateAuthorityData": "test1223",
				},
			},
			adc: AKODeploymentConfig{
				ObjectMeta: v1.ObjectMeta{
					Name: "test",
				},
				Spec: AKODeploymentConfigSpec{
					Controller:         "1.1.1.1",
					ControllerVersion:  "20.1.7",
					CloudName:          "fake-cloud",
					ServiceEngineGroup: "fake-seg",
					AdminCredentialRef: &SecretRef{
						Name:      "test-avi-credentials",
						Namespace: "default",
					},
					CertificateAuthorityRef: &SecretRef{
						Name:      "test-avi-certificate",
						Namespace: "default",
					},
					ClusterSelector: v1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
					ControlPlaneNetwork: ControlPlaneNetwork{
						Name: "fake-control-plane",
						CIDR: "12.0.0.0/24",
					},
					DataNetwork: DataNetwork{
						Name: "fake-data-plane",
						CIDR: "10.0.0.0/24",
						IPPools: []IPPool{
							{
								Start: "10.0.0.1",
								End:   "10.0.0.10",
								Type:  "V4",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := kclient.Create(context.Background(), &tc.adminSecret)
			g.Expect(err).ShouldNot(HaveOccurred())
			err = kclient.Create(context.Background(), &tc.certificateSecret)
			g.Expect(err).ShouldNot(HaveOccurred())
			aviClient.ServiceEngineGroupCreate(&models.ServiceEngineGroup{
				Name: pointer.StringPtr("fake-seg"),
			})
			aviClient.CloudCreate(&models.Cloud{
				Name:            pointer.StringPtr("fake-cloud"),
				IPAMProviderRef: pointer.StringPtr("https://10.0.0.x/api/ipamdnsproviderprofile/test"),
			})
			aviClient.NetworkCreate(&models.Network{
				Name: pointer.StringPtr("fake-control-plane"),
			})
			aviClient.NetworkCreate(&models.Network{
				Name: pointer.StringPtr("fake-data-plane"),
			})
			err = tc.adc.ValidateCreate()
			if !tc.expectErr {
				g.Expect(err).ShouldNot(HaveOccurred())

			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		})
	}
}

func TestUpdateExistingAKODeploymentConfig(t *testing.T) {
	_, g := beforeEach(t)

	testcases := []struct {
		name      string
		old       AKODeploymentConfig
		new       AKODeploymentConfig
		expectErr bool
		expectMsg string
	}{}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.new.ValidateUpdate(&tc.old)

			if !tc.expectErr {
				g.Expect(err).To(BeNil())
			} else {
				g.Expect(err).Should(HaveOccurred())
			}
		})
	}
}
