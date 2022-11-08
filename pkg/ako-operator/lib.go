// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"encoding/json"
	"os"
	"strconv"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Environment variables
const (
	// legacy cluster environment variables
	// DeployInBootstrapCluster - defines if ako operator is deployed in bootstrap cluster
	DeployInBootstrapCluster = "bootstrap_cluster"

	// IsControlPlaneHAProvider - defines if ako operator is going to provide control plane HA
	IsControlPlaneHAProvider = "avi_control_plane_ha_provider"

	// ManagementClusterName - defines the management cluster name ako operator running in
	ManagementClusterName = "tkg_management_cluster_name"

	// ControlPlaneEndpointPort - defines the control plane endpoint port
	ControlPlaneEndpointPort = "control_plane_endpoint_port"

	// cluster class cluster environment variables
	// KubeVipLoadBalancerProvider - defines if cluster using kube-vip to implement load balancer
	// type of service
	KubeVipLoadBalancerProvider = "kubeVipLoadBalancerProvider"

	// AviAPIServerHAProvider - defines if ako operator is going to provide control plane HA
	AviAPIServerHAProvider = "aviAPIServerHAProvider"

	// ApiServerPort - defines the control plane endpoint port
	ApiServerPort = "apiServerPort"
)

func IsBootStrapCluster() bool {
	return os.Getenv(DeployInBootstrapCluster) == "True"
}

// IsClusterClassBasedCluster checks if a cluster is cluster class based cluster
func IsClusterClassBasedCluster(cluster *clusterv1.Cluster) bool {
	if cluster != nil && cluster.Spec.Topology != nil {
		return true
	}
	return false
}

func IsControlPlaneVIPProvider(cluster *clusterv1.Cluster) bool {
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == AviAPIServerHAProvider {
				var aviAPIServerHAProvider bool
				if err := json.Unmarshal(clusterVariable.Value.Raw, &aviAPIServerHAProvider); err == nil {
					return aviAPIServerHAProvider
				}
			}
		}
		return false
	} else {
		return os.Getenv(IsControlPlaneHAProvider) == "True"
	}
}

func IsLoadBalancerProvider(cluster *clusterv1.Cluster) bool {
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == KubeVipLoadBalancerProvider {
				var kubeVipLoadBalancerProvider bool
				if err := json.Unmarshal(clusterVariable.Value.Raw, &kubeVipLoadBalancerProvider); err == nil {
					return !kubeVipLoadBalancerProvider
				}
			}
		}
	}
	return true
}

func validatePortNumber(number int) bool {
	return number > 0 && number < 65536
}

func GetControlPlaneEndpointPort(cluster *clusterv1.Cluster) int32 {
	apiServerPort := 6443
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == ApiServerPort {
				err := json.Unmarshal(clusterVariable.Value.Raw, &apiServerPort)
				if err != nil || !validatePortNumber(apiServerPort) {
					return 6443
				}
			}
		}
		return int32(apiServerPort)
	} else {
		apiServerPort, err := strconv.Atoi(os.Getenv(ControlPlaneEndpointPort))
		println("apiServerPort")
		println(apiServerPort)
		if err != nil || !validatePortNumber(apiServerPort) {
			return 6443
		}
		return int32(apiServerPort)
	}
}
