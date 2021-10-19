// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	ako_operator "gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako-operator"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util/conditions"
)

func SkipCluster(cluster *clusterv1.Cluster) bool {
	// if condition.ready is false and cluster is not being deleted and not bootstrap cluster, skip
	if conditions.IsFalse(cluster, clusterv1.ReadyCondition) &&
		cluster.DeletionTimestamp.IsZero() &&
		!ako_operator.IsBootStrapCluster() {
		return true
	}
	return false
}
