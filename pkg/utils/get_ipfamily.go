// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"net"
)

const (
	InvalidIPFamily = "INVALID"
	IPv4IpFamily    = "V4"
	IPv6IpFamily    = "V6"
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
