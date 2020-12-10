// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// This packages provides utilities to interact with AKO

package ako

import (
	"context"

	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	akoStatefulSetName          = "ako"
	akoConditionType            = "akoStatus"
	akoObjectDeletionDoneStatus = "objDeletionDone"
)

func CleanupFinished(ctx context.Context, remoteClient client.Client, log logr.Logger) (bool, error) {

	ss := &appv1.StatefulSet{}
	if err := remoteClient.Get(ctx, client.ObjectKey{
		Name:      akoStatefulSetName,
		Namespace: "avi-system",
	}, ss); err != nil {
		log.Error(err, "Failed to get AKO StatefulSet")
		return false, err
	}

	return msgFoundInStatus(ss.Status.Conditions, akoObjectDeletionDoneStatus), nil
}

func msgFoundInStatus(conditions []appv1.StatefulSetCondition, msg string) bool {
	for _, c := range conditions {
		if c.Type == akoConditionType && c.Message == msg {
			return true
		}
	}
	return false
}
