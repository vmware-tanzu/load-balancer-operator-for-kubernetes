// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAdcControlPlaneNetworkImmutable(t *testing.T) {
	tests := []struct {
		name            string
		oldControlPlane ControlPlaneNetwork
		newControlPlane ControlPlaneNetwork
		expectErr       bool
	}{
		{
			name: "when the control plane name and CIDR is not changed",
			oldControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.1.0.0/24",
			},
			newControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.1.0.0/24",
			},
			expectErr: false,
		},
		{
			name:            "when the old control plane is empty",
			oldControlPlane: ControlPlaneNetwork{},
			newControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.1.0.0/24",
			},
			expectErr: true,
		},
		{
			name: "when the control plane name is changed",
			oldControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.1.0.0/24",
			},
			newControlPlane: ControlPlaneNetwork{
				Name: "VM Network 2",
				CIDR: "10.1.0.0/24",
			},
			expectErr: true,
		},
		{
			name: "when the control plane CIDR is changed",
			oldControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.1.0.0/24",
			},
			newControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.0.0.0/24",
			},
			expectErr: true,
		},
		{
			name: "when the control plane CIDR is changed",
			oldControlPlane: ControlPlaneNetwork{
				Name: "VM Network 1",
				CIDR: "10.1.0.0/24",
			},
			newControlPlane: ControlPlaneNetwork{},
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			newADC := &AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test-8ed12g",
					Namespace: "integration-test-8ed12g",
				},
				Spec: AKODeploymentConfigSpec{
					DataNetwork: DataNetwork{
						Name: "integration-test-8ed12g",
						CIDR: "10.0.0.0/24",
						IPPools: []IPPool{
							{
								Start: "10.0.0.1",
								End:   "10.0.0.10",
								Type:  "V4",
							},
						},
					},
					ControlPlaneNetwork: tt.newControlPlane,
				},
			}

			oldADC := &AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test-8ed12g",
					Namespace: "integration-test-8ed12g",
				},
				Spec: AKODeploymentConfigSpec{
					DataNetwork: DataNetwork{
						Name: "integration-test-8ed12g",
						CIDR: "10.0.0.0/24",
						IPPools: []IPPool{
							{
								Start: "10.0.0.1",
								End:   "10.0.0.10",
								Type:  "V4",
							},
						},
					},
					ControlPlaneNetwork: tt.oldControlPlane,
				},
			}

			if tt.expectErr {
				g.Expect(newADC.ValidateUpdate(oldADC)).NotTo(Succeed())
			} else {
				g.Expect(newADC.ValidateUpdate(oldADC)).To(Succeed())
			}
		})
	}
}
