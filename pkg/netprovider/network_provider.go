// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package netprovider

import (
	"github.com/pkg/errors"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware/alb-sdk/go/models"
)

type UsableNetworks []UsableNetwork

type UsableNetwork struct {
	NetworkName string `json:"networkName"`
	CIDR        string `json:"cidr"`
}

type UsableNetworkProvider struct{}

func (c *UsableNetworkProvider) AddUsableNetwork(client aviclient.Client, cloudName, networkName string) (bool, error) {
	network, err := client.NetworkGetByName(networkName)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to get Data Network %s from AVI Controller\n", networkName)
	}
	cloud, err := client.CloudGetByName(cloudName)
	if err != nil {
		// Cannot find the configured cloud, requeue the request but
		// leave enough time for operators to resolve this issue
		return false, errors.Wrapf(err, "Failed to find cloud %s, requeue the request\n", cloudName)
	}
	if cloud.IPAMProviderRef == nil {
		// Cannot find any configured IPAM Provider, requeue the request but
		// leave enough time for operators to resolve this issue
		return false, errors.Wrap(err, "No IPAM Provider is registered for the cloud, requeue the request")
	}
	ipamProviderUUID := aviclient.GetUUIDFromRef(*(cloud.IPAMProviderRef))
	ipam, err := client.IPAMDNSProviderProfileGet(ipamProviderUUID)
	if err != nil {
		return false, errors.Wrap(err, "Failed to find IPAM profile")
	}

	// Ensure network is added to the cloud's IPAM Profile as one of its
	// usable Networks
	for _, usableNetwork := range ipam.InternalProfile.UsableNetworks {
		if *usableNetwork.NwRef == *(network.URL) {
			return false, nil
		}
	}
	ipam.InternalProfile.UsableNetworks = append(ipam.InternalProfile.UsableNetworks, &models.IPAMUsableNetwork{NwRef: network.URL})
	_, err = client.IPAMDNSProviderProfileUpdate(ipam)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to add usable network %s\n", *network.Name)
	}
	return true, nil
}
