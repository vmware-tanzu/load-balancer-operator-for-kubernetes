// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako

import (
	"encoding/json"
	"strconv"

	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
)

var _ = Describe("AKO", func() {
	Context("PopulateValues", func() {
		var (
			akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
			rendered            *Values
			err                 error
		)

		JustBeforeEach(func() {
			rendered, err = NewValues(akoDeploymentConfig, "test")
			Expect(err).ToNot(HaveOccurred())
		})

		ensureValueIsExpected := func(value *Values, akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig) {
			Expect(value).ShouldNot(BeNil())
			config := value.LoadBalancerAndIngressService.Config
			akoSettings := config.AKOSettings
			controllerSettings := config.ControllerSettings
			networkSettings := config.NetworkSettings
			l7Settings := config.L7Settings
			rbac := config.Rbac

			expectedPairs := map[string]string{
				akoSettings.ClusterName:                   "test",
				akoSettings.LogLevel:                      akoDeploymentConfig.Spec.ExtraConfigs.Log.LogLevel,      // use default value if not provided
				akoSettings.FullSyncFrequency:             akoDeploymentConfig.Spec.ExtraConfigs.FullSyncFrequency, // use default value if not provided
				akoSettings.CniPlugin:                     akoDeploymentConfig.Spec.ExtraConfigs.CniPlugin,
				akoSettings.DisableStaticRouteSync:        strconv.FormatBool(*akoDeploymentConfig.Spec.ExtraConfigs.DisableStaticRouteSync),
				controllerSettings.CloudName:              akoDeploymentConfig.Spec.CloudName,
				controllerSettings.ControllerIP:           akoDeploymentConfig.Spec.Controller,
				controllerSettings.ServiceEngineGroupName: akoDeploymentConfig.Spec.ServiceEngineGroup,
				networkSettings.NetworkName:               akoDeploymentConfig.Spec.DataNetwork.Name,
				networkSettings.ControlPlaneNetworkName:   akoDeploymentConfig.Spec.ControlPlaneNetwork.Name,
				networkSettings.ControlPlaneNetworkCIDR:   akoDeploymentConfig.Spec.ControlPlaneNetwork.CIDR,
				networkSettings.SubnetIP:                  "10.0.0.0",
				networkSettings.SubnetPrefix:              "24",
				config.PersistentVolumeClaim:              akoDeploymentConfig.Spec.ExtraConfigs.Log.PersistentVolumeClaim,
				config.MountPath:                          akoDeploymentConfig.Spec.ExtraConfigs.Log.MountPath,
				config.LogFile:                            akoDeploymentConfig.Spec.ExtraConfigs.Log.LogFile,
				value.LoadBalancerAndIngressService.Name:  "ako-test",
				rbac.PspPolicyApiVersion:                  akoDeploymentConfig.Spec.ExtraConfigs.Rbac.PspPolicyAPIVersion,
				rbac.PspPolicyApiVersion:                  "test/1.2",
				l7Settings.ShardVSSize:                    akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.ShardVSSize,
				l7Settings.ServiceType:                    akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.ServiceType,
			}
			for k, v := range expectedPairs {
				Expect(k).To(Equal(v))
			}

			expectedBoolPairs := map[bool]bool{
				l7Settings.DisableIngressClass:  *akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.DisableIngressClass,
				l7Settings.DefaultIngController: *akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.DefaultIngressController,
				rbac.PspEnabled:                 *akoDeploymentConfig.Spec.ExtraConfigs.Rbac.PspEnabled,
			}
			for k, v := range expectedBoolPairs {
				Expect(k).To(Equal(v))
			}

			if len(akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList) != 0 {
				nodeNetworkListJson, jsonerr := json.Marshal(akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList)
				Expect(jsonerr).ShouldNot(HaveOccurred())
				Expect(networkSettings.NodeNetworkListJson).To(Equal(string(nodeNetworkListJson)))
			} else {
				Expect(networkSettings.NodeNetworkListJson).Should(BeNil())
			}
			vipNetworkListJson, jsonerr := json.Marshal(
				[]akoov1alpha1.VIPNetwork{{
					NetworkName: akoDeploymentConfig.Spec.DataNetwork.Name,
					CIDR:        "10.0.0.0/24",
				}},
			)
			Expect(jsonerr).ShouldNot(HaveOccurred())
			Expect(networkSettings.VIPNetworkListJson).To(Equal(string(vipNetworkListJson)))
		}

		When("a valid AKODeploymentYaml is provided", func() {
			BeforeEach(func() {
				akoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
					Spec: akoov1alpha1.AKODeploymentConfigSpec{
						CloudName:          "test-cloud",
						Controller:         "10.23.122.1",
						ServiceEngineGroup: "Default-SEG",
						DataNetwork: akoov1alpha1.DataNetwork{
							Name: "test-akdc",
							CIDR: "10.0.0.0/24",
						},
						ControlPlaneNetwork: akoov1alpha1.ControlPlaneNetwork{
							Name: "test-akdc-2",
							CIDR: "10.1.0.0/24",
						},
						ExtraConfigs: akoov1alpha1.ExtraConfigs{
							FullSyncFrequency: "1900",
							Rbac: akoov1alpha1.AKORbacConfig{
								PspEnabled:          pointer.Bool(true),
								PspPolicyAPIVersion: "test/1.2",
							},
							Log: akoov1alpha1.AKOLogConfig{
								LogLevel:              "DEBUG",
								PersistentVolumeClaim: "true",
								MountPath:             "/var/log",
								LogFile:               "test-avi.log",
							},
							IngressConfigs: akoov1alpha1.AKOIngressConfig{
								DisableIngressClass:      pointer.Bool(true),
								DefaultIngressController: pointer.Bool(true),
								ShardVSSize:              "MEDIUM",
								ServiceType:              "NodePort",
								NodeNetworkList: []akoov1alpha1.NodeNetwork{
									{
										NetworkName: "test-node-network-1",
										Cidrs:       []string{"10.0.0.0/24", "192.168.0.0/24"},
									},
								},
							},
							DisableStaticRouteSync: pointer.BoolPtr(true),
							CniPlugin:              "antrea",
						},
					},
				}
			})
			It("should get correct values in the yaml", func() {
				ensureValueIsExpected(rendered, akoDeploymentConfig)
			})
		})

		When("a valid ipv6 AKODeploymentYaml is provided", func() {
			BeforeEach(func() {
				akoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
					Spec: akoov1alpha1.AKODeploymentConfigSpec{
						CloudName:          "test-cloud-2",
						Controller:         "10.23.122.1",
						ServiceEngineGroup: "Default-SEG",
						DataNetwork: akoov1alpha1.DataNetwork{
							Name: "test-akdc",
							CIDR: "2002::1234:abcd:ffff:c0a8:101/64",
						},
						ControlPlaneNetwork: akoov1alpha1.ControlPlaneNetwork{
							Name: "test-akdc-2",
							CIDR: "2002::1234:abcd:ffff:c0a8:101/64",
						},
						ExtraConfigs: akoov1alpha1.ExtraConfigs{
							FullSyncFrequency: "1900",
							IpFamily:          "V6",
							Rbac: akoov1alpha1.AKORbacConfig{
								PspEnabled:          pointer.Bool(true),
								PspPolicyAPIVersion: "test/1.2",
							},
							Log: akoov1alpha1.AKOLogConfig{
								LogLevel:              "DEBUG",
								PersistentVolumeClaim: "true",
								MountPath:             "/var/log",
								LogFile:               "test-avi.log",
							},
							IngressConfigs: akoov1alpha1.AKOIngressConfig{
								DisableIngressClass:      pointer.Bool(true),
								DefaultIngressController: pointer.Bool(true),
								ShardVSSize:              "MEDIUM",
								ServiceType:              "NodePort",
								NodeNetworkList: []akoov1alpha1.NodeNetwork{
									{
										NetworkName: "test-node-network-1",
										Cidrs:       []string{"2002::1234:abcd:ffff:c0a8:101/64"},
									},
								},
							},
							DisableStaticRouteSync: pointer.BoolPtr(true),
							CniPlugin:              "antrea",
						},
					},
				}
			})
			It("should get correct values in the yaml", func() {
				vipNetworkList, jsonerr := json.Marshal(
					[]akoov1alpha1.VIPNetwork{{
						NetworkName: akoDeploymentConfig.Spec.DataNetwork.Name,
						V6CIDR:      "2002::1234:abcd:ffff:c0a8:101/64",
					}},
				)
				networkSettings := rendered.LoadBalancerAndIngressService.Config.NetworkSettings
				Expect(jsonerr).ShouldNot(HaveOccurred())
				Expect(networkSettings.VIPNetworkListJson).To(Equal(string(vipNetworkList)))
			})
		})
	})
})
