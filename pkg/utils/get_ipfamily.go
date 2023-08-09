// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"fmt"
	"net"

	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	InvalidIPFamily   = "INVALID"
	IPv4IpFamily = "V4"
	IPv6IpFamily = "V6"
)

// GetIPFamily returns a cluster primary IPFamily from the configuration provided.
func GetPrimaryIPFamily(c *capi.Cluster) (string, error) {
	var podCIDRs, serviceCIDRs []string
	var podsIPFamily, servicesIPFamily string
	var err error

	if c.Spec.ClusterNetwork != nil {
		if c.Spec.ClusterNetwork.Pods != nil {
			podCIDRs = c.Spec.ClusterNetwork.Pods.CIDRBlocks
		}
		if c.Spec.ClusterNetwork.Services != nil {
			serviceCIDRs = c.Spec.ClusterNetwork.Services.CIDRBlocks
		}
	}
	if len(podCIDRs) == 0 && len(serviceCIDRs) == 0 {
		return IPv4IpFamily, nil
	}

	if len(podCIDRs) != 0 {
		podsIPFamily, err = ipFamilyForCIDRStrings(podCIDRs)
		if err != nil {
			return InvalidIPFamily, fmt.Errorf("pods: %s", err)
		}
	}

	if len(serviceCIDRs) != 0 {
		servicesIPFamily, err = ipFamilyForCIDRStrings(serviceCIDRs)
		if err != nil {
			return InvalidIPFamily, fmt.Errorf("services: %s", err)
		}
	}

	if podsIPFamily != servicesIPFamily && len(podCIDRs) != 0 && len(serviceCIDRs) != 0{
		return InvalidIPFamily, errors.New("pods and services IP family mismatch")
	}

	if len(podCIDRs) == 0 {
		return servicesIPFamily, nil
	}
	return podsIPFamily, nil
}

//ipFamilyForCIDRStrings returns primary ip family
func ipFamilyForCIDRStrings(cidrs []string) (string, error) {
	if len(cidrs) > 2 {
		return InvalidIPFamily, errors.New("too many CIDRs specified")
	}

	ip, _, err := net.ParseCIDR(cidrs[0])
	if err != nil {
		return InvalidIPFamily, fmt.Errorf("could not parse CIDR: %s", err)
	}

	if ip.To4() != nil {
		return IPv4IpFamily, nil
	}
	return IPv6IpFamily, nil

}
