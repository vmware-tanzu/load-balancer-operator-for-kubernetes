// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	TKGSystemNamespace  = "tkg-system"
	TKGClusterNameLabel = "tkg.tanzu.vmware.com/cluster-name"

	AkoUserRoleName              = "ako-essential-role"
	ClusterFinalizer             = "ako-operator.networking.tkg.tanzu.vmware.com"
	AkoDeploymentConfigFinalizer = "ako-operator.networking.tkg.tanzu.vmware.com"
	AkoDeploymentConfigKind      = "AKODeploymentConfig"
	AkoDeploymentConfigVersion   = "networking.tanzu.vmware.com/v1alpha1"
	AkoConfigMapName             = "avi-k8s-config"

	AviClusterLabel                                              = "networking.tkg.tanzu.vmware.com/avi"
	AviClusterSecretType                                         = "avi.cluster.x-k8s.io/secret"
	AviSecretName                                                = "avi-secret"
	AviNamespace                                                 = "avi-system"
	AviCertificateKey                                            = "certificateAuthorityData"
	AviResourceCleanupReason                                     = "AviResourceCleanup"
	AviResourceCleanupSucceededCondition clusterv1.ConditionType = "AviResourceCleanupSucceeded"
	AviUserCleanupSucceededCondition     clusterv1.ConditionType = "AviUserCleanupSucceeded"
)
