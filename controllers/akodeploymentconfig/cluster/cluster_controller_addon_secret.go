// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ClusterReconciler) ReconcileAddonSecret(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Starts reconciling add on secret")
	res := ctrl.Result{}
	newSecret, err := r.createAKOAddonSecret(cluster, obj)
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
			return res, r.Create(ctx, newSecret)
		}
		log.Error(err, "Failed to get AKO Deployment Secret, requeue")
		return res, err
	}
	secret = newSecret.DeepCopy()
	return res, r.Update(ctx, secret)
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
	return res, r.Delete(ctx, secret)
}

func (r *ClusterReconciler) akoAddonSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-addon"
}

func (r *ClusterReconciler) createAKOAddonSecret(cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig) (*corev1.Secret, error) {
	secretStringData, err := AkoAddonSecretDataYaml(cluster, obj)
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

func AkoAddonSecretDataYaml(cluster *clusterv1.Cluster, obj *akoov1alpha1.AKODeploymentConfig) (string, error) {
	secret, err := ako.NewValues(obj, cluster.Namespace+"-"+cluster.Name)
	if err != nil {
		return "", err
	}
	return secret.YttYaml()
}
