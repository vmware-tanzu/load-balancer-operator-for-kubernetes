// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AKO Operator", func() {
	Context("If ako operator is deployed in bootstrap cluster", func() {
		When("ako operator is deployed in bootstrap cluster", func() {
			BeforeEach(func() {
				os.Setenv(DeployInBootstrapCluster, "True")
			})
			It("should return True", func() {
				Expect(IsBootStrapCluster()).Should(Equal(true))
			})
		})
	})

	Context("If ako operator is going to provide control plane HA", func() {
		When("ako operator provides control plane HA", func() {
			BeforeEach(func() {
				os.Setenv(IsControlPlaneHAProvider, "True")
			})
			It("should return True", func() {
				Expect(IsHAProvider()).Should(Equal(true))
			})
		})
	})

	Context("Get control plane endpoint port", func() {
		When("There is a valid control plane endpoint port", func() {
			BeforeEach(func() {
				os.Setenv(ControlPlaneEndpointPort, "6001")
			})
			It("should return port in env", func() {
				Expect(GetControlPlaneEndpointPort()).Should(Equal(int32(6001)))
			})
		})

		When("There is an invalid control plane endpoint port", func() {
			BeforeEach(func() {
				os.Setenv(ControlPlaneEndpointPort, "-1")
			})
			It("should return port 6443", func() {
				Expect(GetControlPlaneEndpointPort()).Should(Equal(int32(6443)))
			})
		})
	})
})
