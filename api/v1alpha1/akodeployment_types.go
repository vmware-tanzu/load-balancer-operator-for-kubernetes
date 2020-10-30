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

// AKODeploymentSpec defines the desired state of an AKODeployment
// AKODeployment describes the shared configurations for AKO deployments across a set
// of Clusters.
type AKODeploymentSpec struct {
	CloudName string

	// Label selector for Clusters. The Clusters that are
	// selected by this will be the ones affected by this AKODeployment.
	// It must match the Cluster labels. This field is immutable.
	ClusterSelector metav1.LabelSelector `json:"clusterSelector"`

	// credentialRef points to a Secret resource which includes the username
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
	CredentialRef SecretReference `json:"credentialRef,omitempty"`

	// CertificateAuthorityRef points to a Secret resource that includes the
	// AVI Controller's CA
	//
	// * certificateAuthorityData   PEM-encoded certificate authority
	//                              certificates
	//
	CertificateAuthorityRef SecretReference `json:"certificateAuthorityRef"`

	// The AVI tenant for the current AKODeployment
	// This field is optional. When this field is not provided,
	// CredentialRef must be non-nil.
	// +optional
	Tenant AVITenant `json:"tenant,omitempty"`

	// DataNetworks describes the Data Networks the AKO will be deployed
	// with.
	// This field is immutable.
	DataNetworks []DataNetwork `json:"dataNetworks"`

	// SpecRef points to a Secret that contains the deployment spec for AKO
	//
	// * akoDeployment              The deployment spec for AKO, which
	//                              includes but is not limited to the
	//                              ServiceAccount, RoleBinding, ConfigMap,
	//                              StatefulSet
	//
	SpecRef SecretReference `json:"specRef"`
}

// AVITenant describes settings for an AVI Tenant object
type AVITenant struct {
	// Context is the type of AVI tenant context. Defaults to Provider. This field is immutable.
	// +kubebuilder:validation:Enum=Provider,Tenant
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
	// Address represents the starting IP address of the pool.
	Address string `json:"address"`
	// Count represents the number of IP addresses in the pool.
	Count int64 `json:"count"`
}

// SecretReference references a Kind Secret object in the same kubernetes
// cluster
type SecretReference struct {
	// Name is the name of resource being referenced.
	Name string `json:"name"`
	// Namespace of the resource being referenced. If empty, cluster scoped
	// resource is assumed.
	// +kubebuilder:default:=default
	Namespace string `json:"namespace,omitempty"`
}

// AKODeploymentStatus defines the observed state of AKODeployment
type AKODeploymentStatus struct {
	// ObservedGeneration reflects the generation of the most recently
	// observed AKODeployment.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions defines current state of the AKODeployment.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// AKODeployment is the Schema for the akodeployments API
type AKODeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AKODeploymentSpec   `json:"spec,omitempty"`
	Status AKODeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AKODeploymentList contains a list of AKODeployment
type AKODeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AKODeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AKODeployment{}, &AKODeploymentList{})
}
