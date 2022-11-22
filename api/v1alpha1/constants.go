// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	TKGSystemNamespace            = "tkg-system"
	TKGClusterNameLabel           = "tkg.tanzu.vmware.com/cluster-name"
	TKGClusterNameSpaceLabel      = "tkg.tanzu.vmware.com/cluster-namespace"
	TKGManagememtClusterRoleLabel = "cluster-role.tkg.tanzu.vmware.com/management"

	TKGAddonAnnotationKey          = "tkg.tanzu.vmware.com/addon-type"
	TKGAddOnLabelAddonNameKey      = "tkg.tanzu.vmware.com/addon-name"
	TKGAddOnLabelClusterNameKey    = "tkg.tanzu.vmware.com/cluster-name"
	TKGAddOnLabelClusterctlKey     = "clusterctl.cluster.x-k8s.io/move"
	TKGAddOnSecretType             = "tkg.tanzu.vmware.com/addon"
	TKGAddOnSecretDataKey          = "values.yaml"
	TKGDataValueFormatString       = "#@data/values\n#@overlay/match-child-defaults missing_ok=True\n---\n"
	TKGSkipDeletePkgiAnnotationKey = "run.tanzu.vmware.com/skip-packageinstall-deletion"

	ManagementClusterAkoDeploymentConfig = "install-ako-for-management-cluster"
	WorkloadClusterAkoDeploymentConfig   = "install-ako-for-all"

	AkoUserRoleName                  = "ako-essential-role"
	ClusterFinalizer                 = "ako-operator.networking.tkg.tanzu.vmware.com"
	AkoDeploymentConfigFinalizer     = "ako-operator.networking.tkg.tanzu.vmware.com"
	AkoDeploymentConfigKind          = "AKODeploymentConfig"
	AkoDeploymentConfigVersion       = "networking.tanzu.vmware.com/v1alpha1"
	AkoStatefulSetName               = "ako"
	AkoConfigMapName                 = "avi-k8s-config"
	AkoConfigMapCloudNameKey         = "cloudName"
	AkoConfigMapControllerIPKey      = "controllerIP"
	AkoConfigMapVipNetworkListKey    = "vipNetworkList"
	AkoClusterBootstrapRefNamePrefix = "load-balancer-and-ingress-service.tanzu.vmware.com"
	AkoPackageInstallName            = "load-balancer-and-ingress-service"

	AviClusterLabel                                              = "networking.tkg.tanzu.vmware.com/avi"
	AviClusterDeleteConfigLabel                                  = "networking.tkg.tanzu.vmware.com/avi-config-delete"
	AviClusterSecretType                                         = "avi.cluster.x-k8s.io/secret"
	AviNamespace                                                 = "avi-system"
	AviCredentialName                                            = "avi-controller-credentials"
	AviCAName                                                    = "avi-controller-ca"
	AviCertificateKey                                            = "certificateAuthorityData"
	AviResourceCleanupReason                                     = "AviResourceCleanup"
	AviResourceCleanupSucceededCondition clusterv1.ConditionType = "AviResourceCleanupSucceeded"
	AviUserCleanupSucceededCondition     clusterv1.ConditionType = "AviUserCleanupSucceeded"
	PreTerminateAnnotation                                       = clusterv1.PreTerminateDeleteHookAnnotationPrefix + "/avi-cleanup"

	HAServiceName                      = "control-plane"
	HAServiceBootstrapClusterFinalizer = "ako-operator.networking.tkg.tanzu.vmware.com/ha"
	HAServiceAnnotationsKey            = "skipnodeport.ako.vmware.com/enabled"
	HAAVIInfraSettingAnnotationsKey    = "aviinfrasetting.ako.vmware.com/name"

	AKODeploymentConfigControllerName = "akodeploymentconfig-controller"
)
