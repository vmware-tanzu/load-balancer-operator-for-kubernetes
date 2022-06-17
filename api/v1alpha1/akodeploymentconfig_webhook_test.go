// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
)

func beforeEach(t *testing.T) (staticADC AKODeploymentConfig, g *WithT) {
	runTest = true
	staticADC = AKODeploymentConfig{}
	g = NewWithT(t)
	return staticADC, g
}

func TestCreateNewAKODeploymentConfig(t *testing.T) {
	_, g := beforeEach(t)

	testcases := []struct {
		name      string
		adc       AKODeploymentConfig
		expectErr bool
		expectMsg string
	}{}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.adc.ValidateCreate()
			if !tc.expectErr {
				g.Expect(err).To(BeNil())
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
