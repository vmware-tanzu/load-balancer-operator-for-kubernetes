// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	ClusterFinalizer = "ako-operator.network.tkg.tanzu.vmware.com"

	AkoUserRoleName              = "ako-essential-role"
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
