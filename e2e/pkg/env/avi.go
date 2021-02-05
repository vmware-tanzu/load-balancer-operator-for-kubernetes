// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/aviclient"
)

func NewAviRunner(runner *KubectlRunner) *aviclient.Client {

	aviClient, _ := aviclient.NewAviClient(&aviclient.AviClientConfig{
		ServerIP: GetAviObject(runner, "akodeploymentconfig", "ako-deployment-config", "spec", "controller"),
		Username: GetAviObject(runner, "secret", "controller-credentials", "data", "username"),
		Password: GetAviObject(runner, "secret", "controller-credentials", "data", "password"),
		CA:       GetAviObject(runner, "secret", "controller-ca", "data", "certificateAuthorityData"),
	})

	return aviClient
}

func EnsureAviObjectDeleted(aviClient *aviclient.Client, clusterName string, obj string) {
	Eventually(func() bool {
		var err error

		switch obj {
		case "virtualservice":
			_, err = aviClient.VirtualService.GetByName(clusterName + "--default-static-ip")
		case "pool":
			_, err = aviClient.Pool.GetByName(clusterName + "--default-static-ip--80")
		default:
			GinkgoT().Logf("EnsureAviObjectDeleted function doesn't support checking " + obj)
			return false
		}

		if err != nil {
			if strings.Contains(err.Error(), "No object of type "+obj) {
				GinkgoT().Logf("No object of type " + obj + " with name " + clusterName + " is found")
				return true
			}
			GinkgoT().Logf("Avi Client query error:" + err.Error())
			return false
		}
		GinkgoT().Logf(obj + " with name " + clusterName + " is found unexpectedly, return false")
		return false
	}, "30s", "5s").Should(BeTrue())
}
