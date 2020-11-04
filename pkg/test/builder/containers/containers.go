// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package containers

import (
	"fmt"
	"strings"

	//nolint
	. "github.com/onsi/ginkgo"
	//nolint
	. "github.com/onsi/gomega"

	testutil "gitlab.eng.vmware.com/tkg/load-balancer-api/test/util"
	"gitlab.eng.vmware.com/tkg/load-balancer-api/util/rand"
)

// DockerContainer describes the config we use to manage a docker container
type DockerContainer struct {
	args []string
	name string
}

// Container is an interface for basic operations on docker containers
type Container interface {
	Run()
	Kill()
	IP() string
}

// Run runs the container using provided args
func (d DockerContainer) Run() {
	Expect(testutil.RunDocker(d.args...)).To(Succeed(), fmt.Sprintf("Run docker container %s", d.name))
}

// Kill kills the container
func (d DockerContainer) Kill() {
	By(fmt.Sprintf("shutdown docker container %s", d.name))
	Expect(testutil.RunDocker("kill", d.name)).To(Succeed(), fmt.Sprintf("shutdown docker container %s", d.name))
}

// IP returns the container IP Address
func (d DockerContainer) IP() string {
	args := []string{
		"inspect", d.name, "-f", `'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'`,
	}
	out, err := testutil.RunDockerWithStdOut(args...)
	Expect(err).To(BeNil())
	return strings.Trim(out, "'")
}

// HTTPEcho returns the args for a simple echo server that could be used in
// tests as backend servers
func HTTPEcho(msg string, port int) DockerContainer {
	name := rand.Hash(7)
	return DockerContainer{
		args: []string{
			"run", "-d", "--name", name, "--rm", "hashicorp/http-echo", "-listen", fmt.Sprintf(":%d", port), "-text", msg,
		},
		name: name,
	}
}

// CurlInDockerNetwork spins up a container in Docker to query the provided
// address and returns if the result matches the expected message
func CurlInDockerNetwork(ip string, expected string) (bool, error) {
	args := []string{
		"run", "--rm", "photon", "curl", "-sSm", "1", fmt.Sprintf("http://%s", ip),
	}
	out, err := testutil.RunDockerWithStdOut(args...)
	return out == expected, err
}
