// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	"github.com/go-logr/logr"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako"
	akoo "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runv1alpha3 "github.com/vmware-tanzu/tanzu-framework/apis/run/v1alpha3"
)

func (r *ClusterReconciler) ReconcileAddonSecret(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Starts reconciling add on secret")
	res := ctrl.Result{}
	aviSecret, err := r.getClusterAviUserSecret(cluster, ctx)
	if err != nil {
		log.Info("Failed to get cluster avi user secret, requeue")
		return res, err
	}

	// when avi is ha provider and deploy ako in management cluster, need to wait for
	// control plane load balancer type of service creating
	if akoo.IsControlPlaneVIPProvider(cluster) && cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		svc := &corev1.Service{}
		if err = r.Get(ctx, client.ObjectKey{
			Name:      cluster.Namespace + "-" + cluster.Name + "-" + akoov1alpha1.HAServiceName,
			Namespace: akoov1alpha1.TKGSystemNamespace,
		}, svc); err != nil {
			log.Info("Failed to get cluster control plane load balancer type of service, requeue")
			return res, err
		}
	}

	newAddonSecret, err := r.createAKOAddonSecret(cluster, obj, aviSecret)
	if err != nil {
		log.Info("Failed to convert AKO Deployment Config to add-on secret, requeue the request")
		return res, err
	}
	secret := &corev1.Secret{}
	if err = r.Get(ctx, client.ObjectKey{
		Name:      r.akoAddonSecretName(cluster),
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AKO add on secret doesn't exist, start creating it")
			return res, r.Create(ctx, newAddonSecret)
		}
		log.Error(err, "Failed to get AKO Deployment Secret, requeue")
		return res, err
	}
	secret = newAddonSecret.DeepCopy()
	if err := r.Update(ctx, secret); err != nil {
		log.Error(err, "Failed to update ako add on secret, requeue")
		return res, err
	}

	if akoo.IsClusterClassBasedCluster(cluster) {
		// patch cluster bootstrap here
		if err := r.patchAkoPackageRefToClusterBootstrap(ctx, cluster); err != nil {
			log.Error(err, "Failed to patch ako package ref to cluster bootstrap, requeue")
			return res, err
		}
	}
	return res, nil
}

func (r *ClusterReconciler) ReconcileAddonSecretDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	_ *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Starts reconciling add on secret deletion")
	res := ctrl.Result{}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      r.akoAddonSecretName(cluster),
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AKO add on secret already deleted")
			return res, nil
		}
		log.Error(err, "Failed to get AKO Deployment Secret, requeue")
		return res, err
	}
	if err := r.Delete(ctx, secret); err != nil {
		log.Error(err, "Failed to delete ako add on secret, requeue")
		return res, err
	}

	if akoo.IsClusterClassBasedCluster(cluster) {
		// remove cluster bootstrap correspondingly
		if err := r.removeAkoPackageRefFromClusterBootstrap(ctx, cluster); err != nil {
			log.Error(err, "Failed to remove ako package ref from cluster bootstrap, requeue")
			return res, err
		}
	}
	return res, nil
}

func (r *ClusterReconciler) aviUserSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-avi-credentials"
}

func (r *ClusterReconciler) akoAddonSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-addon"
}

func (r *ClusterReconciler) createAKOAddonSecret(cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig, aviUsersecret *corev1.Secret) (*corev1.Secret, error) {
	secretStringData, err := AkoAddonSecretDataYaml(cluster, obj, aviUsersecret)
	if err != nil {
		return nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.akoAddonSecretName(cluster),
			Namespace: cluster.Namespace,
			Annotations: map[string]string{
				akoov1alpha1.TKGAddonAnnotationKey: "networking/load-balancer-and-ingress-service",
			},
			Labels: map[string]string{
				akoov1alpha1.TKGAddOnLabelAddonNameKey:   "load-balancer-and-ingress-service",
				akoov1alpha1.TKGAddOnLabelClusterNameKey: cluster.Name,
				akoov1alpha1.TKGAddOnLabelClusterctlKey:  "",
			},
		},
		Type: akoov1alpha1.TKGAddOnSecretType,
		StringData: map[string]string{
			akoov1alpha1.TKGAddOnSecretDataKey: secretStringData,
		},
	}
	return secret, nil
}

