// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig_test

import (
	"github.com/avinetworks/sdk/go/models"
	"github.com/avinetworks/sdk/go/session"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	akoov1alpha1 "github.com/vmware-samples/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/ako"
	controllerruntime "github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/controller-runtime"
	"github.com/vmware-samples/load-balancer-operator-for-kubernetes/pkg/test/builder"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cluster-api/util/conditions"
	kcfg "sigs.k8s.io/cluster-api/util/kubeconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func intgTestAkoDeploymentConfigController() {
	var (
		ctx                              *builder.IntegrationTestContext
		cluster                          *clusterv1.Cluster
		akoDeploymentConfig              *akoov1alpha1.AKODeploymentConfig
		defaultAkoDeploymentConfig       *akoov1alpha1.AKODeploymentConfig
		controllerCredentials            *corev1.Secret
		controllerCA                     *corev1.Secret
		staticCluster                    *clusterv1.Cluster
		staticAkoDeploymentConfig        *akoov1alpha1.AKODeploymentConfig
		staticDefaultAkoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
		staticControllerCredentials      *corev1.Secret
		staticControllerCA               *corev1.Secret
		testLabels                       map[string]string
		err                              error

		networkUpdate        *models.Network
		userUpdateCalled     bool
		userRoleCreateCalled bool
		userCreateCalled     bool
	)

	staticCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-test-8ed12g",
			Namespace: "integration-test-8ed12g",
		},
		Spec: clusterv1.ClusterSpec{},
	}

	defaultAkoDeploymentConfigCommonSpec := akoov1alpha1.AKODeploymentConfigSpec{
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
	akoDeploymentConfigCommonSpec := defaultAkoDeploymentConfigCommonSpec.DeepCopy()
	akoDeploymentConfigCommonSpec.ClusterSelector = metav1.LabelSelector{
		MatchLabels: map[string]string{
			"test": "true",
		},
	}
	staticAkoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ako-deployment-config",
		},
		Spec: *akoDeploymentConfigCommonSpec,
	}
	staticDefaultAkoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: akoov1alpha1.WorkloadClusterAkoDeploymentConfig},
		Spec:       defaultAkoDeploymentConfigCommonSpec,
	}

	staticControllerCredentials = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "controller-credentials",
			Namespace: "default",
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("Admin!23"),
		},
	}
	staticControllerCA = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "controller-ca",
			Namespace: "default",
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"certificateAuthorityData": []byte("-----BEGIN CERTIFICATE-----MIICxzCCAa+gAwIBAgIUWxv6EsFnaXvTF4Lbwk9BucKJhgowDQYJKoZIhvcNAQELBQAwEzERMA8GA1UEAwwIZTJlLXRlc3QwHhcNMjEwMjE2MjAzNTU1WhcNMjIwMjE2MjAzNTU1WjATMREwDwYDVQQDDAhlMmUtdGVzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALxAXjEjZvandPgciqEerY7ptVqzdPIP0MHFA/ky0e7NVszgjHj5OcWAPnPD11p0zkR1tXknJRSJOeJnbJLNWTF5ApsOZWP9tUHt+TmvA3hVKZQiFb79VlF/VaVdJPb9vMYFjJyAlZj6rH8HABQ/Y9ysUozVaFaIMcx4sdIWxG0eGfmFT1Yrh1yZGnf0pESfcx4IzFZmQQIRvqKHZToaQEGX6T6oisM37qdrbWmPdQF2S3aFyoW/lDEqoVNJ5pzDdbTOf4CaqaPsNXbhQFF6LX2Q9kAnltwLxcdVN+KvN3Hqgif0jokEqmojhSK/bCatesMwImTmaYKG2+lK9dHbUCMCAwEAAaMTMBEwDwYDVR0RBAgwBocECqFgVTANBgkqhkiG9w0BAQsFAAOCAQEAcvgfGtVc9416oPbI7e11Kufy3DptOsMjFz7S5W1ifhDfRseEpv1oVgg4+qFVVBMyQgfH1DZ985TbwsozGCib4cU00/Tk7aoFy5TNC2xP8XJgJb5TDC4EaISgR2GPDsIW+BqkYX5jCDEMqnlGJBjG6V/z5OhqWUFZmb5Ly5qjxqt6JBP+E/z5fnZquFFNjhGIgnlpQDF6plKzYnJy5d3Yc5EurmYOmoQ/7gX/Sv6RTbha4UnQh4LsURT/sBurMW3fFdZsD5cH0t8SgxOeqsDa8YSr2T74BRm5rqKRGgX5Rz0TBJ9m6ViO3VwBceJFd2O/Gd6ElhJ81SM0lOp/jf5Hlg==-----END CERTIFICATE-----"),
		},
	}

	createObjects := func(objs ...client.Object) {
		for _, o := range objs {
			err = ctx.Client.Create(ctx.Context, o)
			Expect(err).To(BeNil())
		}
	}
	updateObjects := func(objs ...client.Object) {
		for _, o := range objs {
			err = ctx.Client.Update(ctx.Context, o)
			Expect(err).To(BeNil())
		}
	}
	deleteObjects := func(objs ...client.Object) {
		for _, o := range objs {
			// ignore error
			_ = ctx.Client.Delete(ctx.Context, o)
		}
	}
	getCluster := func(obj *clusterv1.Cluster, name, namespace string) error {
		err := ctx.Client.Get(ctx.Context, client.ObjectKey{
			Name:      name,
			Namespace: namespace,
		}, obj)
		return err
	}

	ensureAKODeploymentConfigFinalizerMatchExpectation := func(key client.ObjectKey, expectReconciled bool) {
		Eventually(func() bool {
			obj := &akoov1alpha1.AKODeploymentConfig{}
			err := ctx.Client.Get(ctx.Context, key, obj)
			if err != nil {
				println(err.Error())
				return false
			}
			finalizer := akoov1alpha1.AkoDeploymentConfigFinalizer
			if expectReconciled {
				if !controllerruntime.ContainsFinalizer(obj, finalizer) {
					return false
				}
			} else {
				if controllerruntime.ContainsFinalizer(obj, finalizer) {
					return false
				}
			}
			return true
		}).Should(BeTrue())
	}
	ensureClusterFinalizerMatchExpectation := func(key client.ObjectKey, expect bool) {
		Eventually(func() bool {
			obj := &clusterv1.Cluster{}
			err := ctx.Client.Get(ctx.Context, key, obj)
			if err != nil {
				return false
			}
			finalizer := akoov1alpha1.ClusterFinalizer
			return controllerruntime.ContainsFinalizer(obj, finalizer) == expect
		}).Should(BeTrue())
	}
	ensureRuntimeObjectMatchExpectation := func(key client.ObjectKey, obj client.Object, expect bool) {
		Eventually(func() bool {
			var res bool
			if err := ctx.Client.Get(ctx.Context, key, obj); err != nil {
				if apierrors.IsNotFound(err) {
					res = false
				} else {
					return false
				}
			} else {
				res = true
			}
			return res == expect
		}).Should(BeTrue())
	}

	ensureAKOAddOnSecretDeleteConfigMatchExpectation := func(key client.ObjectKey, expect bool) {
		Eventually(func() bool {
			var res bool
			obj := &corev1.Secret{}
			if err := ctx.Client.Get(ctx.Context, key, obj); err != nil {
				if apierrors.IsNotFound(err) {
					res = false
				} else {
					return false
				}
			} else {
				values, err := ako.NewValuesFromBytes(obj.Data["values.yaml"])
				if err != nil {
					return false
				}
				res = values.LoadBalancerAndIngressService.Config.AKOSettings.DeleteConfig == "true"
			}
			return res == expect
		}).Should(BeTrue())
	}

	ensureClusterAviLabelMatchExpectation := func(key client.ObjectKey, label string, expect bool) {
		Eventually(func() bool {
			obj := &clusterv1.Cluster{}
			err := ctx.Client.Get(ctx.Context, key, obj)
			if err != nil {
				return false
			}
			_, ok := obj.Labels[label]
			return expect == ok
		}).Should(BeTrue())
	}

	ensureSubnetMatchExpectation := func(newIPAddrEnd string, expect bool) {
		Eventually(func() bool {
			found := false
			for _, subnet := range networkUpdate.ConfiguredSubnets {
				for _, sr := range subnet.StaticRanges {
					if *(sr.End.Addr) == newIPAddrEnd {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			return expect == found
		}).Should(BeTrue())
	}

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
		akoDeploymentConfig = staticAkoDeploymentConfig.DeepCopy()
		defaultAkoDeploymentConfig = staticDefaultAkoDeploymentConfig.DeepCopy()
		cluster = staticCluster.DeepCopy()
		cluster.Namespace = ctx.Namespace
		controllerCredentials = staticControllerCredentials.DeepCopy()
		controllerCA = staticControllerCA.DeepCopy()

		testLabels = map[string]string{
			"test": "true",
		}
		ctx.AviClient.Network.SetGetByNameFn(func(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
			res := &models.Network{
				URL: pointer.StringPtr("10.0.0.1"),
			}
			return res, nil
		})
		ctx.AviClient.Network.SetUpdateFn(func(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error) {
			networkUpdate = obj
			return &models.Network{}, nil
		})
		ctx.AviClient.Cloud.SetGetByNameCloudFunc(func(name string, options ...session.ApiOptionsParams) (*models.Cloud, error) {
			res := &models.Cloud{
				IPAMProviderRef: pointer.StringPtr("https://10.0.0.x/api/ipamdnsproviderprofile/ipamdnsproviderprofile-f08403a1-0dc7-4f13-bda3-0ba2fa476516"),
			}
			return res, nil
		})
		ctx.AviClient.IPAMDNSProviderProfile.SetGetIPAMFunc(func(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
			res := &models.IPAMDNSProviderProfile{
				InternalProfile: &models.IPAMDNSInternalProfile{
					UsableNetworks: []*models.IPAMUsableNetwork{{NwRef: pointer.StringPtr("10.0.0.1")}},
				},
			}
			return res, nil
		})
		ctx.AviClient.IPAMDNSProviderProfile.SetUpdateIPAMFn(func(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
			return obj, nil
		})
		ctx.AviClient.User.SetGetByNameUserFunc(func(name string, options ...session.ApiOptionsParams) (*models.User, error) {
			return &models.User{}, nil
		})
		ctx.AviClient.User.SetDeleteByNameUserFunc(func(name string, options ...session.ApiOptionsParams) error {
			return nil
		})
		ctx.AviClient.User.SetUpdateUserFunc(func(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
			userUpdateCalled = true
			return &models.User{}, nil
		})
		ctx.AviClient.User.SetCreateUserFunc(func(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
			userCreateCalled = true
			return &models.User{}, nil
		})
		ctx.AviClient.Role.SetGetByNameRoleFunc(func(name string, options ...session.ApiOptionsParams) (*models.Role, error) {
			return &models.Role{}, errors.New("No object of type role with name intg-test-avi-role is found")
		})
		ctx.AviClient.Role.SetCreateRoleFunc(func(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error) {
			userRoleCreateCalled = true
			return &models.Role{}, nil
		})
		ctx.AviClient.Tenant.SetGetTenantFunc(func(uuid string, options ...session.ApiOptionsParams) (*models.Tenant, error) {
			return &models.Tenant{}, nil
		})
	})
	AfterEach(func() {
		ctx.AfterEach()
		ctx = nil
	})

	When("An AKODeploymentConfig is created", func() {
		BeforeEach(func() {
			createObjects(akoDeploymentConfig, cluster, controllerCredentials, controllerCA)
			conditions.MarkTrue(cluster, clusterv1.ReadyCondition)
			err = ctx.Client.Status().Update(ctx, cluster)
			Expect(err).To(BeNil())
			_ = kcfg.CreateSecret(ctx, ctx.Client, cluster)
		})
		AfterEach(func() {
			latestCluster := &clusterv1.Cluster{}
			if err := getCluster(latestCluster, cluster.Name, cluster.Namespace); err == nil {
				latestCluster.Finalizers = nil
				updateObjects(latestCluster)
				deleteObjects(latestCluster)
				ensureRuntimeObjectMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, &clusterv1.Cluster{}, false)
			}

			deleteObjects(akoDeploymentConfig)
			ensureRuntimeObjectMatchExpectation(client.ObjectKey{
				Name: akoDeploymentConfig.Name,
			}, &akoov1alpha1.AKODeploymentConfig{}, false)

			deleteObjects(controllerCredentials, controllerCA)
		})
		When("there is no matching cluster", func() {
			It("cluster should not have ClusterFinalizer", func() {
				ensureClusterFinalizerMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, false)
			})
		})
		When("there is a matching cluster", func() {
			BeforeEach(func() {
				cluster.Labels = testLabels
				updateObjects(cluster)

				By("should add AkoDeploymentConfigFinalizer to the AKODeploymentConfig")
				ensureAKODeploymentConfigFinalizerMatchExpectation(client.ObjectKey{
					Name: akoDeploymentConfig.Name,
				}, true)

				By("should apply Cluster Label")
				ensureClusterAviLabelMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, akoov1alpha1.AviClusterLabel, true)

				//Reconcile -> reconcileNormal -> reconcileClusters(normal phase) -> addClusterFinalizer
				By("should add Cluster Finalizer")
				ensureClusterFinalizerMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, true)

				//Reconcile -> reconcileNormal -> reconcileClusters(normal phase) -> r.clusterReconciler.ReconcileAddonSecret
				By("should Reconcile Cluster add-on secret")
				ensureRuntimeObjectMatchExpectation(client.ObjectKey{
					Name:      cluster.Name + "-load-balancer-and-ingress-service-addon",
					Namespace: cluster.Namespace,
				}, &corev1.Secret{}, true)
			})

			When("akoDeploymentConfig and cluster are created", func() {
				// Reconcile -> reconcileNormal
				When("AKODeploymentConfig is not being deleted", func() {
					// Reconcile -> reconcileNormal -> r.reconcileNetworkSubnets
					When("subnet exists and AKODeploymentConfig's subnet is contained", func() {
						BeforeEach(func() {
							ctx.AviClient.Network.SetGetByNameFn(func(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
								return &models.Network{
									URL: pointer.StringPtr("10.0.0.1"),
									ConfiguredSubnets: []*models.Subnet{{
										Prefix: &models.IPAddrPrefix{
											IPAddr: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.1"),
											},
											Mask: pointer.Int32Ptr(24),
										},
										StaticRanges: []*models.IPAddrRange{{
											Begin: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.1"),
											},
											End: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.10"),
											},
										}},
									}},
								}, nil
							})
						})
						It("shouldn't reconcile Network Subnets", func() {
							ensureSubnetMatchExpectation("10.0.0.10", true)
						})
					})

					When("subnet exists and AKODeploymentConfig's subnet is not contained", func() {
						BeforeEach(func() {
							ctx.AviClient.Network.SetGetByNameFn(func(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
								return &models.Network{
									URL: pointer.StringPtr("10.0.0.1"),
									ConfiguredSubnets: []*models.Subnet{{
										Prefix: &models.IPAddrPrefix{
											IPAddr: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.1"),
											},
											Mask: pointer.Int32Ptr(24),
										},
										StaticRanges: []*models.IPAddrRange{{
											Begin: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.1"),
											},
											End: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.5"),
											},
										}},
									}},
								}, nil
							})
						})
						It("should merge Network Subnets", func() {
							ensureSubnetMatchExpectation("10.0.0.10", true)
						})
					})

					When("subnet doesn't exist", func() {
						BeforeEach(func() {
							ctx.AviClient.Network.SetGetByNameFn(func(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
								return &models.Network{
									URL: pointer.StringPtr("10.0.0.1"),
									ConfiguredSubnets: []*models.Subnet{{
										Prefix: &models.IPAddrPrefix{
											IPAddr: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.10"),
											},
											Mask: pointer.Int32Ptr(24),
										},
										StaticRanges: []*models.IPAddrRange{{
											Begin: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.10"),
											},
											End: &models.IPAddr{
												Addr: pointer.StringPtr("10.0.0.20"),
											},
										}},
									}},
								}, nil
							})
						})
						It("shouldn't reconcile Network Subnets", func() {
							ensureSubnetMatchExpectation("10.0.0.10", true)
							ensureSubnetMatchExpectation("10.0.0.20", true)
						})
					})

					// Reconcile -> reconcileNormal -> r.reconcileCloudUsableNetwork
					// No need to test since all the functions in reconcileCloudUsableNetwork are fake functions

					When("the cluster is not deleted ", func() {
						//Reconcile -> reconcileNormal -> r.userReconciler.reconcileAviUserNormal
						When("AVI user credentials managed by tkg system", func() {
							It("should create Avi user secret", func() {
								ensureRuntimeObjectMatchExpectation(client.ObjectKey{
									Name:      cluster.Name + "-" + "avi-credentials",
									Namespace: cluster.Namespace,
								}, &corev1.Secret{}, true)
							})

							When("AVI user exists", func() {
								It("should update AVI user", func() {
									Eventually(func() bool {
										return userUpdateCalled
									}).Should(BeTrue())
								})
							})

							When("AVI user doesn't exist", func() {
								BeforeEach(func() {
									ctx.AviClient.User.SetGetByNameUserFunc(func(name string, options ...session.ApiOptionsParams) (*models.User, error) {
										return &models.User{}, errors.New("No object of type user with name intg-test-avi-user is found")
									})
								})
								It("should create role and user", func() {
									Eventually(func() bool {
										return userRoleCreateCalled && userCreateCalled
									}).Should(BeTrue())
								})
							})
						})
					})

					When("cluster has avi-delete-config label", func() {
						It("AddOnSecret disableConfig must be true when avi-delete-config label is set", func() {
							latestCluster := &clusterv1.Cluster{}
							err := getCluster(latestCluster, cluster.Name, cluster.Namespace)
							Expect(err).To(BeNil())
							latestCluster.Labels[akoov1alpha1.AviClusterDeleteConfigLabel] = "true"
							updateObjects(latestCluster)

							ensureAKOAddOnSecretDeleteConfigMatchExpectation(client.ObjectKey{
								Name:      cluster.Name + "-load-balancer-and-ingress-service-addon",
								Namespace: cluster.Namespace,
							}, true)
						})

						It("AddonSecret disableConfig must be false when avi-delete-config label is unset", func() {
							latestCluster := &clusterv1.Cluster{}
							err := getCluster(latestCluster, cluster.Name, cluster.Namespace)
							Expect(err).To(BeNil())
							delete(latestCluster.Labels, akoov1alpha1.AviClusterDeleteConfigLabel)
							updateObjects(latestCluster)

							ensureAKOAddOnSecretDeleteConfigMatchExpectation(client.ObjectKey{
								Name:      cluster.Name + "-load-balancer-and-ingress-service-addon",
								Namespace: cluster.Namespace,
							}, false)
						})
					})

					When("the cluster is being deleted ", func() {
						When("the cluster is ready", func() {
							BeforeEach(func() {
								latestCluster := &clusterv1.Cluster{}
								err := getCluster(latestCluster, cluster.Name, cluster.Namespace)
								Expect(err).To(BeNil())
								conditions.MarkTrue(latestCluster, akoov1alpha1.AviResourceCleanupSucceededCondition)
								err = ctx.Client.Status().Update(ctx, latestCluster)
								Expect(err).To(BeNil())
								deleteObjects(latestCluster)

								ensureRuntimeObjectMatchExpectation(client.ObjectKey{
									Name:      cluster.Name,
									Namespace: cluster.Namespace,
								}, &clusterv1.Cluster{}, false)
							})

							//Reconcile -> reconcileNormal -> r.userReconciler.ReconcileAviUserDelete
							It("should delete Avi user", func() {
								ensureRuntimeObjectMatchExpectation(client.ObjectKey{
									Name:      cluster.Name + "-" + "avi-credentials",
									Namespace: cluster.Namespace,
								}, &corev1.Secret{}, false)
							})
						})
						When("the cluster is not ready", func() {
							BeforeEach(func() {
								obj := &clusterv1.Cluster{}
								err := getCluster(obj, cluster.Name, cluster.Namespace)
								Expect(err).To(BeNil())
								conditions.MarkFalse(obj, clusterv1.ReadyCondition, clusterv1.DeletingReason, clusterv1.ConditionSeverityInfo, "")
								conditions.MarkTrue(obj, akoov1alpha1.AviResourceCleanupSucceededCondition)
								err = ctx.Client.Status().Update(ctx, obj)
								Expect(err).To(BeNil())
								deleteObjects(obj)
								ensureRuntimeObjectMatchExpectation(client.ObjectKey{
									Name:      obj.Name,
									Namespace: obj.Namespace,
								}, &clusterv1.Cluster{}, false)
							})

							//Reconcile -> reconcileNormal -> r.userReconciler.ReconcileAviUserDelete
							It("should delete Avi user", func() {
								ensureRuntimeObjectMatchExpectation(client.ObjectKey{
									Name:      cluster.Name + "-" + "avi-credentials",
									Namespace: cluster.Namespace,
								}, &corev1.Secret{}, false)
							})
						})
					})

				})

				// Reconcile -> reconcileDelete
				When("AKODeploymentConfig is being deleted", func() {
					BeforeEach(func() {
						deleteObjects(akoDeploymentConfig)
						ensureRuntimeObjectMatchExpectation(client.ObjectKey{
							Name: akoDeploymentConfig.Name,
						}, &akoov1alpha1.AKODeploymentConfig{}, false)
					})

					// Reconcile -> reconcileDelete -> phases.ReconcilePhases(normal)
					When("the cluster is not deleted ", func() {
						//Reconcile -> reconcileDelete -> reconcileClusters(normal phase) -> r.reconcileClustersDelete -> r.removeClusterLabel
						It("should remove Cluster Label", func() {
							ensureClusterAviLabelMatchExpectation(client.ObjectKey{
								Name:      cluster.Name,
								Namespace: cluster.Namespace,
							}, akoov1alpha1.AviClusterLabel, false)
						})
						//Reconcile -> reconcileDelete -> reconcileClusters(normal phase) -> r.reconcileClustersDelete -> r.removeClusterFinalizer
						It("should remove Cluster Finalizer", func() {
							ensureClusterFinalizerMatchExpectation(client.ObjectKey{
								Name:      cluster.Name,
								Namespace: cluster.Namespace,
							}, false)
						})
						//Reconcile -> reconcileDelete -> reconcileClusters(normal phase) -> r.reconcileClustersDelete -> r.clusterReconciler.ReconcileAddonSecretDelete
						It("should remove add-on secret", func() {
							ensureRuntimeObjectMatchExpectation(client.ObjectKey{
								Name:      cluster.Name + "-load-balancer-and-ingress-service-addon",
								Namespace: cluster.Namespace,
							}, &corev1.Secret{}, false)
						})
					})

					When("the cluster is being deleted ", func() {
						BeforeEach(func() {
							deleteObjects(cluster)
							ensureRuntimeObjectMatchExpectation(client.ObjectKey{
								Name:      cluster.Name,
								Namespace: cluster.Namespace,
							}, &clusterv1.Cluster{}, false)
						})

						//Reconcile -> reconcileDelete -> r.reconcileClustersDelete -> r.clusterReconciler.ReconcileAddonSecretDelete
						It("should remove Cluster Add-on Secret", func() {
							ensureRuntimeObjectMatchExpectation(client.ObjectKey{
								Name:      cluster.Name + "-load-balancer-and-ingress-service-addon",
								Namespace: cluster.Namespace,
							}, &corev1.Secret{}, false)
						})
					})
				})
			})
			When("there is default ADC install-ako-for-all", func() {
				BeforeEach(func() {
					createObjects(defaultAkoDeploymentConfig)
					ensureRuntimeObjectMatchExpectation(client.ObjectKey{
						Name: akoov1alpha1.WorkloadClusterAkoDeploymentConfig,
					}, &akoov1alpha1.AKODeploymentConfig{}, true)
				})

				AfterEach(func() {
					deleteObjects(defaultAkoDeploymentConfig)
					ensureRuntimeObjectMatchExpectation(client.ObjectKey{
						Name: akoov1alpha1.WorkloadClusterAkoDeploymentConfig,
					}, &akoov1alpha1.AKODeploymentConfig{}, false)
				})

				It("is selected by a customized ADC", func() {
					ensureClusterAviLabelMatchExpectation(client.ObjectKey{
						Name:      cluster.Name,
						Namespace: cluster.Namespace,
					}, akoov1alpha1.AviClusterSelectedLabel, true)
				})

				When("no longer selected by a customized ADC", func() {
					BeforeEach(func() {
						latestCluster := &clusterv1.Cluster{}
						Expect(getCluster(latestCluster, cluster.Name, cluster.Namespace)).To(BeNil())
						delete(latestCluster.Labels, "test")
						updateObjects(latestCluster)

						ensureClusterAviLabelMatchExpectation(client.ObjectKey{
							Name:      cluster.Name,
							Namespace: cluster.Namespace,
						}, "test", false)
					})

					It("should drop the skip-default-adc label (AviClusterSelectedLabel)", func() {
						ensureClusterAviLabelMatchExpectation(client.ObjectKey{
							Name:      cluster.Name,
							Namespace: cluster.Namespace,
						}, akoov1alpha1.AviClusterSelectedLabel, false)
					})
				})
			})
		})
	})
}
