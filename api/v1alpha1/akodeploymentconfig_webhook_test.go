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

type ModifyTestCaseInputFunc func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig)

func beforeAll(t *testing.T) (staticAdminSecret, staticCASecret *corev1.Secret, staticADC AKODeploymentConfig, g *WithT) {
	runTest = true
	kclient = fake.NewClientBuilder().Build()
	aviClient = aviclient.NewFakeAviClient()
	configureAVIController()

	staticAdminSecret = &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-avi-credentials",
			Namespace: "default",
		},
		StringData: map[string]string{
			"username": "admin",
			"password": "test123",
		},
	}

	staticCASecret = &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-avi-certificate",
			Namespace: "default",
		},
		StringData: map[string]string{
			"certificateAuthorityData": "test1223",
		},
	}

	staticADC = AKODeploymentConfig{
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
	}
	g = NewWithT(t)
	return staticAdminSecret, staticCASecret, staticADC, g
}

func afterEach(adminSecret, certificateSecret *corev1.Secret, g *WithT) {
	if adminSecret != nil {
		err := kclient.Delete(context.Background(), adminSecret)
		g.Expect(err).ShouldNot(HaveOccurred())
	}
	if certificateSecret != nil {
		err := kclient.Delete(context.Background(), certificateSecret)
		g.Expect(err).ShouldNot(HaveOccurred())
	}
	// restore avi client configuration
	configureAVIController()
}

func configureAVIController() {
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
}

