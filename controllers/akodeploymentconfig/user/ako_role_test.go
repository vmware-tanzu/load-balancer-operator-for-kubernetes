// Copyright 2024 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"testing"
)

func TestRolePermissionAndMapMatch(t *testing.T) {
	if len(AkoRolePermission) != len(AkoRolePermissionMap) {
		t.Errorf("len(AkoRolePermission) == %d, len(AkoRolePermissionMap) == %d", len(AkoRolePermission), len(AkoRolePermissionMap))
	}

	allMatch := true
	for _, permission := range AkoRolePermission {
		if *permission.Type != AkoRolePermissionMap[*permission.Resource] {
			allMatch = false
			t.Logf("AkoRolePermission[%s] == %s, AkoRolePermissionMap[%s] == %s", *permission.Resource, *permission.Type, *permission.Resource, AkoRolePermissionMap[*permission.Resource])
		}
	}

	if !allMatch {
		t.Error("Not all entries in AkoRolePermission and AkoRolePermissionMap match")
	}
}
