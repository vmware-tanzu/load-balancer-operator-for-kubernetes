// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)



var _ = ginkgo.Describe("Test get primary ipFamily", func() {
	ginkgo.It("should return V4 from CIDR", func() {
		cidr := "192.168.0.0/16"
		ipFamily := GetIPFamilyFromCidr(cidr)
		Expect(ipFamily).To(Equal("V4"))
	})

	ginkgo.It("should return V6 from CIDR", func() {
		cidr := "2002::1234:abcd:ffff:c0a8:101/64"
		ipFamily := GetIPFamilyFromCidr(cidr)
		Expect(ipFamily).To(Equal("V6"))
	})

	ginkgo.It("should return INVALID from CIDR", func() {
		cidr := "2002::1234:abcd:ffff:c0a8:101"
		ipFamily := GetIPFamilyFromCidr(cidr)
		Expect(ipFamily).To(Equal("INVALID"))
	})
})
