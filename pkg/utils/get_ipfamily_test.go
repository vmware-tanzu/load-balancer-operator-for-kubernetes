// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)



var _ = ginkgo.Describe("Test get primary ipFamily", func() {
	ginkgo.It("should return ipv4", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16"},
					},
					Services: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V4"))
		Expect(err).To(BeNil())
	})

	ginkgo.It("should return ipv6", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"2002::1234:abcd:ffff:c0a8:101/64"},
					},
					Services: &capi.NetworkRanges{
						CIDRBlocks: []string{"2002::1234:abcd:ffff:c0a8:101/64"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V6"))
		Expect(err).To(BeNil())
	})

	ginkgo.It("should return right ipfamily when service cidr is empty", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"2002::1234:abcd:ffff:c0a8:101/64"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V6"))
		Expect(err).To(BeNil())
	})

	ginkgo.It("should return right ipfamily when pod cidr is empty", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Services: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V4"))
		Expect(err).To(BeNil())
	})

	ginkgo.It("should return right ipfamily when service cidr is empty", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V4"))
		Expect(err).To(BeNil())
	})

	ginkgo.It("should return V4 when both podcidr and service cidr are empty", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V4"))
		Expect(err).To(BeNil())
	})

	ginkgo.It("should return INVALID, if length of podcidr is larger than 2", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16","2002::1234:abcd:ffff:c0a8:101/64","10.10.0.0/16"},
					},
					Services: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("INVALID"))
		Expect(err).NotTo(BeNil())
	})

	ginkgo.It("should return INVALID, if pod family is different from service family", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"2002::1234:abcd:ffff:c0a8:101/64"},
					},
					Services: &capi.NetworkRanges{
						CIDRBlocks: []string{"192.168.0.0/16"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("INVALID"))
		Expect(err).NotTo(BeNil())
	})

	ginkgo.It("should return primary ipfamily", func() {
		clusterClassCluster := &capi.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cluster-class-cluster",
				Namespace: "default",
			},
			Spec: capi.ClusterSpec{
				ClusterNetwork: &capi.ClusterNetwork{
					Pods: &capi.NetworkRanges{
						CIDRBlocks: []string{"2002::1234:abcd:ffff:c0a8:101/64", "192.168.0.0/16"},
					},
					Services: &capi.NetworkRanges{
						CIDRBlocks: []string{"2002::1234:abcd:ffff:c0a8:101/64", "192.168.0.0/16"},
					},
				},
			},
		}
		ipFamily, err := GetPrimaryIPFamily(clusterClassCluster)
		Expect(ipFamily).To(Equal("V6"))
		Expect(err).To(BeNil())
	})
})
