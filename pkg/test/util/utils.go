// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"os/exec"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	capi "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
)

type ExpectResult int

const (
	EXIST    ExpectResult = 0
	NOTFOUND ExpectResult = 1
	ERROR    ExpectResult = 2
)

// FindModuleDir returns the on-disk directory for the provided Go module.
func FindModuleDir(module string) string {
	cmd := exec.Command("go", "mod", "download", "-json", module)
	out, err := cmd.Output()
	if err != nil {
		klog.Fatalf("Failed to run go mod to find module %q directory", module)
	}
	info := struct{ Dir string }{}
	if err := json.Unmarshal(out, &info); err != nil {
		klog.Fatalf("Failed to unmarshal output from go mod command: %v", err)
	} else if info.Dir == "" {
		klog.Fatalf("Failed to find go module %q directory, received %v", module, string(out))
	}
	return info.Dir
}

func CreateObjects(ctx *builder.IntegrationTestContext, objs ...runtime.Object) {
	for _, o := range objs {
		err := ctx.Client.Create(ctx.Context, o)
		Expect(err).ShouldNot(HaveOccurred())
		ensureRuntimeObjectCreated(ctx, o)
	}
}

func UpdateObjects(ctx *builder.IntegrationTestContext, objs ...runtime.Object) {
	for _, o := range objs {
		err := ctx.Client.Update(ctx.Context, o)
		Expect(err).ShouldNot(HaveOccurred())
		ensureRuntimeObjectCreated(ctx, o)
	}
}

func UpdateObjectsStatus(ctx *builder.IntegrationTestContext, objs ...runtime.Object) {
	for _, o := range objs {
		err := ctx.Client.Status().Update(ctx.Context, o)
		Expect(err).ShouldNot(HaveOccurred())
	}
}

func DeleteObjects(ctx *builder.IntegrationTestContext, objs ...runtime.Object) {
	for _, o := range objs {
		// ignore error
		_ = ctx.Client.Delete(ctx.Context, o)
	}
}

func ensureRuntimeObjectCreated(ctx *builder.IntegrationTestContext, o runtime.Object) {
	switch obj := o.(type) {
	case *corev1.Namespace:
		obj = o.(*corev1.Namespace)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	case *capi.Machine:
		obj = o.(*capi.Machine)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	case *capi.Cluster:
		obj = o.(*capi.Cluster)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	default:
		klog.Fatal("Unknown type object")
	}
}

func EnsureRuntimeObjectMatchExpectation(ctx *builder.IntegrationTestContext, objKey client.ObjectKey, obj runtime.Object, expectResult ExpectResult) {
	Eventually(func() bool {
		res := EXIST
		if err := ctx.Client.Get(ctx.Context, objKey, obj); err != nil {
			if apierrors.IsNotFound(err) {
				res = NOTFOUND
			} else {
				res = ERROR
			}
		}
		return res == expectResult
	}).Should(BeTrue())
}
