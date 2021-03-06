// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/aviclient"
)

var testEnv TestEnvSpec

func LoadTestEnv(path string) error {
	if NotExist(path) {
		return errors.New("path doesn't exist")
	}
	data, err := ReadFromFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &testEnv)
	if err != nil {
		return err
	}
	return nil
}

// TestEnvSpec is the specification for an e2e tests which includes:
// 1. the infra information where these tests will run;
// 2. an array of test cases that'll be run and the test specific settings;
type TestEnvSpec struct {
	Env   Env
	Tests []TestCaseSpec
}

type Env struct {
	TKGConfig                   string      `json:"tkg-config"`
	ManagementClusterKubeconfig Kubecontext `json:"mc-kubeconfig"`
	Worker                      string      `json:"worker"`
}

type Kubecontext struct {
	Path    string `json:"path"`
	Context string `json:"context"`
}

type TestCaseSpec struct {
	Name                string       `json:"name"`
	AKODeploymentConfig YamlTarget   `json:"akoDeploymentConfig"`
	YAMLs               []YamlTarget `json:"yamls"`
}

type YamlTarget struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// E2ETestCase runs tests case in one separate namespace
type E2ETestCase struct {
	Clients             Clients
	AKODeploymentConfig YamlTarget
	YAMLs               []YamlTarget
}

type Clients struct {
	Kubectl *KubectlRunner
	TKGCli  *TKGRunner
	VIP     *VIPRunner
	Avi     aviclient.Client
}

// Init initializes the namespace
func (o *E2ETestCase) Init() {
	CreateNamespace(o.Clients.Kubectl)
}

// Teardown deletes the namespace
func (o *E2ETestCase) Teardown() {
	DeleteNamespace(o.Clients.Kubectl)
}

type labelGetter func() map[string]string

// LoadTestTest checks if the testcase is registered to run.
// It takes one parameter:
//    string: name of the testcase
// It returns two values:
//    bool: if it's true, then the test case is not registered and should be skipped
//    *E2ETestCase: an encapsulation of a Test Case's env
func LoadTestCase(name string) (bool, *E2ETestCase) {
	namespace := "akoo-e2e-" + GenerateRandomName()
	res := &E2ETestCase{
		Clients: Clients{
			Kubectl: NewKubectlRunner(testEnv.Env.ManagementClusterKubeconfig.Path, testEnv.Env.ManagementClusterKubeconfig.Context, namespace),
			TKGCli:  NewTKGRunner(testEnv.Env.TKGConfig, namespace),
			VIP:     NewVIPRunner(testEnv.Env.Worker),
			Avi:     nil,
		},
	}
	for _, test := range testEnv.Tests {
		if test.Name == name {
			res.AKODeploymentConfig = test.AKODeploymentConfig
			res.YAMLs = test.YAMLs
			return false, res
		}
	}
	return true, res
}

func GenerateRandomName() string {
	rand.Seed(time.Now().UnixNano())
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