func TestCreateNewAKODeploymentConfig(t *testing.T) {
	staticAdminSecret, staticCASecret, staticADC, g := beforeAll(t)
	testcases := []struct {
		name              string
		adminSecret       *corev1.Secret
		certificateSecret *corev1.Secret
		adc               *AKODeploymentConfig
		customizeInput    ModifyTestCaseInputFunc
		expectErr         bool
	}{
		{
			name:              "valid akodeployment config should pass webhook validation",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
		{
			name:              "default controller version should pass webhook validation",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ControllerVersion = ""
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
		{
			name:              "wrong controller version formt should return error",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ControllerVersion = "test adc version"
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "cluster selector should not be empty for non-default adc",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ClusterSelector = v1.LabelSelector{}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "default adc cluster selector can be empty",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Name = WorkloadClusterAkoDeploymentConfig
				adc.Spec.ClusterSelector = v1.LabelSelector{}
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
		{
			name:              "control plane network field can be empty",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ControlPlaneNetwork = ControlPlaneNetwork{}
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
		{
			name:              "should throw error if not find avi admin secret or certificate sercret",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				return nil, nil, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if not find avi cloud",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				aviClient.CloudCreate(nil)
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if not find avi service engine group",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				aviClient.ServiceEngineGroupCreate(nil)
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if not find avi control plane network",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				aviClient.NetworkCreate(nil)
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if invalid control plane network cidr",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ControlPlaneNetwork = ControlPlaneNetwork{
					Name: "VM Network 1",
					CIDR: "test 1",
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if not find avi data plane network",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				aviClient.NetworkCreate(nil)
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if invalid data plane network cidr",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.DataNetwork = DataNetwork{
					Name: "VM Network 1",
					CIDR: "test 1",
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if invalid data plane network ip pools",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.DataNetwork = DataNetwork{
					Name: "VM Network 1",
					CIDR: "10.0.0.1/24",
					IPPools: []IPPool{
						{
							Start: "test",
							End:   "test",
							Type:  "V4",
						},
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if invalid data plane network ip pools",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.DataNetwork = DataNetwork{
					Name: "VM Network 1",
					CIDR: "10.0.0.1/24",
					IPPools: []IPPool{
						{
							Start: "12.0.0.5",
							End:   "12.0.0.1",
							Type:  "V4",
						},
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "should throw error if addr type doesn't match ippool type",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.DataNetwork = DataNetwork{
					Name: "VM Network 1",
					CIDR: "10.0.0.1/24",
					IPPools: []IPPool{
						{
							Start: "2002::1234:abcd:ffff:c0a8:101/64",
							End:   "2002::1234:abcd:ffff:c0a8:102/64",
							Type:  "V6",
						},
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "valid ipv6 cidr",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.DataNetwork = DataNetwork{
					Name: "VM Network 1",
					CIDR: "2002::1234:abcd:ffff:c0a8:101/64",
					IPPools: []IPPool{
						{
							Start: "2002::1234:abcd:ffff:c0a8:101",
							End:   "2002::1234:abcd:ffff:c0a8:102",
							Type:  "V6",
						},
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.adminSecret, tc.certificateSecret, tc.adc = tc.customizeInput(tc.adminSecret, tc.certificateSecret, tc.adc)
			if tc.adminSecret != nil {
				err := kclient.Create(context.Background(), tc.adminSecret)
				g.Expect(err).ShouldNot(HaveOccurred())
			}
			if tc.certificateSecret != nil {
				err := kclient.Create(context.Background(), tc.certificateSecret)
				g.Expect(err).ShouldNot(HaveOccurred())
			}

			_, err := tc.adc.ValidateCreate()
			if !tc.expectErr {
				g.Expect(err).ShouldNot(HaveOccurred())
			} else {
				g.Expect(err).Should(HaveOccurred())
			}

			afterEach(tc.adminSecret, tc.certificateSecret, g)
		})
	}
}

func TestUpdateExistingAKODeploymentConfig(t *testing.T) {
	staticAdminSecret, staticCASecret, staticADC, g := beforeAll(t)
	testcases := []struct {
		name              string
		adminSecret       *corev1.Secret
		certificateSecret *corev1.Secret
		old               *AKODeploymentConfig
		new               *AKODeploymentConfig
		customizeInput    ModifyTestCaseInputFunc
		expectErr         bool
	}{
		{
			name:              "valid akodeployment update should pass webhook validation",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			old:               staticADC.DeepCopy(),
			new:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				aviClient.CloudCreate(&models.Cloud{
					Name:            pointer.StringPtr("fake-new-cloud"),
					IPAMProviderRef: pointer.StringPtr("https://10.0.0.x/api/ipamdnsproviderprofile/test"),
				})
				aviClient.ServiceEngineGroupCreate(&models.ServiceEngineGroup{
					Name: pointer.StringPtr("fake-seg"),
				})
				aviClient.NetworkCreate(&models.Network{
					Name: pointer.StringPtr("fake-new-data-plane"),
				})
				adc.Spec.CloudName = "fake-new-cloud"
				adc.Spec.ServiceEngineGroup = "fake-new-seg"
				adc.Spec.DataNetwork = DataNetwork{
					Name: "fake-new-data-plane",
					CIDR: "11.0.0.0/24",
					IPPools: []IPPool{
						{
							Start: "11.0.0.1",
							End:   "11.0.0.10",
							Type:  "V4",
						},
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
		{
			name:              "akodeployment should not update cluster selector",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			old:               staticADC.DeepCopy(),
			new:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ClusterSelector = v1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "bar",
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "akodeployment should not update cluster selector",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			old:               staticADC.DeepCopy(),
			new:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ClusterSelector = v1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "bar",
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "akodeployment should not update control plane network",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			old:               staticADC.DeepCopy(),
			new:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Spec.ControlPlaneNetwork = ControlPlaneNetwork{
					Name: "fake-new-control-plane",
					CIDR: "12.0.0.0/24",
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
		{
			name:              "akodeployment should not update to invalid cloud, seg and data plane",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			old:               staticADC.DeepCopy(),
			new:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				aviClient.CloudCreate(nil)
				aviClient.NetworkCreate(nil)
				aviClient.ServiceEngineGroupCreate(nil)

				adc.Spec.CloudName = "fake-new-cloud"
				adc.Spec.ServiceEngineGroup = "fake-new-seg"
				adc.Spec.DataNetwork = DataNetwork{
					Name: "fake-new-data-plane",
					CIDR: "11.0.0.0/24",
					IPPools: []IPPool{
						{
							Start: "test",
							End:   "test",
							Type:  "V4",
						},
					},
				}
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.adminSecret, tc.certificateSecret, tc.new = tc.customizeInput(tc.adminSecret, tc.certificateSecret, tc.new)
			if tc.adminSecret != nil {
				err := kclient.Create(context.Background(), tc.adminSecret)
				g.Expect(err).ShouldNot(HaveOccurred())
			}
			if tc.certificateSecret != nil {
				err := kclient.Create(context.Background(), tc.certificateSecret)
				g.Expect(err).ShouldNot(HaveOccurred())
			}
			_, err := tc.new.ValidateUpdate(tc.old)
			if !tc.expectErr {
				g.Expect(err).ShouldNot(HaveOccurred())
			} else {
				g.Expect(err).Should(HaveOccurred())
			}

			afterEach(tc.adminSecret, tc.certificateSecret, g)
		})
	}
}

func TestDeleteAKODeploymentConfig(t *testing.T) {
	staticAdminSecret, staticCASecret, staticADC, g := beforeAll(t)

	testcases := []struct {
		name              string
		adminSecret       *corev1.Secret
		certificateSecret *corev1.Secret
		adc               *AKODeploymentConfig
		customizeInput    ModifyTestCaseInputFunc
		expectErr         bool
	}{
		{
			name:              "valid akodeployment config should pass webhook validation",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				return adminSecret, certificateSecret, adc
			},
			expectErr: false,
		},
		{
			name:              "management cluster akodeployment object should not be deleted",
			adminSecret:       staticAdminSecret.DeepCopy(),
			certificateSecret: staticCASecret.DeepCopy(),
			adc:               staticADC.DeepCopy(),
			customizeInput: func(adminSecret, certificateSecret *corev1.Secret, adc *AKODeploymentConfig) (*corev1.Secret, *corev1.Secret, *AKODeploymentConfig) {
				adc.Name = ManagementClusterAkoDeploymentConfig
				return adminSecret, certificateSecret, adc
			},
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.adminSecret, tc.certificateSecret, tc.adc = tc.customizeInput(tc.adminSecret, tc.certificateSecret, tc.adc)
			if tc.adminSecret != nil {
				err := kclient.Create(context.Background(), tc.adminSecret)
				g.Expect(err).ShouldNot(HaveOccurred())
			}
			if tc.certificateSecret != nil {
				err := kclient.Create(context.Background(), tc.certificateSecret)
				g.Expect(err).ShouldNot(HaveOccurred())
			}

			_, err := tc.adc.ValidateDelete()
			if !tc.expectErr {
				g.Expect(err).ShouldNot(HaveOccurred())
			} else {
				g.Expect(err).Should(HaveOccurred())
			}

			afterEach(tc.adminSecret, tc.certificateSecret, g)
		})
	}
}
