/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// AKODeploymentConfigSpec defines the desired state of an AKODeploymentConfig
// AKODeploymentConfig describes the shared configurations for AKO deployments across a set
// of Clusters.
type AKODeploymentConfigSpec struct {
	// CloudName speficies the AVI Cloud AKO will be deployed with
	CloudName string `json:"cloudName"`

	// Controller is the AVI Controller endpoint to which AKO talks to
	// provision Load Balancer resources
	// The format is [scheme://]address[:port]
	// * scheme                     http or https, defaults to https if not
	//                              specified
	// * address                    IP address of the AVI Controller
	//                              specified
	// * port                       if not specified, use default port for
	//                              the corresponding scheme
	Controller string `json:"controller"`

	// Label selector for Clusters. The Clusters that are
	// selected by this will be the ones affected by this
	// AKODeploymentConfig.
	// It must match the Cluster labels. This field is immutable.
	// By default AviClusterLabel is used
	// +optional
	ClusterSelector metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// WorkloadCredentialRef points to a Secret resource which includes the username
	// and password to access and configure the Avi Controller.
	//
	// * username                   Username used with basic authentication for
	//                              the Avi REST API
	// * password                   Password used with basic authentication for
	//                              the Avi REST API
	//
	// This field is optional. When it's not specified, username/password
	// will be automatically generated for each Cluster and Tenant needs to
	// be non-nil in this case.
	// +optional
	WorkloadCredentialRef SecretReference `json:"workloadCredentialRef,omitempty"`

	// AdminCredentialRef points to a Secret resource which includes the username
	// and password to access and configure the Avi Controller.
	//
	// * username                   Username used with basic authentication for
	//                              the Avi REST API
	// * password                   Password used with basic authentication for
	//                              the Avi REST API
	//
	// This credential needs to be bound with admin tenant and will be used
	// by AKO Operator to automate configurations and operations.
	// +optional
	AdminCredentialRef SecretReference `json:"adminCredentialRef"`

	// CertificateAuthorityRef points to a Secret resource that includes the
	// AVI Controller's CA
	//
	// * certificateAuthorityData   PEM-encoded certificate authority
	//                              certificates
	//
	CertificateAuthorityRef SecretReference `json:"certificateAuthorityRef"`

	// The AVI tenant for the current AKODeploymentConfig
	// This field is optional.
	// +optional
	Tenant AVITenant `json:"tenant,omitempty"`

	// DataNetworks describes the Data Networks the AKO will be deployed
	// with.
	// This field is immutable.
	DataNetwork DataNetwork `json:"dataNetwork"`

	// CustomizedConfigs contains the customized configurations for AKO to override
	// defaults
	//
	// When CustomizedConfigs is non-empty, AKO Operator will create
	// ClusterResourceSet only associated with the specified configs
	//
	// +optional
	CustomizedConfigs []ConfigRef `json:"customizedConfigs,omitempty"`
}

// ConfigRef specifies a resource.
type ConfigRef struct {
	// Name of the resource
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Kind of the resource. Supported kinds are: Secrets and ConfigMaps.
	// +kubebuilder:validation:Enum=Secret;ConfigMap
	Kind string `json:"kind"`

	// Namespace of the resource
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`
}

// AVITenant describes settings for an AVI Tenant object
type AVITenant struct {
	// Context is the type of AVI tenant context. Defaults to Provider. This field is immutable.
	// +kubebuilder:validation:Enum=Provider;Tenant
	Context string `json:"context,omitempty"`

	// Name is the name of the tenant. This field is immutable.
	Name string `json:"name"`
}

// DataNetwork describes one AVI Data Network
type DataNetwork struct {
	Name    string   `json:"name"`
	CIDR    string   `json:"cidr"`
	IPPools []IPPool `json:"ipPools"`
}

// IPPool defines a contiguous range of IP Addresses
type IPPool struct {
	// Start represents the starting IP address of the pool.
	Start string `json:"start"`
	// End represents the ending IP address of the pool.
	End string `json:"end"`
	// Type represents the type of IP Address
	// +kubebuilder:validation:Enum=V4;
	Type string `json:"type"`
}

// SecretReference references a Kind Secret object in the same kubernetes
// cluster
type SecretReference struct {
	// Name is the name of resource being referenced.
	Name string `json:"name"`
	// Namespace of the resource being referenced.
	Namespace string `json:"namespace"`
}

// AKODeploymentConfigStatus defines the observed state of AKODeploymentConfig
type AKODeploymentConfigStatus struct {
	// ObservedGeneration reflects the generation of the most recently
	// observed AKODeploymentConfig.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions defines current state of the AKODeploymentConfig.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// AKODeploymentConfig is the Schema for the akodeploymentconfigs API
type AKODeploymentConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AKODeploymentConfigSpec   `json:"spec,omitempty"`
	Status AKODeploymentConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AKODeploymentConfigList contains a list of AKODeploymentConfig
type AKODeploymentConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AKODeploymentConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AKODeploymentConfig{}, &AKODeploymentConfigList{})
}
