// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package env

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var servicesListRaw = `
{
    "apiVersion": "v1",
    "items": [
        {
            "apiVersion": "v1",
            "kind": "Service",
            "metadata": {
                "creationTimestamp": "2021-01-20T22:48:36Z",
                "labels": {
                    "component": "apiserver",
                    "provider": "kubernetes"
                },
                "managedFields": [
                    {
                        "apiVersion": "v1",
                        "fieldsType": "FieldsV1",
                        "fieldsV1": {
                            "f:metadata": {
                                "f:labels": {
                                    ".": {},
                                    "f:component": {},
                                    "f:provider": {}
                                }
                            },
                            "f:spec": {
                                "f:clusterIP": {},
                                "f:ports": {
                                    ".": {},
                                    "k:{\"port\":443,\"protocol\":\"TCP\"}": {
                                        ".": {},
                                        "f:name": {},
                                        "f:port": {},
                                        "f:protocol": {},
                                        "f:targetPort": {}
                                    }
                                },
                                "f:sessionAffinity": {},
                                "f:type": {}
                            }
                        },
                        "manager": "kube-apiserver",
                        "operation": "Update",
                        "time": "2021-01-20T22:48:36Z"
                    }
                ],
                "name": "kubernetes",
                "namespace": "default",
                "resourceVersion": "225",
                "selfLink": "/api/v1/namespaces/default/services/kubernetes",
                "uid": "fe05f4a0-7e86-4aa4-a2a6-292f18b9e4a7"
            },
            "spec": {
                "clusterIP": "100.64.0.1",
                "ports": [
                    {
                        "name": "https",
                        "port": 443,
                        "protocol": "TCP",
                        "targetPort": 6443
                    }
                ],
                "sessionAffinity": "None",
                "type": "ClusterIP"
            },
            "status": {
                "loadBalancer": {}
            }
        },
        {
            "apiVersion": "v1",
            "kind": "Service",
            "metadata": {
                "annotations": {
                    "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"name\":\"nginx-service\",\"namespace\":\"default\"},\"spec\":{\"ports\":[{\"port\":80,\"protocol\":\"TCP\",\"targetPort\":80}],\"selector\":{\"app\":\"nginx\"},\"type\":\"LoadBalancer\"}}\n"
                },
                "creationTimestamp": "2021-01-20T22:54:08Z",
                "managedFields": [
                    {
                        "apiVersion": "v1",
                        "fieldsType": "FieldsV1",
                        "fieldsV1": {
                            "f:metadata": {
                                "f:annotations": {
                                    ".": {},
                                    "f:kubectl.kubernetes.io/last-applied-configuration": {}
                                }
                            },
                            "f:spec": {
                                "f:externalTrafficPolicy": {},
                                "f:ports": {
                                    ".": {},
                                    "k:{\"port\":80,\"protocol\":\"TCP\"}": {
                                        ".": {},
                                        "f:port": {},
                                        "f:protocol": {},
                                        "f:targetPort": {}
                                    }
                                },
                                "f:selector": {
                                    ".": {},
                                    "f:app": {}
                                },
                                "f:sessionAffinity": {},
                                "f:type": {}
                            }
                        },
                        "manager": "kubectl-client-side-apply",
                        "operation": "Update",
                        "time": "2021-01-20T22:54:08Z"
                    },
                    {
                        "apiVersion": "v1",
                        "fieldsType": "FieldsV1",
                        "fieldsV1": {
                            "f:status": {
                                "f:loadBalancer": {
                                    "f:ingress": {}
                                }
                            }
                        },
                        "manager": "akc",
                        "operation": "Update",
                        "time": "2021-01-20T22:54:13Z"
                    }
                ],
                "name": "nginx-service",
                "namespace": "default",
                "resourceVersion": "2199",
                "selfLink": "/api/v1/namespaces/default/services/nginx-service",
                "uid": "b9940994-d6b4-4184-bf89-f1cae197cde2"
            },
            "spec": {
                "clusterIP": "100.66.159.93",
                "externalTrafficPolicy": "Cluster",
                "ports": [
                    {
                        "nodePort": 30203,
                        "port": 80,
                        "protocol": "TCP",
                        "targetPort": 80
                    }
                ],
                "selector": {
                    "app": "nginx"
                },
                "sessionAffinity": "None",
                "type": "LoadBalancer"
            },
            "status": {
                "loadBalancer": {
                    "ingress": [
                        {
                            "ip": "10.206.101.135"
                        }
                    ]
                }
            }
        },
        {
            "apiVersion": "v1",
            "kind": "Service",
            "metadata": {
                "annotations": {
                    "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"labels\":{\"service.kubernetes.io/service-proxy-name\":\"nsx-t\"},\"name\":\"static-ip\",\"namespace\":\"default\"},\"spec\":{\"externalTrafficPolicy\":\"Local\",\"healthCheckNodePort\":32126,\"ports\":[{\"port\":80,\"protocol\":\"TCP\",\"targetPort\":80}],\"selector\":{\"app\":\"static-ip\"},\"type\":\"LoadBalancer\"}}\n"
                },
                "creationTimestamp": "2021-01-22T21:08:38Z",
                "managedFields": [
                    {
                        "apiVersion": "v1",
                        "fieldsType": "FieldsV1",
                        "fieldsV1": {
                            "f:metadata": {
                                "f:annotations": {
                                    ".": {},
                                    "f:kubectl.kubernetes.io/last-applied-configuration": {}
                                }
                            },
                            "f:spec": {
                                "f:ports": {
                                    ".": {},
                                    "k:{\"port\":80,\"protocol\":\"TCP\"}": {
                                        ".": {},
                                        "f:port": {},
                                        "f:protocol": {},
                                        "f:targetPort": {}
                                    }
                                },
                                "f:selector": {
                                    ".": {},
                                    "f:app": {}
                                },
                                "f:sessionAffinity": {},
                                "f:type": {}
                            }
                        },
                        "manager": "kubectl-client-side-apply",
                        "operation": "Update",
                        "time": "2021-01-22T21:08:38Z"
                    },
                    {
                        "apiVersion": "v1",
                        "fieldsType": "FieldsV1",
                        "fieldsV1": {
                            "f:status": {
                                "f:loadBalancer": {
                                    "f:ingress": {}
                                }
                            }
                        },
                        "manager": "akc",
                        "operation": "Update",
                        "time": "2021-01-22T21:08:45Z"
                    },
                    {
                        "apiVersion": "v1",
                        "fieldsType": "FieldsV1",
                        "fieldsV1": {
                            "f:spec": {
                                "f:externalTrafficPolicy": {}
                            }
                        },
                        "manager": "kubectl-edit",
                        "operation": "Update",
                        "time": "2021-01-22T21:09:08Z"
                    }
                ],
                "name": "static-ip",
                "namespace": "default",
                "resourceVersion": "657159",
                "selfLink": "/api/v1/namespaces/default/services/static-ip",
                "uid": "185c84b4-6205-464e-afbf-b20460539282"
            },
            "spec": {
                "clusterIP": "100.64.27.54",
                "externalTrafficPolicy": "Cluster",
                "ports": [
                    {
                        "nodePort": 32211,
                        "port": 80,
                        "protocol": "TCP",
                        "targetPort": 80
                    }
                ],
                "selector": {
                    "app": "static-ip"
                },
                "sessionAffinity": "None",
                "type": "LoadBalancer"
            },
            "status": {
                "loadBalancer": {
                    "ingress": [
                        {
                            "ip": "10.206.104.2"
                        }
                    ]
                }
            }
        }
    ],
    "kind": "List",
    "metadata": {
        "resourceVersion": "",
        "selfLink": ""
    }
}
`

var _ = Describe("kubectl unit tests", func() {
	var (
		serviceListRawBytes []byte
	)

	BeforeEach(func() {
		serviceListRawBytes = []byte(servicesListRaw)
	})

	Context("getLoadBalancerTypeServiceIPsFromJson", func() {
		When("there are one lb type services in the list", func() {
			It("should return two ips", func() {
				ips, err := getLoadBalancerTypeServiceIPsFromJson(serviceListRawBytes)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ips)).Should(Equal(2))
				sort.Strings(ips)
				expected := []string{"10.206.104.2", "10.206.101.135"}
				sort.Strings(expected)
				Expect(expected).To(Equal(ips))
			})
		})
	})
})
