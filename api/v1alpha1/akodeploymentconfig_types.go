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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AKODeploymentConfigSpec defines the desired state of AKODeploymentConfig
type AKODeploymentConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of AKODeploymentConfig. Edit AKODeploymentConfig_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// AKODeploymentConfigStatus defines the observed state of AKODeploymentConfig
type AKODeploymentConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
