// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

func TestAdcClusterSelectorNonEmpty(t *testing.T) {
	CustomizedAkoDeploymentConfig := "install-ako-for-l7"
	tests := []struct {
		name             string
		adcName          string
		clusterSelector  metav1.LabelSelector
		expectErr        bool
		expectErrType    field.ErrorType
		expectErrMessage string
	}{
		{
			name:            "when default adc has empty cluster selector",
			adcName:         WorkloadClusterAkoDeploymentConfig,
			clusterSelector: metav1.LabelSelector{},
			expectErr:       false,
		},
		{
			name:            "when default adc has nonempty cluster selector",
			adcName:         WorkloadClusterAkoDeploymentConfig,
			clusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			expectErr:       false,
		},
		{
			name:            "when customized adc has nonempty cluster selector",
			adcName:         CustomizedAkoDeploymentConfig,
			clusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			expectErr:       false,
		},
		{
			name:             "when customized adc has empty cluster selector",
			adcName:          CustomizedAkoDeploymentConfig,
			clusterSelector:  metav1.LabelSelector{},
			expectErr:        true,
			expectErrType:    field.ErrorTypeInvalid,
			expectErrMessage: "field should not be empty for non-default ADC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			adc := &AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.adcName,
				},
				Spec: AKODeploymentConfigSpec{ClusterSelector: tt.clusterSelector},
			}
			err := adc.ValidateCreate()
			if !tt.expectErr {
				g.Expect(err).To(BeNil())
			} else {
				fieldErr, ok := err.(*errors.StatusError)
				g.Expect(ok).To(BeTrue())
				g.Expect(fieldErr.Status().Details.Causes).To(Not(BeEmpty()))
				g.Expect(fieldErr.Status().Details.Causes[0].String()).To(ContainSubstring(tt.expectErrType.String()))
				g.Expect(fieldErr.Status().Details.Causes[0].String()).To(ContainSubstring(tt.expectErrMessage))
			}
		})
	}
}

func TestAdcClusterSelectorImmutable(t *testing.T) {
	tests := []struct {
		name               string
		oldClusterSelector metav1.LabelSelector
		newClusterSelector metav1.LabelSelector
		expectErr          bool
		expectErrType      field.ErrorType
		expectErrMessage   string
	}{
		{
			name:               "when label selector is not changed",
			oldClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			newClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			expectErr:          false,
		},
		{
			name:               "when label selector key is changed",
			oldClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			newClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo1": "bar"}},
			expectErr:          true,
			expectErrType:      field.ErrorTypeInvalid,
			expectErrMessage:   "field should not be changed",
		},
		{
			name:               "when label selector value is changed",
			oldClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
			newClusterSelector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar1"}},
			expectErr:          true,
			expectErrType:      field.ErrorTypeInvalid,
			expectErrMessage:   "field should not be changed",
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
				Spec: AKODeploymentConfigSpec{ClusterSelector: tt.newClusterSelector},
			}

			oldADC := &AKODeploymentConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test-8ed12g",
					Namespace: "integration-test-8ed12g",
				},
				Spec: AKODeploymentConfigSpec{ClusterSelector: tt.oldClusterSelector},
			}

			err := newADC.ValidateUpdate(oldADC)
			if !tt.expectErr {
				g.Expect(err).To(BeNil())
			} else {
				fieldErr, ok := err.(*errors.StatusError)
				g.Expect(ok).To(BeTrue())
				g.Expect(fieldErr.Status().Details.Causes).To(Not(BeEmpty()))
				g.Expect(fieldErr.Status().Details.Causes[0].String()).To(ContainSubstring(tt.expectErrType.String()))
				g.Expect(fieldErr.Status().Details.Causes[0].String()).To(ContainSubstring(tt.expectErrMessage))
			}
		})
	}
}
