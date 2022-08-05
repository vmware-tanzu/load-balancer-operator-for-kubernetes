// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"math/rand"
	"time"

	p "github.com/vmware-tanzu/carvel-kapp-controller/pkg/apiserver/apis/datapackaging/v1alpha1"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	runv1alpha3 "github.com/vmware-tanzu/tanzu-framework/apis/run/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/apimachinery/pkg/runtime/schema"
	// "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// CharSet defines the alphanumeric set for random string generation.
	CharSet = "0123456789abcdefghijklmnopqrstuvwxyz"
)

var rnd = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

// RandomString returns a random alphanumeric string.
func RandomString(n int) string {
	result := make([]byte, n)
	for i := range result {
		result[i] = CharSet[rnd.Intn(len(CharSet))]
	}
	return string(result)
}

// AKODeploymentConfig test data
var DefaultAkoDeploymentConfigCommonSpec = akoov1alpha1.AKODeploymentConfigSpec{
	DataNetwork: akoov1alpha1.DataNetwork{
		Name: "integration-test-8ed12g",
		CIDR: "10.0.0.0/24",
		IPPools: []akoov1alpha1.IPPool{
			{
				Start: "10.0.0.1",
				End:   "10.0.0.10",
				Type:  "V4",
			},
		},
	},
	ControlPlaneNetwork: akoov1alpha1.ControlPlaneNetwork{
		Name: "integration-test-8ed12g",
		CIDR: "10.1.0.0/24",
	},
	ServiceEngineGroup: "ha-test",
	AdminCredentialRef: &akoov1alpha1.SecretRef{
		Name:      "controller-credentials",
		Namespace: "default",
	},
	CertificateAuthorityRef: &akoov1alpha1.SecretRef{
		Name:      "controller-ca",
		Namespace: "default",
	},
}

// GetManagementADC returns a pointer to install-ako-for-management-cluster ADC for testing
// it always selects cluster labelled as {"cluster-role.tkg.tanzu.vmware.com/management": ""}
func GetManagementADC() *akoov1alpha1.AKODeploymentConfig {
	adc := &akoov1alpha1.AKODeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: akoov1alpha1.ManagementClusterAkoDeploymentConfig},
		Spec:       DefaultAkoDeploymentConfigCommonSpec,
	}
	adc.Spec.ClusterSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"cluster-role.tkg.tanzu.vmware.com/management": "",
		},
	}
	return adc
}

// GetDefaultADC returns a pointer to install-ako-for-all ADC for testing
// it has empty cluster selector to match all clusters
func GetDefaultADC() *akoov1alpha1.AKODeploymentConfig {
	return &akoov1alpha1.AKODeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: akoov1alpha1.WorkloadClusterAkoDeploymentConfig},
		Spec:       DefaultAkoDeploymentConfigCommonSpec,
	}
}

var (
	// CustomizedADCLabels defines a common selector used for testing
	CustomizedADCLabels = map[string]string{"test": "true"}
)

// GetCustomizedADC returns a pointer to a customized ADC for testing
// with specified cluster selector labels
func GetCustomizedADC(labels map[string]string) *akoov1alpha1.AKODeploymentConfig {
	spec := DefaultAkoDeploymentConfigCommonSpec.DeepCopy()
	spec.ClusterSelector = metav1.LabelSelector{
		MatchLabels: labels,
	}
	return &akoov1alpha1.AKODeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ako-deployment-config",
		},
		Spec: *spec,
	}
}

// var test = corev1.Secret{
// 	TypeMeta:   metav1.TypeMeta{},
// 	ObjectMeta: metav1.ObjectMeta{},
// 	Immutable:  new(bool),
// 	Data:       map[string][]byte{},
// 	StringData: map[string]string{},
// 	Type:       "",
// }
// Cluster test data

var DefaultCluster = clusterv1.Cluster{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "integration-test",
		Namespace: "default",
	},
	Spec: clusterv1.ClusterSpec{},
}

