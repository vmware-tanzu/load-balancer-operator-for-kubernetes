// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	stringRaw, _ := json.Marshal("10.1.1.1")
	legacyCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "legacy-cluster",
			Namespace:   "default",
			Annotations: map[string]string{},
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
						Value: apiextensionsv1.JSON{
							Raw: boolRaw,
						},
					},
					{
						Name: KubeVipLoadBalancerProvider,
						Value: apiextensionsv1.JSON{
							Raw: boolRaw,
						},
					},
					{
						Name: ApiServerPort,
						Value: apiextensionsv1.JSON{
							Raw: intRaw,
						},
					},
					{
						Name: ApiServerEndpoint,
						Value: apiextensionsv1.JSON{
							Raw: stringRaw,
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

	Context("If ako operator is deployed in bootstrap cluster", func() {
		When("set cluster_class_enabled to true ", func() {
			BeforeEach(func() {
				os.Setenv(ClusterClassEnabled, "True")
			})
			AfterEach(func() {
				os.Unsetenv(ClusterClassEnabled)
			})
			It("should return True", func() {
				Expect(IsClusterClassEnabled()).Should(Equal(true))
			})
		})
		When("set cluster_class_enabled to false ", func() {
			BeforeEach(func() {
				os.Setenv(ClusterClassEnabled, "False")
			})
			AfterEach(func() {
				os.Unsetenv(ClusterClassEnabled)
			})
			It("should return false", func() {
				Expect(IsClusterClassEnabled()).Should(Equal(false))
			})
		})
		When("didn't set cluster_class_enabled", func() {
			It("should return false", func() {
				Expect(IsClusterClassEnabled()).Should(Equal(false))
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
					isVIPProvider, err := IsControlPlaneVIPProvider(legacyCluster)
					Expect(isVIPProvider).Should(Equal(true))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("ako operator doesn't provide control plane HA", func() {
				BeforeEach(func() {
					os.Setenv(IsControlPlaneHAProvider, "False")
				})
				AfterEach(func() {
					os.Unsetenv(IsControlPlaneHAProvider)
				})
				It("should return False", func() {
					isVIPProvider, err := IsControlPlaneVIPProvider(legacyCluster)
					Expect(isVIPProvider).Should(Equal(false))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Context("Cluster Class Cluster Case", func() {
			When("ako operator provides control plane HA", func() {
				It("should return True", func() {
					isVIPProvider, err := IsControlPlaneVIPProvider(clusterClassCluster)
					Expect(isVIPProvider).Should(Equal(true))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("ako operator doesn't provide control plane HA", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{}
				})
				It("should return false", func() {
					isVIPProvider, err := IsControlPlaneVIPProvider(cluster)
					Expect(isVIPProvider).Should(Equal(false))
					Expect(err).ShouldNot(HaveOccurred())
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
							Value: apiextensionsv1.JSON{
								Raw: boolRaw,
							},
						},
					}
				})
				It("should return false", func() {
					isVIPProvider, err := IsControlPlaneVIPProvider(cluster)
					Expect(isVIPProvider).Should(Equal(false))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("ako operator doesn't provide control plane HA", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					errRaw := []byte("test")
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: AviAPIServerHAProvider,
							Value: apiextensionsv1.JSON{
								Raw: errRaw,
							},
						},
					}
				})
				It("should return false", func() {
					isVIPProvider, err := IsControlPlaneVIPProvider(cluster)
					Expect(isVIPProvider).Should(Equal(false))
					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})

	Context("ako operator load balancer provider", func() {
		Context("Legacy Cluster Cases", func() {
			When("ako operator is load balancer provider", func() {
				It("should return True", func() {
					isLBProvider, err := IsLoadBalancerProvider(nil)
					Expect(isLBProvider).Should(Equal(true))
					Expect(err).ShouldNot(HaveOccurred())
				})
				It("should return True", func() {
					isLBProvider, err := IsLoadBalancerProvider(legacyCluster)
					Expect(isLBProvider).Should(Equal(true))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Context("Cluster Class Cluster Case", func() {
			When("ako operator doesn't provide load balancer", func() {
				It("should return false", func() {
					isLBProvider, err := IsLoadBalancerProvider(clusterClassCluster)
					Expect(isLBProvider).Should(Equal(false))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("ako operator is load balancer provider by default", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{}
				})
				It("should return true", func() {
					isLBProvider, err := IsLoadBalancerProvider(cluster)
					Expect(isLBProvider).Should(Equal(true))
					Expect(err).ShouldNot(HaveOccurred())
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
							Value: apiextensionsv1.JSON{
								Raw: boolRaw,
							},
						},
					}
				})
				It("should return true", func() {
					isLBProvider, err := IsLoadBalancerProvider(cluster)
					Expect(isLBProvider).Should(Equal(true))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("invalid input ", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					boolRaw := []byte("randoom string")
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: KubeVipLoadBalancerProvider,
							Value: apiextensionsv1.JSON{
								Raw: boolRaw,
							},
						},
					}
				})
				It("should return true", func() {
					isLBProvider, err := IsLoadBalancerProvider(cluster)
					Expect(isLBProvider).Should(Equal(true))
					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})

	Context("get cluster endpoint", func() {
		Context("Legacy Cluster Cases", func() {
			When("didn't specify cluster endpoint", func() {
				It("should not return cluster endpoint", func() {
					endpoint, err := GetControlPlaneEndpoint(legacyCluster)
					Expect(endpoint).Should(Equal(""))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("Specify cluster endpoint", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = legacyCluster.DeepCopy()
					cluster.Annotations[ClusterControlPlaneAnnotations] = "10.1.1.1"
				})
				It("should return cluster endpoint", func() {
					endpoint, err := GetControlPlaneEndpoint(cluster)
					Expect(endpoint).Should(Equal("10.1.1.1"))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Context("Cluster Class Cluster Case", func() {
			When("Specify cluster endpoint", func() {
				It("should return endpoint", func() {
					endpoint, err := GetControlPlaneEndpoint(clusterClassCluster)
					Expect(endpoint).Should(Equal("10.1.1.1"))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("Doesn't specify cluster endpoint", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{}
				})
				It("should return false", func() {
					endpoint, err := GetControlPlaneEndpoint(cluster)
					Expect(endpoint).Should(Equal(""))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("invalid input ", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					errRaw := []byte("randoom string")
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: ApiServerEndpoint,
							Value: apiextensionsv1.JSON{
								Raw: errRaw,
							},
						},
					}
				})
				It("should return true", func() {
					endpoint, err := GetControlPlaneEndpoint(cluster)
					Expect(endpoint).Should(Equal(""))
					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})

	Context("set cluster endpoint", func() {
		Context("Legacy Cluster Cases", func() {
			When("didn't specify cluster endpoint", func() {
				It("should not set cluster endpoint", func() {
					SetControlPlaneEndpoint(legacyCluster, "10.1.1.1")
					Expect(legacyCluster.Spec.Topology).Should(BeNil())

				})
			})
		})
		Context("Cluster Class Cases", func() {
			When("specify cluster endpoint", func() {
				It("should set cluster endpoint", func() {
					SetControlPlaneEndpoint(clusterClassCluster, "10.1.1.2")
					Expect(clusterClassCluster.Spec.Topology.Variables).Should(ContainElement(clusterv1.ClusterVariable{
						Name:  ApiServerEndpoint,
						Value: apiextensionsv1.JSON{Raw: []byte("\"10.1.1.2\"")},
					}))
				})
			})
		})
		Context("Cluster Class Cases", func() {
			When("specify cluster endpoint", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{}
				})
				It("should set cluster endpoint", func() {
					SetControlPlaneEndpoint(cluster, "10.1.1.3")
					Expect(cluster.Spec.Topology.Variables).Should(ContainElement(clusterv1.ClusterVariable{
						Name:  ApiServerEndpoint,
						Value: apiextensionsv1.JSON{Raw: []byte("\"10.1.1.3\"")},
					}))
				})
			})
		})
	})

	Context("Get control plane endpoint port", func() {
		Context("Legacy cluster Cases", func() {
			When("No env variables set", func() {
				It("should return port in env", func() {
					port, err := GetControlPlaneEndpointPort(legacyCluster)
					Expect(port).Should(Equal(int32(6443)))
					Expect(err).ShouldNot(HaveOccurred())
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
					port, err := GetControlPlaneEndpointPort(legacyCluster)
					Expect(port).Should(Equal(int32(6001)))
					Expect(err).ShouldNot(HaveOccurred())
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
					port, err := GetControlPlaneEndpointPort(legacyCluster)
					Expect(port).Should(Equal(int32(6001)))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("There is an invalid control plane endpoint port", func() {
				BeforeEach(func() {
					os.Setenv(ControlPlaneEndpointPort, "test")
				})
				AfterEach(func() {
					os.Unsetenv(ControlPlaneEndpointPort)
				})
				It("should return port 6443", func() {
					port, err := GetControlPlaneEndpointPort(legacyCluster)
					Expect(port).Should(Equal(int32(6443)))
					Expect(err).Should(HaveOccurred())
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
					port, err := GetControlPlaneEndpointPort(legacyCluster)
					Expect(port).Should(Equal(int32(6443)))
					Expect(err).Should(HaveOccurred())
				})
			})
		})

		Context("Cluster Class Cases", func() {
			When("There is a valid control plane endpoint port", func() {
				It("should return 31005 as expected", func() {
					port, err := GetControlPlaneEndpointPort(clusterClassCluster)
					Expect(port).Should(Equal(int32(31005)))
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			When("There is no control plane endpoint port", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{}
				})
				It("should return default value 6443", func() {
					port, err := GetControlPlaneEndpointPort(cluster)
					Expect(port).Should(Equal(int32(6443)))
					Expect(err).ShouldNot(HaveOccurred())
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
							Value: apiextensionsv1.JSON{
								Raw: intRaw,
							},
						},
					}
				})
				It("should return default value 6443", func() {
					port, err := GetControlPlaneEndpointPort(cluster)
					Expect(port).Should(Equal(int32(6443)))
					Expect(err).Should(HaveOccurred())
				})
			})
			When("There is an invalid control plane endpoint port", func() {
				var cluster *clusterv1.Cluster
				BeforeEach(func() {
					cluster = clusterClassCluster.DeepCopy()
					errRaw := []byte("test")
					cluster.Spec.Topology.Variables = []clusterv1.ClusterVariable{
						{
							Name: ApiServerPort,
							Value: apiextensionsv1.JSON{
								Raw: errRaw,
							},
						},
					}
				})
				It("should return default value 6443", func() {
					port, err := GetControlPlaneEndpointPort(cluster)
					Expect(port).Should(Equal(int32(6443)))
					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})
})
