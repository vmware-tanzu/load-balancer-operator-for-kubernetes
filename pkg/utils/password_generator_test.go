// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/utils"
	"strings"
)

var _ = ginkgo.Describe("Test password generate", func() {
	ginkgo.It("should contain lowercase", func() {
		pwd := utils.GenereatePassword(5, true, false, false, false)
		Expect(strings.ContainsAny(pwd, "abcdefghijklmnopqrstuvwxyz"))
		Expect(len(pwd)).To(Equal(5))
	})
	ginkgo.It("should contain uppercase", func() {
		pwd := utils.GenereatePassword(5, false, true, false, false)
		Expect(strings.ContainsAny(pwd, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
		Expect(len(pwd)).To(Equal(5))
	})
	ginkgo.It("should contain specials", func() {
		pwd := utils.GenereatePassword(5, false, false, true, false)
		Expect(strings.ContainsAny(pwd, "~=+%^*/()[]{}/!@#$?|"))
		Expect(len(pwd)).To(Equal(5))
	})
	ginkgo.It("should contain digits", func() {
		pwd := utils.GenereatePassword(5, false, false, false, true)
		Expect(strings.ContainsAny(pwd, "0123456789"))
		Expect(len(pwd)).To(Equal(5))
	})
})
