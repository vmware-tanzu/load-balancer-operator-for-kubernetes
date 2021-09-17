// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func ShowThePlan() {
	GinkgoT().Logf("\nRunning AVI End To End Tests\n")
}

func (o *E2ETestCase) SanityCheck() {
	By("Ensure AKO Operator running")
	EnsurePodRunning(o.Clients.Kubectl, "ako-operator-controller-manager", 1, "tkg-system-networking")
}

func (o *E2ETestCase) EnsureYamlsApplied(yamlPaths []string) {
	for _, path := range yamlPaths {
		Eventually(o.Clients.Kubectl.RunWithoutNamespace("apply", "-f", path), "5s").Should(gexec.Exit())
	}
}

func (o *E2ETestCase) EnsureYamlsRemoved(yamlPaths []string) {
	for _, path := range yamlPaths {
		Eventually(o.Clients.Kubectl.RunWithoutNamespace("delete", "-f", path), "5s").Should(gexec.Exit())
	}
}

func (o *E2ETestCase) EnsureClusterCreated(name string) {
	if !ClusterExists(o.Clients.TKGCli, name) {
		By("Allocating an VIP")
		vip, err := AllocVIP(o.Clients.VIP)
		Expect(err).NotTo(HaveOccurred())
		By(fmt.Sprintf("creating the cluster with VIP: %s", vip))
		CreateCluster(o.Clients.TKGCli, name, vip)
		GetClusterCredential(o.Clients.TKGCli, name)
	}
	By("ensuring the cluster is running")
	EnsureClusterStatus(o.Clients.TKGCli, name, "running")
}

func (o *E2ETestCase) EnsureClusterDeleted(name string) {
	runner := o.Clients.TKGCli
	if ClusterExists(runner, name) {
		By("Deleting the cluster")
		DeleteCluster(runner, name)
		By("ensuring the cluster is gone")
		EnsureClusterGone(runner, name)
	}
}

func (o *E2ETestCase) EnsureClusterLabelApplied(clusterName string, labelGetter labelGetter) {
	labels := labelGetter()
	for k, v := range labels {
		By(fmt.Sprintf("Applying labels k:%s, v:%s to cluster %s", k, v, clusterName))
		ApplyLabelOnCluster(o.Clients.Kubectl, clusterName, k, v)
	}
	EnsureClusterHasLabels(o.Clients.Kubectl, clusterName, labels)
}

func EnsureAKO(testcase *E2ETestCase, clusterName string) {
	wcRunner := NewKubectlRunner(
		// by setting kubeConfigPath to empty, use $HOME/.kube/config by default
		"",
		fmt.Sprintf("%s-admin@%s", clusterName, clusterName),
		testcase.Clients.Kubectl.Namespace)
	EnsurePodRunningWithTimeout(wcRunner, "ako-0", 1, "tkg-system-networking", "180s")
}

func EnsureLoadBalancerService(testcase *E2ETestCase, clusterName string) {
	var paths []string
	for _, p := range testcase.YAMLs {
		paths = append(paths, p.Path)
	}
	wcRunner := NewKubectlRunner(
		// by setting kubeConfigPath to empty, use $HOME/.kube/config by default
		"",
		fmt.Sprintf("%s-admin@%s", clusterName, clusterName),
		testcase.Clients.Kubectl.Namespace)
	EnsureYamlsApplied(wcRunner, paths)
	EnsureLoadBalancerTypeServiceAccessible(wcRunner, 1)
}

func (o *E2ETestCase) EnsureCRSandAviUserDeleted(clusterName string) {
	// Check crs
	EnsureObjectGone(o.Clients.Kubectl, "secret", clusterName+"-ako")
	EnsureObjectGone(o.Clients.Kubectl, "ClusterResourceSet", clusterName+"-ako")
	// Check Avi user
	EnsureObjectGone(o.Clients.Kubectl, "secret", clusterName+"-avi-credentials")
}

func (o *E2ETestCase) EnsureAviResourcesDeleted(clusterName string) {
	o.Clients.Avi = NewAviRunner(o.Clients.Kubectl)
	// Avi Resources are regarded as deleted if VirtualService and Pool are deleted
	EnsureAviObjectDeleted(o.Clients.Avi, clusterName, "virtualservice")
	EnsureAviObjectDeleted(o.Clients.Avi, clusterName, "pool")
}
