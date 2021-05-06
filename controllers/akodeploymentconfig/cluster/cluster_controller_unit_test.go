// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func unitTestAKODeploymentYaml() {
	Context("PopluateValues", func() {
		var (
			akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
			rendered            cluster.Values
			err                 error
			capicluster         *clusterv1.Cluster
		)
		BeforeEach(func() {
			capicluster = &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
			}
		})
		JustBeforeEach(func() {
			rendered, err = cluster.PopluateValues(akoDeploymentConfig, capicluster)
			Expect(err).ToNot(HaveOccurred())
		})

		ensureValueIsExpected := func(value cluster.Values, akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig, cluster *clusterv1.Cluster) {
			expectedPairs := map[string]string{
				value.Image.Repository:                          akoDeploymentConfig.Spec.ExtraConfigs.Image.Repository,
				value.Image.PullPolicy:                          akoDeploymentConfig.Spec.ExtraConfigs.Image.PullPolicy,
				value.Image.Version:                             akoDeploymentConfig.Spec.ExtraConfigs.Image.Version,
				value.AKOSettings.ClusterName:                   cluster.Namespace + "-" + cluster.Name,
				value.ControllerSettings.CloudName:              akoDeploymentConfig.Spec.CloudName,
				value.ControllerSettings.ControllerIP:           akoDeploymentConfig.Spec.Controller,
				value.ControllerSettings.ServiceEngineGroupName: akoDeploymentConfig.Spec.ServiceEngineGroup,
				value.NetworkSettings.NetworkName:               akoDeploymentConfig.Spec.DataNetwork.Name,
				value.NetworkSettings.SubnetIP:                  "10.0.0.0",
				value.NetworkSettings.SubnetPrefix:              "24",
				value.PersistentVolumeClaim:                     akoDeploymentConfig.Spec.ExtraConfigs.Log.PersistentVolumeClaim,
				value.MountPath:                                 akoDeploymentConfig.Spec.ExtraConfigs.Log.MountPath,
				value.LogFile:                                   akoDeploymentConfig.Spec.ExtraConfigs.Log.LogFile,
				value.Name:                                      "ako-test-cluster",
				value.Rbac.PspPolicyApiVersion:                  akoDeploymentConfig.Spec.ExtraConfigs.Rbac.PspPolicyAPIVersion,
				value.Rbac.PspPolicyApiVersion:                  "test/1.2",
				value.L7Settings.ShardVSSize:                    akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.ShardVSSize,
				value.L7Settings.ServiceType:                    akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.ServiceType,
			}
			for k, v := range expectedPairs {
				Expect(k).To(Equal(v))
			}

			expectedBoolPairs := map[bool]bool{
				value.AKOSettings.DisableStaticRouteSync: akoDeploymentConfig.Spec.ExtraConfigs.DisableStaticRouteSync,
				value.L7Settings.DisableIngressClass:     akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.DisableIngressClass,
				value.L7Settings.DefaultIngController:    akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.DefaultIngressController,
				value.Rbac.PspEnabled:                    akoDeploymentConfig.Spec.ExtraConfigs.Rbac.PspEnabled,
			}
			for k, v := range expectedBoolPairs {
				Expect(k).To(Equal(v))
			}

			if len(akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList) != 0 {
				nodeNetworkListJson, jsonerr := json.Marshal(akoDeploymentConfig.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList)
				Expect(jsonerr).ShouldNot(HaveOccurred())
				Expect(value.NetworkSettings.NodeNetworkListJson).To(Equal(string(nodeNetworkListJson)))
			} else {
				Expect(value.NetworkSettings.NodeNetworkListJson).Should(BeNil())
			}
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
						ExtraConfigs: akoov1alpha1.ExtraConfigs{
							Image: akoov1alpha1.AKOImageConfig{
								Repository: "test/image",
								PullPolicy: "IfNotPresent",
								Version:    "1.3.1",
							},
							Rbac: akoov1alpha1.AKORbacConfig{
								PspEnabled:          true,
								PspPolicyAPIVersion: "test/1.2",
							},
							Log: akoov1alpha1.AKOLogConfig{
								PersistentVolumeClaim: "true",
								MountPath:             "/var/log",
								LogFile:               "test-avi.log",
							},
							IngressConfigs: akoov1alpha1.AKOIngressConfig{
								DisableIngressClass:      true,
								DefaultIngressController: true,
								ShardVSSize:              "MEDIUM",
								ServiceType:              "NodePort",
								NodeNetworkList: []akoov1alpha1.NodeNetwork{
									{
										NetworkName: "test-node-network-1",
										Cidrs:       []string{"10.0.0.0/24", "192.168.0.0/24"},
									},
								},
							},
							DisableStaticRouteSync: true,
						},
					},
				}
			})
			It("should get correct values in the yaml", func() {
				ensureValueIsExpected(rendered, akoDeploymentConfig, capicluster)
			})

			It("should populate correct values in crs yaml", func() {
				_, err := cluster.AKODeploymentYaml(akoDeploymentConfig, capicluster)
				Expect(err).ShouldNot(HaveOccurred())
			})

			It("should throw error if template not match", func() {
				akoDeploymentConfig.Spec.DataNetwork.CIDR = "test"
				_, err := cluster.AKODeploymentYaml(akoDeploymentConfig, capicluster)
				Expect(err).Should(HaveOccurred())
				akoDeploymentConfig.Spec.DataNetwork.CIDR = "10.0.0.0/24"
			})
		})
	})
}
