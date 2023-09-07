// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako"
	akoo "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	clusterapipatchutil "sigs.k8s.io/cluster-api/util/patch"
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
	isVIPProvider, err := akoo.IsControlPlaneVIPProvider(cluster)
	if err != nil {
		log.Error(err, "can't unmarshal cluster variables")
		return res, err
	}
	if isVIPProvider && cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		svc := &corev1.Service{}
		if err = r.Get(ctx, client.ObjectKey{
			Name:      cluster.Namespace + "-" + cluster.Name + "-" + akoov1alpha1.HAServiceName,
			Namespace: akoov1alpha1.TKGSystemNamespace,
		}, svc); err != nil {
			log.Info("Failed to get cluster control plane load balancer type of service, requeue")
			return res, err
		}
	}

	//Stop reconciling if AKO ip family doesn't match cluster node ip family
	if err = validateADCAndClusterIpFamily(cluster, obj, isVIPProvider, log); err != nil {
		log.Error(err, "Error: ip family in AKODeploymentConfig doesn't match cluster node ip family")
		conditions.MarkTrue(cluster, akoov1alpha1.ClusterIpFamilyValidationFailedCondition)
		return res, err
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

	// patch cluster bootstrap when it is classy cluster and not in bootstrap cluster
	if akoo.IsClusterClassBasedCluster(cluster) && !akoo.IsBootStrapCluster() {
		log.Info("patching clusterbootstrap with ako packageRef")
		if err := r.patchAkoPackageRefToClusterBootstrap(ctx, log, cluster); err != nil {
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

	// TODO: skip this step since ClusterBootstrap webhook doesn't aloow this yet
	// Add back once this is supported
	// if akoo.IsClusterClassBasedCluster(cluster) {

	// 	// remove cluster bootstrap correspondingly
	// 	if err := r.removeAkoPackageRefFromClusterBootstrap(ctx, cluster); err != nil {
	// 		log.Error(err, "Failed to remove ako package ref from cluster bootstrap, requeue")
	// 		return res, err
	// 	}
	// }
	return res, nil
}

func (r *ClusterReconciler) aviUserSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-avi-credentials"
}

func (r *ClusterReconciler) akoAddonSecretName(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-addon"
}

func (r *ClusterReconciler) akoAddonSecretNameForClusterClass(cluster *clusterv1.Cluster) string {
	return cluster.Name + "-load-balancer-and-ingress-service-data-values"
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

	if akoo.IsClusterClassBasedCluster(cluster) {
		secret.Type = akoov1alpha1.TKGClusterClassAddOnSecretType
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
	return secret.YttYaml(cluster)
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
func (r *ClusterReconciler) patchAkoPackageRefToClusterBootstrap(ctx context.Context, log logr.Logger, cluster *clusterv1.Cluster) error {
	bootstrap, err := r.getClusterBootstrap(ctx, cluster)
	if err != nil {
		return err
	}

	// Create a patch helper for clusterbootstrap
	patchHelper, err := clusterapipatchutil.NewHelper(bootstrap, r.Client)
	if err != nil {
		return err
	}

	akoPackageRefName, err := r.GetAKOPackageRefName(ctx, log, bootstrap)
	if err != nil {
		return err
	}
	expectedAKOClusterBootstrapPackage := &runv1alpha3.ClusterBootstrapPackage{
		RefName: akoPackageRefName,
		ValuesFrom: &runv1alpha3.ValuesFrom{
			SecretRef: r.akoAddonSecretName(cluster),
		},
	}

	// if there is already existing ako packageRef, check if it's up to date, if not, patch it
	index, akoPackageInClusterBootstrap := getAKOPackageRefFromClusterBootstrap(log, bootstrap)
	if akoPackageInClusterBootstrap == nil || index == -1 {
		// ako package ref not presented
		// append ako package ref to cluster bootstrap package install
		log.Info("ako package ref not found in ClusterBootstrap, patching")
		bootstrap.Spec.AdditionalPackages = append(bootstrap.Spec.AdditionalPackages, expectedAKOClusterBootstrapPackage)
	} else {
		// check if it's up to date, if not, patch it
		if !reflect.DeepEqual(akoPackageInClusterBootstrap, expectedAKOClusterBootstrapPackage) {
			log.Info(fmt.Sprintf("ako package ref is not up to date. update from %v to %v", akoPackageInClusterBootstrap, expectedAKOClusterBootstrapPackage))
			bootstrap.Spec.AdditionalPackages[index] = expectedAKOClusterBootstrapPackage
		} else {
			log.Info("ako package ref up to date. skip")
		}
	}

	// Add skip deleting ako packageinstall annotation to clusterboostrap if it doesn't exist
	if _, exist := bootstrap.Annotations[akoov1alpha1.TKGSkipDeletePkgiAnnotationKey]; !exist {
		if bootstrap.Annotations == nil {
			bootstrap.Annotations = make(map[string]string)
		}
		bootstrap.Annotations[akoov1alpha1.TKGSkipDeletePkgiAnnotationKey] += "," + akoov1alpha1.AkoPackageInstallName
	} else {
		if !strings.Contains(bootstrap.Annotations[akoov1alpha1.TKGSkipDeletePkgiAnnotationKey], akoov1alpha1.AkoPackageInstallName) {
			bootstrap.Annotations[akoov1alpha1.TKGSkipDeletePkgiAnnotationKey] += "," + akoov1alpha1.AkoPackageInstallName
		}
	}

	return patchHelper.Patch(ctx, bootstrap.DeepCopy())
}

// This is not supported at the moment. But good thing is we don't have this scenario at the moment
// removeAkoPackageRefFromClusterBootstrap removes the ako package ref from cluster's clusterbootstrap object
func (r *ClusterReconciler) removeAkoPackageRefFromClusterBootstrap(ctx context.Context, cluster *clusterv1.Cluster) error {
	bootstrap, err := r.getClusterBootstrap(ctx, cluster)
	if err != nil {
		return err
	}

	// Create a patch helper for clusterbootstrap
	patchHelper, err := clusterapipatchutil.NewHelper(bootstrap, r.Client)
	if err != nil {
		return err
	}

	for i, clusterBootstrapPackage := range bootstrap.Spec.AdditionalPackages {
		// remove ako package from cluster bootstrap additional packages
		if strings.HasPrefix(clusterBootstrapPackage.RefName, akoov1alpha1.AkoClusterBootstrapRefNamePrefix) {
			bootstrap.Spec.AdditionalPackages[i] = bootstrap.Spec.AdditionalPackages[len(bootstrap.Spec.AdditionalPackages)-1]
			bootstrap.Spec.AdditionalPackages = bootstrap.Spec.AdditionalPackages[:len(bootstrap.Spec.AdditionalPackages)-1]
		}
	}

	return patchHelper.Patch(ctx, bootstrap.DeepCopy())
}

func (r *ClusterReconciler) GetAKOPackageRefName(ctx context.Context, log logr.Logger, cb *runv1alpha3.ClusterBootstrap) (string, error) {
	if cb.Status.ResolvedTKR == "" {
		return "", errors.New("ClusterBootstrap.Status.ResolvedTKR is empty")
	}
	tkrName := cb.Status.ResolvedTKR
	tkr := &runv1alpha3.TanzuKubernetesRelease{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: tkrName}, tkr); err != nil {
		log.Error(err, fmt.Sprintf("unable to get the TanzuKubernetesRelease %s", tkrName))
		return "", err
	}

	akoPackageRefFullName, err := r.GetAKOPackageRefNameFromTKR(log, tkr)
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to complete ako packageRef name from tkr %s", tkrName))
		return "", err
	}

	return akoPackageRefFullName, nil
}

func (r *ClusterReconciler) GetAKOPackageRefNameFromTKR(log logr.Logger, tkr *runv1alpha3.TanzuKubernetesRelease) (string, error) {
	for _, tkrBootstrapPackage := range tkr.Spec.BootstrapPackages {
		if strings.HasPrefix(tkrBootstrapPackage.Name, akoov1alpha1.AkoClusterBootstrapRefNamePrefix) {
			log.Info(fmt.Sprintf("found ako package ref %s in tkr", tkrBootstrapPackage.Name))
			return tkrBootstrapPackage.Name, nil
		}
	}
	return "", fmt.Errorf("no bootstrapPackage name matches the prefix %s within the BootstrapPackages [%v] of TanzuKubernetesRelease %s", akoov1alpha1.AkoClusterBootstrapRefNamePrefix, tkr.Spec.BootstrapPackages, tkr.Name)
}

func getAKOPackageRefFromClusterBootstrap(log logr.Logger, cb *runv1alpha3.ClusterBootstrap) (int, *runv1alpha3.ClusterBootstrapPackage) {
	for index, additionalPackage := range cb.Spec.AdditionalPackages {
		if strings.HasPrefix(additionalPackage.RefName, akoov1alpha1.AkoClusterBootstrapRefNamePrefix) {
			log.Info(fmt.Sprintf("found ako package ref %s in clusterbootstrap", additionalPackage.RefName))
			return index, additionalPackage
		}
	}
	return -1, nil
}

func validateADCAndClusterIpFamily(cluster *clusterv1.Cluster, adc *akoov1alpha1.AKODeploymentConfig, isVIPProvider bool, log logr.Logger) error {
	adcIpFamily := "V4"
	if adc.Spec.ExtraConfigs.IpFamily != "" {
		adcIpFamily = adc.Spec.ExtraConfigs.IpFamily
	}
	nodeIpFamily, err := utils.GetClusterIPFamily(cluster)
	if err != nil {
		log.Error(err, "can't get cluster ip family")
		return err
	}
	if (adcIpFamily == "V4" && nodeIpFamily == "V6") || (adcIpFamily == "V6" && nodeIpFamily == "V4") {
		errInfo := "We are not allowed to create single stack " + nodeIpFamily + " cluster when configure AKO as " + adcIpFamily + " ip family"
		return errors.New(errInfo)
	}
	if isVIPProvider {
		if adcIpFamily == "V4" && nodeIpFamily == "V6,V4" {
			return errors.New("When enabling avi as control plane HA, we are not allowed to create ipv6 primary dual-stack cluster if AKO is configured V4 ip family")
		} else if adcIpFamily == "V6" && nodeIpFamily == "V4,V6" {
			return errors.New("When enabling avi as control plane HA, we are not allowed to create ipv4 primary dual-stack cluster if AKO is configured V6 ip family")
		}

	}
	return nil
}
