// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

const (
	TKGSystemNamespace       = "tkg-system"
	TKGClusterNameLabel      = "tkg.tanzu.vmware.com/cluster-name"
	TKGClusterNameSpaceLabel = "tkg.tanzu.vmware.com/cluster-namespace"

	TKGAddonAnnotationKey       = "tkg.tanzu.vmware.com/addon-type"
	TKGAddOnLabelAddonNameKey   = "tkg.tanzu.vmware.com/addon-name"
	TKGAddOnLabelClusterNameKey = "tkg.tanzu.vmware.com/cluster-name"
	TKGAddOnLabelClusterctlKey  = "clusterctl.cluster.x-k8s.io/move"
	TKGAddOnSecretType          = "tkg.tanzu.vmware.com/addon"
	TKGAddOnSecretDataKey       = "values.yaml"
	TKGDataValueFormatString    = "#@data/values\n#@overlay/match-child-defaults missing_ok=True\n---\n"

	ManagementClusterAkoDeploymentConfig = "install-ako-for-management-cluster"

	AkoUserRoleName              = "ako-essential-role"
	ClusterFinalizer             = "ako-operator.networking.tkg.tanzu.vmware.com"
	AkoDeploymentConfigFinalizer = "ako-operator.networking.tkg.tanzu.vmware.com"
	AkoDeploymentConfigKind      = "AKODeploymentConfig"
	AkoDeploymentConfigVersion   = "networking.tanzu.vmware.com/v1alpha1"
	AkoStatefulSetName           = "ako"

	AVI_VERSION                                                  = "20.1.3"
	AviClusterLabel                                              = "networking.tkg.tanzu.vmware.com/avi"
	AviClusterSelectedLabel                                      = "networking.tkg.tanzu.vmware.com/avi-skip-default-adc"
	AviClusterSecretType                                         = "avi.cluster.x-k8s.io/secret"
	AviNamespace                                                 = "avi-system"
	AviCertificateKey                                            = "certificateAuthorityData"
	AviResourceCleanupReason                                     = "AviResourceCleanup"
	AviResourceCleanupSucceededCondition clusterv1.ConditionType = "AviResourceCleanupSucceeded"
	AviUserCleanupSucceededCondition     clusterv1.ConditionType = "AviUserCleanupSucceeded"
	PreTerminateAnnotation                                       = clusterv1.PreTerminateDeleteHookAnnotationPrefix + "/avi-cleanup"

	HAServiceName                      = "control-plane"
	HAServiceBootstrapClusterFinalizer = "ako-operator.networking.tkg.tanzu.vmware.com/ha"
	HAServiceAnnotationsKey            = "skipnodeport.ako.vmware.com/enabled"
	ClusterControlPlaneAnnotations     = "tkg.tanzu.vmware.com/cluster-controlplane-endpoint"

	AKODeploymentConfigControllerName = "akodeploymentconfig-controller"
)
