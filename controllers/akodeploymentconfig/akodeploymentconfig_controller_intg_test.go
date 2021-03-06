// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package akodeploymentconfig_test

import (
	"github.com/avinetworks/sdk/go/models"
	"github.com/avinetworks/sdk/go/session"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	controllerruntime "gitlab.eng.vmware.com/core-build/ako-operator/pkg/controller-runtime"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/test/builder"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clustereaddonv1alpha3 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util/conditions"
	kcfg "sigs.k8s.io/cluster-api/util/kubeconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func intgTestAkoDeploymentConfigController() {
	var (
		ctx                         *builder.IntegrationTestContext
		cluster                     *clusterv1.Cluster
		akoDeploymentConfig         *akoov1alpha1.AKODeploymentConfig
		controllerCredentials       *corev1.Secret
		controllerCA                *corev1.Secret
		staticCluster               *clusterv1.Cluster
		staticAkoDeploymentConfig   *akoov1alpha1.AKODeploymentConfig
		staticControllerCredentials *corev1.Secret
		staticControllerCA          *corev1.Secret
		testLabels                  map[string]string
		err                         error
	)

	staticCluster = &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-test-8ed12g",
			Namespace: "integration-test-8ed12g",
		},
		Spec: clusterv1.ClusterSpec{},
	}
	staticAkoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ako-deployment-config",
		},
		Spec: akoov1alpha1.AKODeploymentConfigSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"test": "true",
				},
			},
			DataNetwork: akoov1alpha1.DataNetwork{
				Name: "integration-test-8ed12g",
				CIDR: "10.0.0.0/24",
				IPPools: []akoov1alpha1.IPPool{
					akoov1alpha1.IPPool{
						Start: "10.0.0.1",
						End:   "10.0.0.10",
						Type:  "V4",
					},
				},
			},
			AdminCredentialRef: &akoov1alpha1.SecretRef{
				Name:      "controller-credentials",
				Namespace: "default",
			},

			CertificateAuthorityRef: &akoov1alpha1.SecretRef{
				Name:      "controller-ca",
				Namespace: "default",
			},
			WorkloadCredentialRef: &akoov1alpha1.SecretRef{},
			ExtraConfigs: akoov1alpha1.ExtraConfigs{
				Image: akoov1alpha1.AKOImageConfig{
					Repository: "harbor-pks.vmware.com/tkgextensions/tkg-networking/ako",
					PullPolicy: "IfNotPresent",
					Version:    "1.3.2-75300bb1",
				},
			},
		},
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

	createObjects := func(objs ...runtime.Object) {
		for _, o := range objs {
			err = ctx.Client.Create(ctx.Context, o)
			Expect(err).To(BeNil())
		}
	}
	updateObjects := func(objs ...runtime.Object) {
		for _, o := range objs {
			err = ctx.Client.Update(ctx.Context, o)
			Expect(err).To(BeNil())
		}
	}
	deleteObjects := func(objs ...runtime.Object) {
		for _, o := range objs {
			// ignore error
			_ = ctx.Client.Delete(ctx.Context, o)
		}
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
	ensureRuntimeObjectMatchExpectation := func(key client.ObjectKey, obj runtime.Object, expect bool) {
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
	ensureClusterAviLabelMatchExpectation := func(key client.ObjectKey, expect bool) {
		Eventually(func() bool {
			obj := &clusterv1.Cluster{}
			err := ctx.Client.Get(ctx.Context, key, obj)
			if err != nil {
				println(err.Error())
				return false
			}
			_, ok := obj.Labels[akoov1alpha1.AviClusterLabel]
			return expect == ok
		}).Should(BeTrue())
	}

	BeforeEach(func() {
		ctx = suite.NewIntegrationTestContext()
		akoDeploymentConfig = staticAkoDeploymentConfig.DeepCopy()
		cluster = staticCluster.DeepCopy()
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
			res := &models.Network{}
			return res, nil
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
					UsableNetworkRefs: []string{"10.0.0.1"},
				},
			}
			return res, nil
		})
		ctx.AviClient.IPAMDNSProviderProfile.SetUpdateIPAMFn(func(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
			return obj, nil
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
			deleteObjects(cluster)
			ensureRuntimeObjectMatchExpectation(client.ObjectKey{
				Name:      cluster.Name,
				Namespace: cluster.Namespace,
			}, &clusterv1.Cluster{}, false)

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
				}, true)

				//Reconcile -> reconcileNormal -> reconcileClusters(normal phase) -> addClusterFinalizer
				By("should add Cluster Finalizer")
				ensureClusterFinalizerMatchExpectation(client.ObjectKey{
					Name:      cluster.Name,
					Namespace: cluster.Namespace,
				}, true)

				//Reconcile -> reconcileNormal -> reconcileClusters(normal phase) -> r.clusterReconciler.ReconcileCRS
				By("should Reconcile Cluster CRS")
				ensureRuntimeObjectMatchExpectation(client.ObjectKey{
					Name:      cluster.Name + "-ako",
					Namespace: cluster.Namespace,
				}, &clustereaddonv1alpha3.ClusterResourceSet{}, true)
			})

			When("akoDeploymentConfig and cluster are created", func() {

				// Reconcile -> reconcileNormal
				When("AKODeploymentConfig is not being deleted", func() {

				})

				// Reconcile -> reconcileDelete
				When("AKODeploymentConfig is being deleted", func() {
					BeforeEach(func() {
						// ctrlutil.AddFinalizer(akoDeploymentConfig, akoov1alpha1.AkoDeploymentConfigFinalizer)
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
							}, false)
						})
						//Reconcile -> reconcileDelete -> reconcileClusters(normal phase) -> r.reconcileClustersDelete -> r.removeClusterFinalizer
						It("should remove Cluster Finalizer", func() {
							ensureClusterFinalizerMatchExpectation(client.ObjectKey{
								Name:      cluster.Name,
								Namespace: cluster.Namespace,
							}, false)
						})
						It("should remove Cluster CRS", func() {
							ensureRuntimeObjectMatchExpectation(client.ObjectKey{
								Name:      cluster.Name + "-ako",
								Namespace: cluster.Namespace,
							}, &clustereaddonv1alpha3.ClusterResourceSet{}, false)
						})
					})
				})
			})
		})
	})
}
