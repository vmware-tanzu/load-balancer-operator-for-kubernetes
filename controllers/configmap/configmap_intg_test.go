// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package configmap_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	testutil "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/util"
	"github.com/vmware/alb-sdk/go/models"
	"github.com/vmware/alb-sdk/go/session"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"os"

	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/test/builder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func intgTestEnsureUsableNetworkAddedInBootstrapCluster() {

	var (
		ctx *builder.IntegrationTestContext

		configMap                *corev1.ConfigMap
		UpdateIPAMFnCalled       bool
		UpdateIPAMUsableNetworks []*models.IPAMUsableNetwork
	)

	configMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "avi-k8s-config",
			Namespace: "tkg-system",
		},
		Data: map[string]string{
			"controllerIP":   "10.168.176.11",
			"cloudName":      "Default Cloud 2",
			"vipNetworkList": "[{\"networkName\":\"VM Network 2\",\"cidr\":\"10.168.176.0/20\"}]",
		},
	}

	GetIPAMFuncWithUsableNetwork := func(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
		res := &models.IPAMDNSProviderProfile{
			InternalProfile: &models.IPAMDNSInternalProfile{
				UsableNetworks: []*models.IPAMUsableNetwork{{NwRef: pointer.StringPtr("10.168.176.0")}},
			},
		}
		return res, nil
	}

	GetIPAMFuncWithoutUsableNetwork := func(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
		res := &models.IPAMDNSProviderProfile{
			InternalProfile: &models.IPAMDNSInternalProfile{
				UsableNetworks: []*models.IPAMUsableNetwork{},
			},
		}
		return res, nil
	}

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()

		// reset assertion vars
		UpdateIPAMFnCalled = false
		UpdateIPAMUsableNetworks = []*models.IPAMUsableNetwork{}

		// stub the NetworkGetByName of the avi client Network
		ctx.AviClient.Network.SetGetByNameFn(func(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
			if name == "VM Network 2" {
				return &models.Network{URL: pointer.StringPtr("10.168.176.0")}, nil
			}
			return nil, errors.New(fmt.Sprintf("%s network not found\n", name))
		})

		// stub the NetworkGetByName of the avi client Cloud
		ctx.AviClient.Cloud.SetGetByNameCloudFunc(func(name string, options ...session.ApiOptionsParams) (*models.Cloud, error) {
			if name == "Default Cloud 2" {
				return &models.Cloud{
					IPAMProviderRef: pointer.StringPtr("https://10.0.0.x/api/ipamdnsproviderprofile/ipamdnsproviderprofile-f08403a1-0dc7-4f13-bda3-0ba2fa476516"),
				}, nil
			}
			return nil, errors.New(fmt.Sprintf("%s cloud not found\n", name))
		})

		ctx.AviClient.IPAMDNSProviderProfile.SetUpdateIPAMFn(func(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
			UpdateIPAMFnCalled = true
			UpdateIPAMUsableNetworks = obj.InternalProfile.UsableNetworks
			return obj, nil
		})

		err := os.Setenv(ako_operator.IsControlPlaneHAProvider, "False")
		Expect(err).ShouldNot(HaveOccurred())

		err = os.Setenv(ako_operator.DeployInBootstrapCluster, "True")
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
	})

	When("The network already exists in IPAM profile", func() {
		BeforeEach(func() {

			// stub the IPAMDNSProviderProfileGet of the avi client
			ctx.AviClient.IPAMDNSProviderProfile.SetGetIPAMFunc(GetIPAMFuncWithUsableNetwork)
			testutil.CreateObjects(ctx, configMap.DeepCopy())
		})

		AfterEach(func() {
			testutil.DeleteObjects(ctx, configMap.DeepCopy())
			testutil.EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{
				Name:      "avi-k8s-config",
				Namespace: "tkg-system",
			}, &corev1.ConfigMap{}, testutil.NOTFOUND)
		})

		It("IPAM profile should not be updated", func() {
			Consistently(func() bool {
				return UpdateIPAMFnCalled == false
			}, time.Second).Should(BeTrue())
		})
	})

	When("The network does not exists in IPAM profile", func() {
		BeforeEach(func() {
			// stub the IPAMDNSProviderProfileGet of the avi client
			ctx.AviClient.IPAMDNSProviderProfile.SetGetIPAMFunc(GetIPAMFuncWithoutUsableNetwork)
			testutil.CreateObjects(ctx, configMap.DeepCopy())
		})

		AfterEach(func() {
			testutil.DeleteObjects(ctx, configMap.DeepCopy())
			testutil.EnsureRuntimeObjectMatchExpectation(ctx, client.ObjectKey{
				Name:      "avi-k8s-config",
				Namespace: "tkg-system",
			}, &corev1.ConfigMap{}, testutil.NOTFOUND)
		})

		It("IPAM profile should be updated", func() {
			Eventually(func() bool {
				return UpdateIPAMFnCalled == true && len(UpdateIPAMUsableNetworks) == 1 &&
					*UpdateIPAMUsableNetworks[0].NwRef == "10.168.176.0"
			}).Should(BeTrue())
		})
	})
}
