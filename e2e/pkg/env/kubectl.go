// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bitly/go-simplejson"
	homedir "github.com/mitchellh/go-homedir"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type KubectlRunner struct {
	ConfigPath string
	Context    string
	Namespace  string
	Timeout    string
}

func NewKubectlRunner(kubeConfigPath, context, namespace string) *KubectlRunner {
	if kubeConfigPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			GinkgoT().Logf("Cannot get home directory: %s\n", err.Error())
		}
		kubeConfigPath = home + "/.kube/config"
	}
	return &KubectlRunner{
		ConfigPath: kubeConfigPath,
		Context:    context,
		Namespace:  namespace,
		Timeout:    "60s",
	}
}

func (runner *KubectlRunner) RunWithArgs(args ...string) *gexec.Session {
	command := exec.Command("kubectl", args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	GinkgoT().Log("running kubectl with args", args)
	return session
}

func (runner *KubectlRunner) RunWithoutNamespace(args ...string) *gexec.Session {
	newArgs := append([]string{"--kubeconfig", runner.ConfigPath, "--context", runner.Context}, args...)
	return runner.RunWithArgs(newArgs...)
}

func (runner *KubectlRunner) RunInNamespace(namespace string, args ...string) *gexec.Session {
	newArgs := append([]string{"--kubeconfig", runner.ConfigPath, "--namespace", namespace, "--context", runner.Context}, args...)
	return runner.RunWithArgs(newArgs...)
}

func (runner *KubectlRunner) Run(args ...string) *gexec.Session {
	return runner.RunInNamespace(runner.Namespace, args...)
}

func CreateNamespace(r *KubectlRunner) {
	Eventually(r.RunWithoutNamespace("create", "namespace", r.Namespace), "5s").Should(gexec.Exit(0))
}

func DeleteNamespace(r *KubectlRunner) {
	Eventually(r.RunWithoutNamespace("delete", "namespace", r.Namespace), "30s").Should(gexec.Exit(0))
}

func ApplyLabelOnCluster(r *KubectlRunner, name, key, val string) {
	Eventually(r.Run("label", "cluster", name, fmt.Sprintf("%s=%s", key, val), "--overwrite"), "10s").Should(gexec.Exit(0))
}

func EnsureYamlsApplied(runner *KubectlRunner, yamlPaths []string) {
	for _, path := range yamlPaths {
		Eventually(runner.RunWithoutNamespace("apply", "-f", path), "5s").Should(gexec.Exit())
	}
}

func EnsureClusterHasLabels(runner *KubectlRunner, name string, labels map[string]string) {
	// TODO(fangyuanl): implement this
	GinkgoT().Logf("To be implemented")
}

func AKODeploymentConfigLabelsGetter(testcase *E2ETestCase) labelGetter {
	name := testcase.AKODeploymentConfig.Name
	runner := testcase.Clients.Kubectl
	return func() map[string]string {
		s1 := runner.RunWithoutNamespace("get", "akodeploymentconfig", name, "-o", "json")
		Eventually(s1, "10s", "1s").Should(gexec.Exit(0))

		labels, err := getSelectorFromJson(s1.Out.Contents())
		Expect(err).NotTo(HaveOccurred())

		return labels
	}
}

func EnsurePodRunningWithTimeout(runner *KubectlRunner, podNamePrefix string, expectedNum int, namespace, timeout string) {
	Eventually(func() int {
		statuses := GetPodsStatuses(runner, podNamePrefix, expectedNum, namespace)
		var res []string
		for _, s := range statuses {
			if s == "Running" {
				res = append(res, s)
			}
		}
		return len(res)
	}, timeout, "5s").Should(Equal(expectedNum))
}

func EnsurePodRunning(runner *KubectlRunner, podNamePrefix string, expectedNum int, namespace string) {
	EnsurePodRunningWithTimeout(runner, podNamePrefix, expectedNum, namespace, "30s")
}

func EnsureLoadBalancerTypeServiceAccessible(runner *KubectlRunner, expectedNum int) {
	// Only process LB type SVC in default namespace for now
	namespace := "default"
	port := 80
	lbSvcIPs := GetLoadBalancerServices(runner, expectedNum, namespace)
	for _, ip := range lbSvcIPs {
		EnsureIPAccessible(ip, port)
	}
}

func GetLoadBalancerServices(runner *KubectlRunner, expectedNum int, namespace string) []string {
	var res []string
	Eventually(func() int {
		s1 := runner.RunInNamespace(namespace, "get", "services", "-o", "json")
		// default polling interval is 10*millisecond which is too short.
		// keeps polling until kubectl command returns successfully
		// polling here is to overcome networking jitter or apiserver latency
		Eventually(s1, "10s", "1s").Should(gexec.Exit(0))

		r, err := getLoadBalancerTypeServiceIPsFromJson(s1.Out.Contents())
		Expect(err).NotTo(HaveOccurred())
		// return list of pod statuses, no guarantee on list length
		res = r
		return len(r)
		// 5 minutes timeout wait for ip available
	}, "300s", "5s").Should(Equal(expectedNum))
	return res
}

func EnsureIPAccessible(ip string, port int) {
	Eventually(func() error {
		resp, err := http.Get(fmt.Sprintf("http://%s:%d", ip, port))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		GinkgoT().Logf(string(body[:]))
		return err
		// 15 minutes timeout because Service Engine might need to be
		// created
	}, "900s", "5s").ShouldNot(HaveOccurred())
}

func GetPodsStatuses(runner *KubectlRunner, podNamePrefix string, expectedNum int, namespace string) []string {
	var res []string
	// poll every 5 seconds to make sure returned number of pod statuses matches expected number
	// total timeout is decided by expected number of pods: accumulative 1 minute per pod
	timeout := expectedNum * 30
	Eventually(func() int {
		s1 := runner.RunInNamespace(namespace, "get", "pods", "-o", "json")
		// default polling interval is 10*millisecond which is too short.
		// keeps polling until kubectl command returns successfully
		// polling here is to overcome networking jitter or apiserver latency
		Eventually(s1, "30s", "1s").Should(gexec.Exit(0))

		r, err := getPodsStatusesFromJson(s1.Out.Contents(), podNamePrefix)
		Expect(err).NotTo(HaveOccurred())
		// return list of pod statuses, no guarantee on list length
		res = r
		return len(r)
	}, strconv.Itoa(timeout)+"s", "5s").Should(Equal(expectedNum))
	return res
}

func getPodsStatusesFromJson(b []byte, podNamePrefix string) ([]string, error) {
	var res []string
	j, err := simplejson.NewJson(b)
	if err != nil {
		return res, err
	}
	items, err := j.Get("items").Array()
	if err != nil {
		return res, err
	}
	for _, item := range items {
		if itemj, ok := item.(map[string]interface{}); ok {
			if metadataj, ok := itemj["metadata"].(map[string]interface{}); ok {
				if names, ok := metadataj["name"].(string); ok {
					if strings.HasPrefix(names, podNamePrefix) {
						if statusj, ok := itemj["status"].(map[string]interface{}); ok {
							if phases, ok := statusj["phase"].(string); ok {
								res = append(res, phases)
							}
						}
					}
				}
			}
		}
	}
	return res, nil
}

func getSelectorFromJson(b []byte) (map[string]string, error) {
	j, err := simplejson.NewJson(b)
	if err != nil {
		return nil, err
	}
	if itemj, ok := j.Interface().(map[string]interface{}); ok {
		if specj, ok := itemj["spec"].(map[string]interface{}); ok {
			if clusterSelectorj, ok := specj["clusterSelector"].(map[string]interface{}); ok {
				if matchLabels, ok := clusterSelectorj["matchLabels"].(map[string]interface{}); ok {
					res := make(map[string]string)
					for k, v := range matchLabels {
						if vs, ok := v.(string); ok {
							res[k] = vs
						}
					}
					return res, nil
				}
			}
		}
	}
	return nil, nil
}

func getLoadBalancerTypeServiceIPsFromJson(b []byte) ([]string, error) {
	var res []string
	j, err := simplejson.NewJson(b)
	if err != nil {
		return res, err
	}
	items, err := j.Get("items").Array()
	if err != nil {
		return res, err
	}
	for _, item := range items {
		if itemj, ok := item.(map[string]interface{}); ok {
			if statusj, ok := itemj["status"].(map[string]interface{}); ok {
				if loadBalancerj, ok := statusj["loadBalancer"].(map[string]interface{}); ok {
					if ingressj, ok := loadBalancerj["ingress"].([]interface{}); ok {
						for _, ingressi := range ingressj {
							if ingress, ok := ingressi.(map[string]interface{}); ok {
								if ip, ok := ingress["ip"].(string); ok {
									res = append(res, ip)
								}
							}
						}
					}
				}
			}
		}
	}
	return res, nil
}

func EnsureObjectGone(runner *KubectlRunner, obj, objName string) {

	Eventually(func() error {
		s := runner.RunInNamespace(runner.Namespace, "get", obj, "-o", "json")
		Eventually(s, "10s", "2s").Should(gexec.Exit(0))
		r, err := ensureObjectNotFound(s.Out.Contents(), objName)
		if err != nil {
			return err
		}
		Expect(r).Should(BeFalse())
		return nil
	}, "30s", "5s").ShouldNot(HaveOccurred())
}

func ensureObjectNotFound(b []byte, objName string) (bool, error) {
	j, err := simplejson.NewJson(b)
	if err != nil {
		return false, err
	}
	items, err := j.Get("items").Array()
	if err != nil {
		return false, err
	}
	if len(items) == 0 {
		return false, nil
	}
	for _, item := range items {
		if itemj, ok := item.(map[string]interface{}); ok {
			if specj, ok := itemj["metadata"].(map[string]interface{}); ok {
				if objName == specj["name"] {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func GetAviObject(runner *KubectlRunner, resourceType, resourceName, field, obj string) string {
	s1 := runner.RunWithoutNamespace("get", resourceType, resourceName, "-o", "json")
	Eventually(s1, "10s", "1s").Should(gexec.Exit(0))

	j, err1 := simplejson.NewJson(s1.Out.Contents())
	Expect(err1).NotTo(HaveOccurred())
	encodedvalue, err2 := j.Get(field).Get(obj).String()
	Expect(err2).NotTo(HaveOccurred())

	if obj == "controller" {
		GinkgoT().Logf("avi controller ip for creating avi client: " + encodedvalue)
		return encodedvalue
	}
	value, err3 := base64.StdEncoding.DecodeString(encodedvalue)
	Expect(err3).NotTo(HaveOccurred())
	GinkgoT().Logf(obj + " for creating avi client: " + string(value))
	return string(value)
}
