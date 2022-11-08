// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("AKO Operator lib unit test", func() {
	var (
		legacyCluster       *clusterv1.Cluster
		clusterClassCluster *clusterv1.Cluster
	)

	boolRaw, _ := json.Marshal(true)
	intRaw, _ := json.Marshal(31005)

	legacyCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "legacy-cluster",
			Namespace: "default",
		},
		Spec: clusterv1.ClusterSpec{},
	}

	clusterClassCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cluster-class-cluster",
			Namespace: "default",
		},
		Spec: clusterv1.ClusterSpec{
			Topology: &clusterv1.Topology{
				Variables: []clusterv1.ClusterVariable{
					{
						Name: AviAPIServerHAProvider,
						Value: v1.JSON{
							Raw: boolRaw,
						},
					},
					{
						Name: KubeVipLoadBalancerProvider,
						Value: v1.JSON{
							Raw: boolRaw,
						},
					},
					{
						Name: ApiServerPort,
						Value: v1.JSON{
							Raw: intRaw,
						},
					},
				},
			},
		},
	}

	Context("If ako operator is deployed in bootstrap cluster", func() {
		When("ako operator is deployed in bootstrap cluster", func() {
			BeforeEach(func() {
				os.Setenv(DeployInBootstrapCluster, "True")
			})
			AfterEach(func() {
				os.Unsetenv(DeployInBootstrapCluster)
			})
			It("should return True", func() {
				Expect(IsBootStrapCluster()).Should(Equal(true))
			})
		})
		When("ako operator is deployed in mgmt cluster", func() {
			BeforeEach(func() {
				os.Setenv(DeployInBootstrapCluster, "False")
			})
			AfterEach(func() {
				os.Unsetenv(DeployInBootstrapCluster)
			})
			It("should return false", func() {
				Expect(IsBootStrapCluster()).Should(Equal(false))
			})
		})
	})

	Context("ako operator control plane HA provider", func() {
		Context("Legacy Cluster Cases", func() {
			When("ako operator provides control plane HA", func() {
				BeforeEach(func() {
					os.Setenv(IsControlPlaneHAProvider, "True")
				})
				AfterEach(func() {
					os.Unsetenv(IsControlPlaneHAProvider)
				})
				It("should return True", func() {
					Expect(IsControlPlaneVIPProvider(nil)).Should(Equal(true))
				})
			})
			When("ako operator provides control plane HA", func() {
				BeforeEach(func() {
					os.Setenv(IsControlPlaneHAProvider, "True")
				})
				AfterEach(func() {
					os.Unsetenv(IsControlPlaneHAProvider)
				})
				It("should return True", func() {
					Expect(IsControlPlaneVIPProvider(legacyCluster)).Should(Equal(true))
				})
			})
			When("ako operator doesn't provide control plane HA", func() {
				BeforeEach(func() {
					os.Setenv(IsControlPlaneHAProvider, "False")
				})
				AfterEach(func() {
					os.Unsetenv(IsControlPlaneHAProvider)
				})
				It("should return True", func() {
					Expect(IsControlPlaneVIPProvider(legacyCluster)).Should(Equal(false))
				})
			})
		})

		Context("Cluster Class Cluster Case", func() {
			When("ako operator provides control plane HA", func() {
				It("should return True", func() {
					Expect(IsControlPlaneVIPProvider(clusterClassCluster)).Should(Equal(true))
				})
			})
			When("ako operator doesn't provide control plane HA", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					boolRaw, _ := json.Marshal(false)
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: AviAPIServerHAProvider,
							Value: v1.JSON{
								Raw: boolRaw,
							},
						},
					}
				})
				It("should return false", func() {
					Expect(IsControlPlaneVIPProvider(cluster)).Should(Equal(false))
				})

				When("ako operator doesn't provide control plane HA", func() {
					var cluster *clusterv1.Cluster
					BeforeEach(func() {
						cluster = clusterClassCluster.DeepCopy()
						errRaw, _ := json.Marshal("test")
						cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
							{
								Name: AviAPIServerHAProvider,
								Value: v1.JSON{
									Raw: errRaw,
								},
							},
						}
					})
					It("should return false", func() {
						Expect(IsControlPlaneVIPProvider(cluster)).Should(Equal(false))
					})
				})
			})
		})
	})

	Context("ako operator load balancer provider", func() {
		Context("Legacy Cluster Cases", func() {
			When("ako operator is load balancer provider", func() {
				It("should return True", func() {
					Expect(IsLoadBalancerProvider(nil)).Should(Equal(true))
				})
				It("should return True", func() {
					Expect(IsLoadBalancerProvider(legacyCluster)).Should(Equal(true))
				})
			})
		})

		Context("Cluster Class Cluster Case", func() {
			When("ako operator doesn't provide load balancer", func() {
				It("should return false", func() {
					Expect(IsLoadBalancerProvider(clusterClassCluster)).Should(Equal(false))
				})
			})
			When("ako operator is load balancer provider", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					boolRaw, _ := json.Marshal(false)
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: KubeVipLoadBalancerProvider,
							Value: v1.JSON{
								Raw: boolRaw,
							},
						},
					}
				})
				It("should return true", func() {
					Expect(IsLoadBalancerProvider(cluster)).Should(Equal(true))
				})
			})
		})
	})

	Context("Get control plane endpoint port", func() {
		Context("Legacy cluster Cases", func() {
			When("There is a valid control plane endpoint port", func() {
				BeforeEach(func() {
					os.Setenv(ControlPlaneEndpointPort, "6001")
				})
				AfterEach(func() {
					os.Unsetenv(ControlPlaneEndpointPort)
				})
				It("should return port in env", func() {
					Expect(GetControlPlaneEndpointPort(nil)).Should(Equal(int32(6001)))
				})
			})

			When("There is a valid control plane endpoint port", func() {
				BeforeEach(func() {
					os.Setenv(ControlPlaneEndpointPort, "6001")
				})
				AfterEach(func() {
					os.Unsetenv(ControlPlaneEndpointPort)
				})
				It("should return port in env", func() {
					Expect(GetControlPlaneEndpointPort(legacyCluster)).Should(Equal(int32(6001)))
				})
			})

			When("There is an invalid control plane endpoint port", func() {
				BeforeEach(func() {
					os.Setenv(ControlPlaneEndpointPort, "0")
				})
				AfterEach(func() {
					os.Unsetenv(ControlPlaneEndpointPort)
				})
				It("should return port 6443", func() {
					Expect(GetControlPlaneEndpointPort(legacyCluster)).Should(Equal(int32(6443)))
				})
			})
		})

		Context("Cluster Class Cases", func() {
			When("There is a valid control plane endpoint port", func() {
				It("should return 31005 as expected", func() {
					Expect(GetControlPlaneEndpointPort(clusterClassCluster)).Should(Equal(int32(31005)))
				})
			})
			When("There is an invalid control plane endpoint port", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					intRaw, _ := json.Marshal(65537)
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: ApiServerPort,
							Value: v1.JSON{
								Raw: intRaw,
							},
						},
					}
				})
				It("should return default value 6443", func() {
					Expect(GetControlPlaneEndpointPort(cluster)).Should(Equal(int32(6443)))
				})
			})
		})
	})
})
