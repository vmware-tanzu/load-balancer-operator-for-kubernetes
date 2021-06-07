// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"os/exec"

	"github.com/bitly/go-simplejson"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type TKGRunner struct {
	ConfigPath string
	Namespace  string
}

func NewTKGRunner(configPath, namespace string) *TKGRunner {
	return &TKGRunner{
		ConfigPath: configPath,
		Namespace:  namespace,
	}
}

func (runner *TKGRunner) RunWithArgs(args ...string) *gexec.Session {
	command := exec.Command("tkg", args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	GinkgoT().Log("running tkg cli with args", args)
	return session
}

func (runner *TKGRunner) RunWithNamespace(namespace string, args ...string) *gexec.Session {
	newArgs := append([]string{"--config", runner.ConfigPath, "--namespace", namespace}, args...)
	return runner.RunWithArgs(newArgs...)
}

func (runner *TKGRunner) Run(args ...string) *gexec.Session {
	newArgs := append([]string{"--config", runner.ConfigPath}, args...)
	newArgs = append(newArgs, []string{"--namespace", runner.Namespace}...)
	return runner.RunWithArgs(newArgs...)
}

func CreateCluster(r *TKGRunner, name string, vip string) {
	Eventually(r.Run("create", "cluster", name, "--plan", "dev", "--controlplane-machine-count", "1", "--vsphere-controlplane-endpoint-ip", vip), "900s").Should(gexec.Exit())
}

func GetClusterCredential(r *TKGRunner, name string) {
	Eventually(r.Run("get", "credentials", name), "30s").Should(gexec.Exit())
}

func DeleteCluster(r *TKGRunner, name string) {
	Eventually(r.Run("delete", "cluster", name, "-y"), "30s").Should(gexec.Exit())
}

func ClusterExists(r *TKGRunner, name string) bool {
	s1 := r.Run("get", "cluster", name, "-o", "json")
	Eventually(s1, "10s").Should(gexec.Exit(0))
	s, err := getClusterStatusFromJson(s1.Out.Contents(), name)
	Expect(err).ToNot(HaveOccurred())
	return s != ""
}

func EnsureClusterGone(r *TKGRunner, name string) {
	EnsureClusterStatusWithTimeout(r, name, "", "300s")
}

func EnsureClusterStatus(r *TKGRunner, name, status string) {
	EnsureClusterStatusWithTimeout(r, name, status, "30s")
}

func EnsureClusterStatusWithTimeout(r *TKGRunner, name, status, timeout string) {
	Eventually(func() bool {
		s1 := r.Run("get", "cluster", name, "-o", "json")
		Eventually(s1, "10s").Should(gexec.Exit(0))
		s, err := getClusterStatusFromJson(s1.Out.Contents(), name)
		Expect(err).ToNot(HaveOccurred())
		return s == status
	}, timeout, "5s").Should(BeTrue())
}

func getClusterStatusFromJson(data []byte, name string) (string, error) {
	var res string
	j, err := simplejson.NewJson(data)
	if err != nil {
		return res, err
	}
	clusters, err := j.Array()
	if err != nil {
		return res, err
	}
	for _, item := range clusters {
		if itemj, ok := item.(map[string]interface{}); ok {
			if names, ok := itemj["name"].(string); ok {
				if names == name {
					if status, ok := itemj["status"].(string); ok {
						return status, nil
					}
				}
			}
		}
	}
	return res, nil
}
