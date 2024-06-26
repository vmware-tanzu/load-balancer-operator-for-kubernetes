// Copyright 2024 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func AVIUserSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-avi-credentials"
}

func AKOAddonSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-addon"
}

func AKOAddonSecretNameForClusterClass(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-data-values"
}
