// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"os/exec"
	"time"

	. "github.com/onsi/gomega"
	p "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	runv1alpha3 "github.com/vmware-tanzu/tanzu-framework/apis/run/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func CreateObjects(ctx *builder.IntegrationTestContext, objs ...client.Object) {
	for _, o := range objs {
		err := ctx.Client.Create(ctx.Context, o)
		Expect(err).ShouldNot(HaveOccurred())
		ensureRuntimeObjectCreated(ctx, o)
	}
}

func UpdateObjectsStatus(ctx *builder.IntegrationTestContext, objs ...client.Object) {
	for _, o := range objs {
		err := ctx.Client.Status().Update(ctx.Context, o)
		Expect(err).ShouldNot(HaveOccurred())
	}
}

func DeleteObjects(ctx *builder.IntegrationTestContext, objs ...client.Object) {
	for _, o := range objs {
		// ignore error
		_ = ctx.Client.Delete(ctx.Context, o)
	}
}

func ensureRuntimeObjectCreated(ctx *builder.IntegrationTestContext, o client.Object) {
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
	case *corev1.ConfigMap:
		obj = o.(*corev1.ConfigMap)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	case *akoov1alpha1.AKODeploymentConfig:
		obj = o.(*akoov1alpha1.AKODeploymentConfig)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	case *runv1alpha3.ClusterBootstrap:
		obj = o.(*runv1alpha3.ClusterBootstrap)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	case *p.Package:
		obj = o.(*p.Package)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	case *corev1.Secret:
		obj = o.(*corev1.Secret)
		EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, obj, EXIST)
	default:
		klog.Fatal("Unknown type object")
	}
}

func EnsureRuntimeObjectMatchExpectation(ctx *builder.IntegrationTestContext, objKey client.ObjectKey, obj client.Object, expectResult ExpectResult) {
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
	}, 30*time.Second).Should(BeTrue())
}

func EnsureClusterAviLabelExists(ctx *builder.IntegrationTestContext, key client.ObjectKey, label string, exists bool) {
	Eventually(func() bool {
		obj := &clusterv1.Cluster{}
		err := ctx.Client.Get(ctx.Context, key, obj)
		if err != nil {
			return false
		}
		_, ok := obj.Labels[label]
		return ok == exists
	}).Should(BeTrue())
}

func EnsureClusterAviLabelMatchExpectation(ctx *builder.IntegrationTestContext, key client.ObjectKey, label, expectVal string) {
	Eventually(func() bool {
		obj := &clusterv1.Cluster{}
		err := ctx.Client.Get(ctx.Context, key, obj)
		if err != nil {
			return false
		}
		val, ok := obj.Labels[label]
		return ok && val == expectVal
	}).Should(BeTrue())
}

func UpdateObjectLabels(ctx *builder.IntegrationTestContext, key client.ObjectKey, labels map[string]string) {
	Eventually(func() error {
		var cluster = new(clusterv1.Cluster)

		if err := ctx.Client.Get(ctx, client.ObjectKey{
			Name:      key.Name,
			Namespace: key.Namespace,
		}, cluster); err != nil {
			return err
		}
		cluster.Labels = labels
		if err := ctx.Client.Update(ctx, cluster, &client.UpdateOptions{}); err != nil {
			return err
		}
		return nil
	}).Should(Succeed())
}

func EnsureClusterBootstrapPackagesMatchExpectation(ctx *builder.IntegrationTestContext, key client.ObjectKey, refName string, exists bool) {
	Eventually(func() bool {
		obj := &runv1alpha3.ClusterBootstrap{}
		err := ctx.Client.Get(ctx.Context, key, obj)
		if err != nil {
			return false
		}
		found := findPkgByRefinCB(obj, refName)
		return found == exists
	}).Should(BeTrue())
}

func findPkgByRefinCB(cb *runv1alpha3.ClusterBootstrap, refName string) bool {
	for _, n := range cb.Spec.AdditionalPackages {
		if n.RefName == refName {
			return true
		}
	}
	return false
}
