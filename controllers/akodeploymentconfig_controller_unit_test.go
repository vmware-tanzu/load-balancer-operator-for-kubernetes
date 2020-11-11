// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers_test

import (
	"bytes"
	"net"

	"github.com/avinetworks/sdk/go/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/controllers"
)

func unitTestEnsureStaticRanges() {
	var (
		subnet   *models.Subnet
		ipPools  []akoov1alpha1.IPPool
		addrType string
	)

	Context("sorting subnets", func() {
		JustBeforeEach(func() {
			controllers.SortSubnetStaticRanges(subnet)
		})
		When("subnet is nil", func() {
			It("should not panic", func() {})
		})
		When("subnet static ranges are not sorted", func() {
			BeforeEach(func() {
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.5", "V4"),
							End:   controllers.GetAddr("192.168.100.7", "V4"),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.1", "V4"),
							End:   controllers.GetAddr("192.168.100.3", "V4"),
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
	Context("sorting ippools", func() {
		JustBeforeEach(func() {
			controllers.SortIPPools(ipPools)
		})
		When("ippools are not sorted", func() {
			BeforeEach(func() {
				ipPools = []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "192.168.100.5",
						End:   "192.168.100.7",
						Type:  "V4",
					},
					akoov1alpha1.IPPool{
						Start: "192.168.100.1",
						End:   "192.168.100.3",
						Type:  "V4",
					},
				}
			})
			It("should sort the ipPools according to the Begin addr", func() {
				var prev []byte
				for i := 0; i < len(ipPools); i++ {
					if i == 0 {
						prev = net.ParseIP(ipPools[i].Start)
						continue
					}
					cur := net.ParseIP(ipPools[i].Start)
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
			res = controllers.IsStaticRangeEqual(r1, r2)
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
						Begin: controllers.GetAddr("192.168.100.1", "V4"),
						End:   controllers.GetAddr("192.168.100.3", "V4"),
					},
					&models.IPAddrRange{
						Begin: controllers.GetAddr("192.168.100.9", "V4"),
						End:   controllers.GetAddr("192.168.100.12", "V4"),
					},
				}
				r2 = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: controllers.GetAddr("192.168.100.1", "V4"),
						End:   controllers.GetAddr("192.168.100.3", "V4"),
					},
					&models.IPAddrRange{
						Begin: controllers.GetAddr("192.168.100.9", "V4"),
						End:   controllers.GetAddr("192.168.100.12", "V4"),
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
						Begin: controllers.GetAddr("192.168.100.1", "V4"),
						End:   controllers.GetAddr("192.168.100.4", "V4"),
					},
					&models.IPAddrRange{
						Begin: controllers.GetAddr("192.168.100.9", "V4"),
						End:   controllers.GetAddr("192.168.100.12", "V4"),
					},
				}
				r2 = []*models.IPAddrRange{
					&models.IPAddrRange{
						Begin: controllers.GetAddr("192.168.100.1", "V4"),
						End:   controllers.GetAddr("192.168.100.3", "V4"),
					},
					&models.IPAddrRange{
						Begin: controllers.GetAddr("192.168.100.9", "V4"),
						End:   controllers.GetAddr("192.168.100.12", "V4"),
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
			modified bool
			expected []*models.IPAddrRange
		)
		JustBeforeEach(func() {
			modified = controllers.EnsureStaticRanges(subnet, ipPools, addrType)
		})
		When("staticRanges and ipPools are non-overlapping", func() {
			When("all the intervals are contiguous", func() {
				BeforeEach(func() {
					addrType = "V4"
					subnet = &models.Subnet{
						StaticRanges: []*models.IPAddrRange{
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.1", addrType),
								End:   controllers.GetAddr("192.168.100.4", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.7", addrType),
								End:   controllers.GetAddr("192.168.100.10", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.200", addrType),
								End:   controllers.GetAddr("192.168.100.202", addrType),
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
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.200", addrType),
						},
					}
				})
				It("should merge everything into one", func() {
					Expect(modified).To(BeTrue())
					Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
				})
			})
			When("static ranges is the superset of ip pools", func() {
				BeforeEach(func() {
					addrType = "V4"
					subnet = &models.Subnet{
						StaticRanges: []*models.IPAddrRange{
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.1", addrType),
								End:   controllers.GetAddr("192.168.100.20", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.192", addrType),
								End:   controllers.GetAddr("192.168.100.200", addrType),
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
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.20", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.192", addrType),
							End:   controllers.GetAddr("192.168.100.200", addrType),
						},
					}
				})
				It("should not change anything", func() {
					Expect(modified).NotTo(BeTrue())
					if modified {
						Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
					}
				})
			})
			When("ip pools is the superset of static ranges", func() {
				BeforeEach(func() {
					addrType = "V4"
					subnet = &models.Subnet{
						StaticRanges: []*models.IPAddrRange{
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.1", addrType),
								End:   controllers.GetAddr("192.168.100.3", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.5", addrType),
								End:   controllers.GetAddr("192.168.100.8", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.12", addrType),
								End:   controllers.GetAddr("192.168.100.16", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.192", addrType),
								End:   controllers.GetAddr("192.168.100.200", addrType),
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
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.21", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.100", addrType),
							End:   controllers.GetAddr("192.168.100.203", addrType),
						},
					}
				})
				It("should be equal to ippools", func() {
					Expect(modified).To(BeTrue())
					Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
				})
			})
			When("some of the intervals are contiguous", func() {
				BeforeEach(func() {
					addrType = "V4"
					subnet = &models.Subnet{
						StaticRanges: []*models.IPAddrRange{
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.1", addrType),
								End:   controllers.GetAddr("192.168.100.4", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.192", addrType),
								End:   controllers.GetAddr("192.168.100.200", addrType),
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
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.7", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.21", addrType),
							End:   controllers.GetAddr("192.168.100.25", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.192", addrType),
							End:   controllers.GetAddr("192.168.100.200", addrType),
						},
					}
				})
				It("should merge those overlapped", func() {
					Expect(modified).To(BeTrue())
					Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
				})
			})
			When("static range is empty", func() {
				BeforeEach(func() {
					addrType = "V4"
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
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.2", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.3", addrType),
							End:   controllers.GetAddr("192.168.100.17", addrType),
						},
					}
				})
				It("should add ippools to the static range", func() {
					Expect(modified).To(BeTrue())
					Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
				})
			})
			When("ip pool is empty", func() {
				BeforeEach(func() {
					addrType = "V4"
					subnet = &models.Subnet{
						StaticRanges: []*models.IPAddrRange{
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.192", addrType),
								End:   controllers.GetAddr("192.168.100.200", addrType),
							},
							&models.IPAddrRange{
								Begin: controllers.GetAddr("192.168.100.1", addrType),
								End:   controllers.GetAddr("192.168.100.4", addrType),
							},
						},
					}
					ipPools = []akoov1alpha1.IPPool{}
					expected = []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.4", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.192", addrType),
							End:   controllers.GetAddr("192.168.100.200", addrType),
						},
					}
				})
				It("should not change anything", func() {
					Expect(modified).ToNot(BeTrue())
					if modified {
						Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
					}
				})
			})
		})
		When("staticRanges and ipPools are identical", func() {
			BeforeEach(func() {
				addrType = "V4"
				subnet = &models.Subnet{
					StaticRanges: []*models.IPAddrRange{
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.1", addrType),
							End:   controllers.GetAddr("192.168.100.3", addrType),
						},
						&models.IPAddrRange{
							Begin: controllers.GetAddr("192.168.100.7", addrType),
							End:   controllers.GetAddr("192.168.100.10", addrType),
						},
					},
				}
				ipPools = []akoov1alpha1.IPPool{
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
			})
			It("should not change anything", func() {
				Expect(modified).NotTo(BeTrue())
				if !modified {
					Expect(controllers.IsStaticRangeEqual(subnet.StaticRanges, expected))
				}
			})
		})
	})
}
