// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Runtime Handlers Suite")
}
