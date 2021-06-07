// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig_test

import (
	"bytes"
	"net"

	"github.com/avinetworks/sdk/go/models"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers/akodeploymentconfig"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func unitTestEnsureStaticRanges() {

	Context("sorting static ranges", func() {
		var (
			subnet   *models.Subnet
			addrType string
		)
		BeforeEach(func() {
			addrType = "V4"
		})
		JustBeforeEach(func() {
			akodeploymentconfig.SortStaticRanges(subnet.StaticRanges)
		})
		When("subnet static ranges are not sorted", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.5", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.7", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						},
					},
				}
			})
			It("should sort the subnets according to the Begin addr", func() {
				var prev []byte
				for i := 0; i < len(subnet.StaticRanges); i++ {
					if i == 0 {
						prev = net.ParseIP(*subnet.StaticRanges[0].Begin.Addr)
						continue
					}
					cur := net.ParseIP(*subnet.StaticRanges[i].Begin.Addr)
					Expect(bytes.Compare(prev, cur) < 0).To(Equal(true))
				}
			})
		})
	})
	Context("IsStaticRangeEqual", func() {
		var (
			r1  []*models.IPAddrRange
			r2  []*models.IPAddrRange
			res bool
		)
		JustBeforeEach(func() {
			res = akodeploymentconfig.IsStaticRangeEqual(r1, r2)
		})
		When("both are empty", func() {
			BeforeEach(func() {
				r1 = []*models.IPAddrRange{}
				r2 = []*models.IPAddrRange{}
			})
			It("should be equal", func() {
				Expect(res).To(BeTrue())
			})
		})
		When("r1 equals r2", func() {
			BeforeEach(func() {
				r1 = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.1", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.3", "V4"),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.9", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.12", "V4"),
					},
				}
				r2 = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.1", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.3", "V4"),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.9", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.12", "V4"),
					},
				}
			})
			It("should be equal", func() {
				Expect(res).To(BeTrue())
			})
		})
		When("r1 doesn't equal r2", func() {
			BeforeEach(func() {
				r1 = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.1", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.4", "V4"),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.9", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.12", "V4"),
					},
				}
				r2 = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.1", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.3", "V4"),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.9", "V4"),
						End:   akodeploymentconfig.GetAddr("192.168.100.12", "V4"),
					},
				}
			})
			It("should not be equal", func() {
				Expect(res).NotTo(BeTrue())
			})
		})
	})
	Context("EnsureStaticRanges", func() {
		var (
			modified         bool
			expected         []*models.IPAddrRange
			expectedModified bool
			subnet           *models.Subnet
			ipPools          []akoov1alpha1.IPPool
			addrType         string
		)
		BeforeEach(func() {
			addrType = "V4"
		})
		JustBeforeEach(func() {
			modified = akodeploymentconfig.EnsureStaticRanges(subnet, ipPools, addrType)
		})
		When("all the intervals are contiguous", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.4", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.7", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.10", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.200", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.202", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.3",
						End:   "192.168.100.7",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.10",
						End:   "192.168.100.202",
						Type:  addrType,
					},
				}
				expected = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.7", addrType),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.10", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.202", addrType),
					},
				}
				expectedModified = true
			})
			It("should update to ip pools", func() {
				Expect(modified).To(Equal(expectedModified))
				if modified {
					Expect(akodeploymentconfig.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
		When("static ranges is the superset of ip pools", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.20", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.192", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.200", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.3",
						End:   "192.168.100.7",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.13",
						End:   "192.168.100.18",
						Type:  addrType,
					},
				}
				expected = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.7", addrType),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.13", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.18", addrType),
					},
				}
				expectedModified = true
			})
			It("should update to ip pools", func() {
				Expect(modified).To(Equal(expectedModified))
				if modified {
					Expect(akodeploymentconfig.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
		When("ip pools is the superset of static ranges", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.5", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.8", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.12", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.16", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.192", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.200", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.1",
						End:   "192.168.100.21",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.100",
						End:   "192.168.100.203",
						Type:  addrType,
					},
				}
				expected = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.21", addrType),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.100", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.203", addrType),
					},
				}
				expectedModified = true
			})
			It("should update to ip pools", func() {
				Expect(modified).To(Equal(expectedModified))
				if modified {
					Expect(akodeploymentconfig.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
		When("some of the intervals are contiguous", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.4", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.192", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.200", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.3",
						End:   "192.168.100.7",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.21",
						End:   "192.168.100.25",
						Type:  addrType,
					},
				}
				expected = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.7", addrType),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.21", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.25", addrType),
					},
				}
				expectedModified = true
			})
			It("should update to ip pools", func() {
				Expect(modified).To(Equal(expectedModified))
				if modified {
					Expect(akodeploymentconfig.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
		When("static range is empty", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{},
				}
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.3",
						End:   "192.168.100.17",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.1",
						End:   "192.168.100.2",
						Type:  addrType,
					},
				}
				expected = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.2", addrType),
					},
					&models.IPAddrRange{
						Begin: akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						End:   akodeploymentconfig.GetAddr("192.168.100.17", addrType),
					},
				}
				expectedModified = true
			})
		})
		When("ip pool is empty", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.192", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.200", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.4", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{}
				expected = []*models.IPAddrRange{}
				expectedModified = true
			})
			It("should update to ip pools", func() {
				Expect(modified).To(Equal(expectedModified))
				if modified {
					Expect(akodeploymentconfig.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
		When("staticRanges and ipPools are identical", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.3", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.7", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.10", addrType),
						},
						&models.IPAddrRange{
							Begin: akodeploymentconfig.GetAddr("192.168.100.121", addrType),
							End:   akodeploymentconfig.GetAddr("192.168.100.134", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.121",
						End:   "192.168.100.134",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.7",
						End:   "192.168.100.10",
						Type:  addrType,
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.1",
						End:   "192.168.100.3",
						Type:  addrType,
					},
				}
				expectedModified = false
			})
			It("should not change anything", func() {
				Expect(modified).To(Equal(expectedModified))
				if modified {
					Expect(akodeploymentconfig.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
	})
	Context("EnsureAviNetwork", func() {
		var (
			cidr     *net.IPNet
			mask     int32
			network  *models.Network
			ipPools  []akoov1alpha1.IPPool
			logger   logr.Logger
			addrType string

			modified         bool
			expected         []*models.IPAddrRange
			expectedModified bool
		)
		BeforeEach(func() {
			addrType = "V4"
			_, cidr, _ = net.ParseCIDR("192.168.100.0/24")
			mask = 24
			log.SetLogger(zap.New())
			logger = log.Log
		})
		JustBeforeEach(func() {
			modified = akodeploymentconfig.EnsureAviNetwork(network, addrType, cidr, mask, ipPools, logger)
			expected = akodeploymentconfig.CreateStaticRangeFromIPPools(ipPools)
		})
		When("update is needed", func() {
			When("ip pools overlaps with static ranges", func() {
				BeforeEach(func() {
					network = &models.Network{
						ConfiguredSubnets: []*models.Subnet{
							&models.Subnet{
								Prefix: &models.IPAddrPrefix{
									IPAddr: akodeploymentconfig.GetAddr("192.168.100.0", addrType),
									Mask:   &mask,
								},
								StaticRanges: []*models.IPAddrRange{
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.30", addrType),
									},
								},
							},
						},
					}
					ipPools = []akoov1alpha1.IPPool{
						akoov1alpha1.IPPool{
							Start: "192.168.100.8",
							End:   "192.168.100.31",
							Type:  addrType,
						},
						akoov1alpha1.IPPool{
							Start: "192.168.100.100",
							End:   "192.168.100.101",
							Type:  addrType,
						},
					}
					expectedModified = true
				})
				It("should update static range to ip pools", func() {
					Expect(modified).To(Equal(expectedModified))
					if modified {
						index, contains := akodeploymentconfig.AviNetworkContainsSubnet(network, cidr.IP.String(), mask)
						Expect(contains).To(BeTrue())
						Expect(akodeploymentconfig.IsStaticRangeEqual(network.ConfiguredSubnets[index].StaticRanges, expected)).To(BeTrue())
					}
				})
			})
			When("static range is the super set of ip pools", func() {
				BeforeEach(func() {
					network = &models.Network{
						ConfiguredSubnets: []*models.Subnet{
							&models.Subnet{
								Prefix: &models.IPAddrPrefix{
									IPAddr: akodeploymentconfig.GetAddr("192.168.100.0", addrType),
									Mask:   &mask,
								},
								StaticRanges: []*models.IPAddrRange{
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.30", addrType),
									},
								},
							},
						},
					}
					ipPools = []akoov1alpha1.IPPool{
						akoov1alpha1.IPPool{
							Start: "192.168.100.1",
							End:   "192.168.100.8",
							Type:  addrType,
						},
						akoov1alpha1.IPPool{
							Start: "192.168.100.14",
							End:   "192.168.100.17",
							Type:  addrType,
						},
					}
					expectedModified = true
				})
				It("should update static range to ip pools", func() {
					Expect(modified).To(Equal(expectedModified))
					if modified {
						index, contains := akodeploymentconfig.AviNetworkContainsSubnet(network, cidr.IP.String(), mask)
						Expect(contains).To(BeTrue())
						Expect(akodeploymentconfig.IsStaticRangeEqual(network.ConfiguredSubnets[index].StaticRanges, expected)).To(BeTrue())
					}
				})
			})
			When("ipPools is the superset of static ranges", func() {
				BeforeEach(func() {
					network = &models.Network{
						ConfiguredSubnets: []*models.Subnet{
							&models.Subnet{
								Prefix: &models.IPAddrPrefix{
									IPAddr: akodeploymentconfig.GetAddr("192.168.100.0", addrType),
									Mask:   &mask,
								},
								StaticRanges: []*models.IPAddrRange{
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.3", addrType),
									},
								},
							},
						},
					}
					ipPools = []akoov1alpha1.IPPool{
						akoov1alpha1.IPPool{
							Start: "192.168.100.1",
							End:   "192.168.100.8",
							Type:  addrType,
						},
						akoov1alpha1.IPPool{
							Start: "192.168.100.14",
							End:   "192.168.100.17",
							Type:  addrType,
						},
					}
					expectedModified = true
				})
				It("should update static range to ip pools", func() {
					Expect(modified).To(Equal(expectedModified))
					if modified {
						index, contains := akodeploymentconfig.AviNetworkContainsSubnet(network, cidr.IP.String(), mask)
						Expect(contains).To(BeTrue())
						Expect(akodeploymentconfig.IsStaticRangeEqual(network.ConfiguredSubnets[index].StaticRanges, expected)).To(BeTrue())
					}
				})
			})
		})
		When("there is no change", func() {
			When("network has matching subnet and ipPools didn't specify", func() {
				BeforeEach(func() {
					network = &models.Network{
						ConfiguredSubnets: []*models.Subnet{
							&models.Subnet{
								Prefix: &models.IPAddrPrefix{
									IPAddr: akodeploymentconfig.GetAddr("192.168.100.0", addrType),
									Mask:   &mask,
								},
								StaticRanges: []*models.IPAddrRange{
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.5", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.7", addrType),
									},
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.3", addrType),
									},
								},
							},
						},
					}
					ipPools = nil
					expectedModified = false
				})
				It("should not update anything", func() {
					Expect(modified).To(Equal(expectedModified))
					if modified {
						index, contains := akodeploymentconfig.AviNetworkContainsSubnet(network, cidr.IP.String(), mask)
						Expect(contains).To(BeTrue())
						Expect(akodeploymentconfig.IsStaticRangeEqual(network.ConfiguredSubnets[index].StaticRanges, expected)).To(BeTrue())
					}
				})
			})
			When("network has matching subnet, ipPools is not empty, and they're identical", func() {
				BeforeEach(func() {
					network = &models.Network{
						ConfiguredSubnets: []*models.Subnet{
							&models.Subnet{
								Prefix: &models.IPAddrPrefix{
									IPAddr: akodeploymentconfig.GetAddr("192.168.100.0", addrType),
									Mask:   &mask,
								},
								StaticRanges: []*models.IPAddrRange{
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.5", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.7", addrType),
									},
									&models.IPAddrRange{
										Begin: akodeploymentconfig.GetAddr("192.168.100.1", addrType),
										End:   akodeploymentconfig.GetAddr("192.168.100.3", addrType),
									},
								},
							},
						},
					}
					ipPools = []akoov1alpha1.IPPool{
						akoov1alpha1.IPPool{
							Start: "192.168.100.1",
							End:   "192.168.100.3",
							Type:  addrType,
						},
						akoov1alpha1.IPPool{
							Start: "192.168.100.5",
							End:   "192.168.100.7",
							Type:  addrType,
						},
					}
					expectedModified = false
				})
				It("should not update anything", func() {
					Expect(modified).To(Equal(expectedModified))
				})
			})
		})
	})
}
