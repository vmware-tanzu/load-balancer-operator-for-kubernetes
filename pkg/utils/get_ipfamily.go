// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"fmt"
	"net"

	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	InvalidIPFamily = "INVALID"
	IPv4IpFamily    = "V4"
	IPv6IpFamily    = "V6"
	DualStackIPFamily = "Dual-Stack"
	DualStackIPv6Primary = "V6,V4"
	DualStackIPv4Primary = "V4,V6"
)

func GetIPFamilyFromCidr(cidr string) string {
	addr, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return InvalidIPFamily
	}

	addrType := IPv4IpFamily
	if addr.To4() == nil {
		addrType = IPv6IpFamily
	}

	return addrType
}

func ipFamilyFromCIDRStrings(cidrs []string) (string, error) {
	if len(cidrs) > 2 {
		return InvalidIPFamily, errors.New("too many CIDRs specified")
	}
	var foundIPv4 bool
	var foundIPv6 bool
	for _, cidr := range cidrs {
		cidrType := GetIPFamilyFromCidr(cidr)
		if cidrType == IPv4IpFamily {
			foundIPv4 = true
		} else {
			foundIPv6 = true
		}
	}
	switch {
	case foundIPv4 && foundIPv6:
		return DualStackIPFamily, nil
	case foundIPv4:
		return IPv4IpFamily, nil
	case foundIPv6:
		return IPv6IpFamily, nil
	default:
		return InvalidIPFamily, nil
	}
}

// GetClusterIPFamily returns a cluster IPFamily from the configuration provided.
func GetClusterIPFamily(c *capi.Cluster) (string, error) {
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
		podsIPFamily, err = ipFamilyFromCIDRStrings(podCIDRs)
		if err != nil {
			return InvalidIPFamily, fmt.Errorf("pods: %s", err)
		}
	}

	if len(serviceCIDRs) != 0 {
		servicesIPFamily, err = ipFamilyFromCIDRStrings(serviceCIDRs)
		if err != nil {
			return InvalidIPFamily, fmt.Errorf("services: %s", err)
		}
	}

	if podsIPFamily != servicesIPFamily && len(podCIDRs) != 0 && len(serviceCIDRs) != 0{
		return InvalidIPFamily, errors.New("pods and services IP family mismatch")
	}

	if podsIPFamily == DualStackIPFamily || servicesIPFamily == DualStackIPFamily {
		if podsIPFamily == DualStackIPFamily{
			podCIDRType := GetIPFamilyFromCidr(podCIDRs[0])
			if podCIDRType == IPv4IpFamily{
				return DualStackIPv4Primary, nil
			} else {
				return DualStackIPv6Primary, nil
			}
		}
		serviceCIDRType := GetIPFamilyFromCidr(serviceCIDRs[0])
		if serviceCIDRType == IPv4IpFamily{
			return DualStackIPv4Primary, nil
		} else {
			return DualStackIPv6Primary, nil
		}
	}

	if len(podCIDRs) == 0 {
		return servicesIPFamily, nil
	}
	return podsIPFamily, nil
}