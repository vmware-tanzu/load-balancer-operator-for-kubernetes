// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AKO Operator Suite")
}
