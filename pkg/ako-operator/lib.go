// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// Legacy cluster environment variables
const (
	// DeployInBootstrapCluster - defines if ako operator is deployed in bootstrap cluster
	DeployInBootstrapCluster = "bootstrap_cluster"

	// IsControlPlaneHAProvider - defines if ako operator is going to provide control plane HA
	IsControlPlaneHAProvider = "avi_control_plane_ha_provider"

	// ClusterControlPlaneAnnotations - defines cluster control plane endpoint
	ClusterControlPlaneAnnotations = "tkg.tanzu.vmware.com/cluster-controlplane-endpoint"

	// ControlPlaneEndpointPort - defines the control plane endpoint port
	ControlPlaneEndpointPort = "control_plane_endpoint_port"
)

// ClusterClass Env variables
const (
	// ClusterClassEnabled - helps check if cluster is classy based cluster when no cluster object create yet.
	ClusterClassEnabled = "cluster_class_enabled"

	// KubeVipLoadBalancerProvider - defines if cluster using kube-vip to implement load balancer
	// type of service
	KubeVipLoadBalancerProvider = "kubeVipLoadBalancerProvider"

	// AviAPIServerHAProvider - defines if ako operator is going to provide control plane HA
	AviAPIServerHAProvider = "aviAPIServerHAProvider"

	// ApiServerPort - defines the control plane endpoint
	ApiServerEndpoint = "apiServerEndpoint"

	// ApiServerPort - defines the control plane endpoint port
	ApiServerPort = "apiServerPort"
)

func IsBootStrapCluster() bool {
	return os.Getenv(DeployInBootstrapCluster) == "True"
}

func IsClusterClassEnabled() bool {
	return os.Getenv(ClusterClassEnabled) == "True"
}

// IsClusterClassBasedCluster checks if a cluster is cluster class based cluster
func IsClusterClassBasedCluster(cluster *clusterv1.Cluster) bool {
	if cluster != nil && cluster.Spec.Topology != nil {
		return true
	}
	return false
}

// IsControlPlaneVIPProvider checks if NSX Advanced Load Balancer is cluster's endpoint VIP provider
func IsControlPlaneVIPProvider(cluster *clusterv1.Cluster) (bool, error) {
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == AviAPIServerHAProvider {
				var aviAPIServerHAProvider bool
				if err := json.Unmarshal(clusterVariable.Value.Raw, &aviAPIServerHAProvider); err != nil {
					return aviAPIServerHAProvider, err
				}
				return aviAPIServerHAProvider, nil
			}
		}
	}
	return os.Getenv(IsControlPlaneHAProvider) == "True", nil
}

// IsLoadBalancerProvider checks if NSX Advanced Load Balancer is cluster's load balancer implementation
// default value is true
func IsLoadBalancerProvider(cluster *clusterv1.Cluster) (bool, error) {
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == KubeVipLoadBalancerProvider {
				var kubeVipLoadBalancerProvider bool
				if err := json.Unmarshal(clusterVariable.Value.Raw, &kubeVipLoadBalancerProvider); err != nil {
					return true, err
				}
				return !kubeVipLoadBalancerProvider, nil
			}
		}
	}
	return true, nil
}

// GetControlPlaneEndpoint returns cluster's API server address
func GetControlPlaneEndpoint(cluster *clusterv1.Cluster) (string, error) {
	apiServerEndpoint, _ := cluster.ObjectMeta.Annotations[ClusterControlPlaneAnnotations]
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == ApiServerEndpoint {
				if err := json.Unmarshal(clusterVariable.Value.Raw, &apiServerEndpoint); err != nil {
					return apiServerEndpoint, err
				}
				return apiServerEndpoint, nil
			}
		}
	}
	return apiServerEndpoint, nil
}

// SetControlPlaneEndpoint sets cluster.spec.topology.variables.apiServerEndpoint
func SetControlPlaneEndpoint(cluster *clusterv1.Cluster, endpoint string) {
	if IsClusterClassBasedCluster(cluster) {
		// endpoint is a string, json.Marshal will never throw error
		raw, _ := json.Marshal(endpoint)
		for i, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == ApiServerEndpoint {
				cluster.Spec.Topology.Variables[i].Value = apiextensionsv1.JSON{Raw: raw}
				return
			}
		}
		endpointVar := &clusterv1.ClusterVariable{
			Name:  ApiServerEndpoint,
			Value: apiextensionsv1.JSON{Raw: raw},
		}
		cluster.Spec.Topology.Variables = append(cluster.Spec.Topology.Variables, *endpointVar)
	}
}

// validatePortNumber checks if given number is a valid port number
func validatePortNumber(number int) bool {
	return number > 0 && number < 65536
}

// GetControlPlaneEndpointPort returns cluster's API server port
// default value is 6443
func GetControlPlaneEndpointPort(cluster *clusterv1.Cluster) (int32, error) {
	apiServerPort := 6443
	if IsClusterClassBasedCluster(cluster) {
		for _, clusterVariable := range cluster.Spec.Topology.Variables {
			if clusterVariable.Name == ApiServerPort {
				err := json.Unmarshal(clusterVariable.Value.Raw, &apiServerPort)
				if err != nil {
					return 6443, err
				}
				if !validatePortNumber(apiServerPort) {
					return 6443, fmt.Errorf("port number %d is not in valid range [1,65535]", apiServerPort)
				}
				return int32(apiServerPort), nil
			}
		}
		return int32(apiServerPort), nil
	} else {
		if port, ok := os.LookupEnv(ControlPlaneEndpointPort); ok {
			apiServerPort, err := strconv.Atoi(port)
			if err != nil {
				return 6443, err
			}
			if !validatePortNumber(apiServerPort) {
				return 6443, fmt.Errorf("port number %d is not in valid range [1,65535]", apiServerPort)
			}
			return int32(apiServerPort), nil
		} else {
			return 6443, nil
		}
	}
}
