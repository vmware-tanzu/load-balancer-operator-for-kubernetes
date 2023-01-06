// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package netprovider

import (
	"strings"

	"github.com/go-logr/logr"
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

func (c *UsableNetworkProvider) AddUsableNetwork(client aviclient.Client, cloudName, networkName string, log logr.Logger) error {
	cloud, err := client.CloudGetByName(cloudName)
	if err != nil {
		// Cannot find the configured cloud, requeue the request but
		// leave enough time for operators to resolve this issue
		return errors.Wrapf(err, "Failed to find cloud %s, requeue the request\n", cloudName)
	}
	if cloud.IPAMProviderRef == nil {
		// Cannot find any configured IPAM Provider, requeue the request but
		// leave enough time for operators to resolve this issue
		return errors.Wrap(err, "No IPAM Provider is registered for the cloud, requeue the request")
	}
	ipamProviderUUID := aviclient.GetUUIDFromRef(*(cloud.IPAMProviderRef))
	ipam, err := client.IPAMDNSProviderProfileGet(ipamProviderUUID)
	if err != nil {
		return errors.Wrap(err, "Failed to find IPAM profile")
	}
	network, err := client.NetworkGetByName(networkName, cloudName)
	if err != nil {
		return errors.Wrapf(err, "Failed to get Data Network %s from AVI Controller\n", networkName)
	}
	// Ensure network is added to the cloud's IPAM Profile as one of its
	// usable Networks
	// sample network url: https://1.1.1.1/api/network/network-38-cloud-c654fba6-6486-4595-911d-52a8f1fbbf77#testnetwork
	// sample usable network ref: https://1.1.1.1/api/network/network-38-cloud-c654fba6-6486-4595-911d-52a8f1fbbf77
	for _, usableNetwork := range ipam.InternalProfile.UsableNetworks {
		if strings.Contains(*(network.URL), *(usableNetwork.NwRef)) {
			log.Info("Network is already one of the cloud's usable network", "network", networkName)
			return nil
		}
	}
	ipam.InternalProfile.UsableNetworks = append(ipam.InternalProfile.UsableNetworks, &models.IPAMUsableNetwork{NwRef: network.URL})
	_, err = client.IPAMDNSProviderProfileUpdate(ipam)
	if err != nil {
		return errors.Wrapf(err, "Failed to add usable network %s\n", *network.Name)
	}

	log.Info("Added Usable Network", "network", networkName)
	return nil
}
