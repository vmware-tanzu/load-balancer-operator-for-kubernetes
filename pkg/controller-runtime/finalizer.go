// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package controllerruntime

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ContainsFinalizer checks a metav1 object that the provided finalizer is present.
// Until https://github.com/kubernetes-sigs/controller-runtime/issues/920
func ContainsFinalizer(o metav1.Object, finalizer string) bool {
	f := o.GetFinalizers()
	for _, e := range f {
		if e == finalizer {
			return true
		}
	}
	return false
}
