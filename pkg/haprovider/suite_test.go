// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package haprovider

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Control Plane HA provider suite")
}