func GetDefaultCluster() *clusterv1.Cluster {
	cluster := DefaultCluster.DeepCopy()
	cluster.Name = cluster.Name + "-" + RandomString(6)
	return cluster
}

func GetManagementCluster() *clusterv1.Cluster {
	cluster := DefaultCluster.DeepCopy()
	cluster.Name = cluster.Name + "-" + RandomString(6) + "-mgmt"
	cluster.Namespace = "tkg-system"
	cluster.Labels = map[string]string{
		"cluster-role.tkg.tanzu.vmware.com/management": "",
	}
	return cluster
}

// ClusterBootstrap test data

var DefaultSecret = &corev1.Secret{
	TypeMeta:   metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{},
	Immutable:  new(bool),
	Data:       map[string][]byte{},
	StringData: map[string]string{},
	Type:       "",
}

var testSecret = corev1.Secret{
	TypeMeta:   metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{},
	Immutable:  new(bool),
	Data:       map[string][]byte{},
	StringData: map[string]string{},
	Type:       "",
}

var startingBootstrapPackage = runv1alpha3.ClusterBootstrapPackage{
	RefName: "Initial-Package",
	ValuesFrom: &runv1alpha3.ValuesFrom{
		SecretRef: "Initial-SecretRef",
	},
}

var cbSpec = runv1alpha3.ClusterBootstrapTemplateSpec{
	Paused:             false,
	CNI:                &runv1alpha3.ClusterBootstrapPackage{},
	CSI:                &runv1alpha3.ClusterBootstrapPackage{},
	CPI:                &runv1alpha3.ClusterBootstrapPackage{},
	Kapp:               &runv1alpha3.ClusterBootstrapPackage{},
	AdditionalPackages: []*runv1alpha3.ClusterBootstrapPackage{},
}

// type TestPackage struct {
// 	Name      string
// 	Namespace string
// 	RefName   string
// }

var DefaultClusterBootstrap = runv1alpha3.ClusterBootstrap{}

var DefaultAKOPackage = p.Package{
	TypeMeta:   metav1.TypeMeta{},
	ObjectMeta: metav1.ObjectMeta{},
	Spec:       p.PackageSpec{},
}

// func GetTestPackage(cluster *clusterv1.Cluster) *TestPackage {
// 	testpkg := TestPackage{
// 		Name:      cluster.Name,
// 		Namespace: cluster.Namespace,
// 		RefName:   "Testing-Package",
// 	}
// 	return &testpkg
// }

func GetDefaultSecret(cluster *clusterv1.Cluster) *corev1.Secret {
	secret := DefaultSecret.DeepCopy()
	secret.SetName(cluster.Name + "-load-balancer-and-ingress-service-addon")
	secret.SetNamespace(cluster.Namespace)
	return secret
}

func GetDefaultCB(cluster *clusterv1.Cluster) *runv1alpha3.ClusterBootstrap {
	clusterBootstrap := DefaultClusterBootstrap.DeepCopy()
	clusterBootstrap.Name = cluster.Name
	clusterBootstrap.Namespace = cluster.Namespace
	clusterBootstrap.Spec = &cbSpec
	return clusterBootstrap
}

// func GetDefaultAKOPackage(cluster *clusterv1.Cluster) *p.Package {
// 	akoPackage := DefaultAKOPackage.DeepCopy()
// 	akoPackage.

// 	akoPackage.Namespace = cluster.Namespace
// 	akoPackage.Spec.RefName = "load-balancer-and-ingress-service.tanzu.vmware.com"

// 	return akoPackage
// }

var cbp = runv1alpha3.ClusterBootstrapPackage{
	RefName:    "testCBP",
	ValuesFrom: &runv1alpha3.ValuesFrom{},
}

func GetDefaultCBP(cluster *clusterv1.Cluster) *runv1alpha3.ClusterBootstrap {

	clusterBootstrapPackage := DefaultClusterBootstrap.DeepCopy()
	clusterBootstrapPackage.Name = cluster.Name
	clusterBootstrapPackage.Namespace = cluster.Namespace
	return clusterBootstrapPackage
}