func AkoAddonSecretDataYaml(cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig, aviUsersecret *corev1.Secret) (string, error) {
	secret, err := ako.NewValues(obj, cluster.Namespace+"-"+cluster.Name)
	if err != nil {
		return "", err
	}

	//Pass cluster role information to ako
	//Avoid setting DeleteConfig for management cluster
	if cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		secret.LoadBalancerAndIngressService.Config.TkgClusterRole = "management"
	} else {
		secret.LoadBalancerAndIngressService.Config.TkgClusterRole = "workload"
		if deleteConfig, exists := cluster.Labels[akoov1alpha1.AviClusterDeleteConfigLabel]; exists {
			if deleteConfig == "true" {
				secret.LoadBalancerAndIngressService.Config.AKOSettings.DeleteConfig = "true"
			}
		}
	}

	secret.LoadBalancerAndIngressService.Config.Avicredentials.Username = string(aviUsersecret.Data["username"][:])
	secret.LoadBalancerAndIngressService.Config.Avicredentials.Password = string(aviUsersecret.Data["password"][:])
	secret.LoadBalancerAndIngressService.Config.Avicredentials.CertificateAuthorityData = string(aviUsersecret.Data[akoov1alpha1.AviCertificateKey][:])
	return secret.YttYaml()
}

func (r *ClusterReconciler) getClusterAviUserSecret(cluster *clusterv1.Cluster, ctx context.Context) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      r.aviUserSecretName(cluster),
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		return secret, err
	}
	return secret, nil
}

// @TODOs:(xudongl): add test cases to cover following functions
// getClusterBootstrap gets cluster's clusterbootstrap object
func (r *ClusterReconciler) getClusterBootstrap(ctx context.Context, cluster *clusterv1.Cluster) (*runv1alpha3.ClusterBootstrap, error) {
	bootstrap := &runv1alpha3.ClusterBootstrap{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}, bootstrap); err != nil {
		return nil, err
	}
	return bootstrap, nil
}

// patchAkoPackageRefToClusterBootstrap adds ako package ref to the cluster's clusterbootstrap object
func (r *ClusterReconciler) patchAkoPackageRefToClusterBootstrap(ctx context.Context, cluster *clusterv1.Cluster) error {
	bootstrap, err := r.getClusterBootstrap(ctx, cluster)
	if err != nil {
		return err
	}

	akoClusterBootstrapPackage := &runv1alpha3.ClusterBootstrapPackage{
		RefName: akoov1alpha1.AkoClusterBootstrapRefName,
		ValuesFrom: &runv1alpha3.ValuesFrom{
			SecretRef: r.akoAddonSecretName(cluster),
		},
	}
	// append ako package ref to cluster bootstrap package install
	bootstrap.Spec.AdditionalPackages = append(bootstrap.Spec.AdditionalPackages, akoClusterBootstrapPackage)
	return r.Update(ctx, bootstrap)
}

// removeAkoPackageRefFromClusterBootstrap removes the ako package ref from cluster's clusterbootstrap object
func (r *ClusterReconciler) removeAkoPackageRefFromClusterBootstrap(ctx context.Context, cluster *clusterv1.Cluster) error {
	bootstrap, err := r.getClusterBootstrap(ctx, cluster)
	if err != nil {
		return err
	}

	for i, clusterBootstrapPackage := range bootstrap.Spec.AdditionalPackages {
		// remove ako package from cluster bootstrap additional packages
		if clusterBootstrapPackage.RefName == akoov1alpha1.AkoClusterBootstrapRefName {
			bootstrap.Spec.AdditionalPackages[i] = bootstrap.Spec.AdditionalPackages[len(bootstrap.Spec.AdditionalPackages)-1]
			bootstrap.Spec.AdditionalPackages = bootstrap.Spec.AdditionalPackages[:len(bootstrap.Spec.AdditionalPackages)-1]
		}
	}

	return r.Update(ctx, bootstrap)
}
