// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"log"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/vmware-samples/load-balancer-operator-for-kubernetes/e2e/pkg/env"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AKO Operator e2e Suite")
}

func init() {
	p, exist := os.LookupEnv("E2E_ENV_SPEC")
	if !exist {
		GinkgoT().Logf("Skip, cannot get e2e test environment spec, is E2E_ENV_SPEC set and pointing to the right path?\n")
		return
	}
	register(p)
}

func register(p string) {
	var _ = BeforeSuite(func() {
		GinkgoT().Logf("Starting settting up environment\n")

		if err := env.LoadTestEnv(p); err != nil {
			log.Fatalf("Cannot load e2e test environment spec, %s", err.Error())
		}

		GinkgoT().Logf("\n\nStarting tests defined in %s\n\n", p)

		env.ShowThePlan()
	})
}
