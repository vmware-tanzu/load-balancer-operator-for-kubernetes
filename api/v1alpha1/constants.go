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
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	ClusterFinalizer = "ako-operator.network.tkg.tanzu.vmware.com"

	AkoDeploymentConfigFinalizer = "akodeploymentconfig.ako-operator.network.tkg.tanzu.vmware.com"
	AkoDeploymentConfigKind      = "AKODeploymentConfig"
	AkoDeploymentConfigVersion   = "network.tanzu.vmware.com/v1alpha1"

	AviClusterLabel                                              = "cluster-service.network.tkg.tanzu.vmware.com/avi"
	AviClusterSecretType                                         = "avi.cluster.x-k8s.io/secret"
	AviSecretName                                                = "avi-secret"
	AviNamespace                                                 = "avi-system"
	AviCertificateKey                                            = "certificateAuthorityData"
	AviResourceCleanupReason                                     = "AviResourceCleanup"
	AviResourceCleanupSucceededCondition clusterv1.ConditionType = "AviResourceCleanupSucceeded"
)
