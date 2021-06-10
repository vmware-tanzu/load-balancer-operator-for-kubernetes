// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"os"
	"strconv"
)

// Environment variables
const (
	// DeployInBootstrapCluster - defines if ako operator is deployed in bootstrap cluster
	DeployInBootstrapCluster = "bootstrap_cluster"

	// IsControlPlaneHAProvider - defines if ako operator is going to provide control plane HA
	IsControlPlaneHAProvider = "avi_control_plane_ha_provider"

	// ManagementClusterName - defines the management cluster name ako operator running in
	ManagementClusterName = "tkg_management_cluster_name"

	// ControlPlaneEndpointPort - defines the control plane endpoint port
	ControlPlaneEndpointPort = "control_plane_endpoint_port"
)

func IsBootStrapCluster() bool {
	return os.Getenv(DeployInBootstrapCluster) == "True"
}

func IsHAProvider() bool {
	return os.Getenv(IsControlPlaneHAProvider) == "True"
}

func GetControlPlaneEndpointPort() int32 {
	port, err := strconv.Atoi(os.Getenv(ControlPlaneEndpointPort))
	if err != nil || port <= 0 {
		return 6443
	}
	return int32(port)
}
