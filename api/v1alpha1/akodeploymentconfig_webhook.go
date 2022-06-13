// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/vmware/alb-sdk/go/clients"
	"github.com/vmware/alb-sdk/go/session"
)

// log is for logging in this package.
var akoDeploymentConfigLog = logf.Log.WithName("akodeploymentconfig-resource")
var kclient client.Client

func (r *AKODeploymentConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kclient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-networking-tkg-tanzu-vmware-com-v1alpha1-akodeploymentconfig,mutating=true,failurePolicy=fail,groups=networking.tkg.tanzu.vmware.com,resources=akodeploymentconfigs,verbs=create;update,versions=v1alpha1,name=vakodeploymentconfig.kb.io,sideEffects=None,admissionReviewVersions=v1;v1alpha1

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-networking-tkg-tanzu-vmware-com-v1alpha1-akodeploymentconfig,mutating=false,failurePolicy=fail,groups=networking.tkg.tanzu.vmware.com,resources=akodeploymentconfigs,versions=v1alpha1,name=vakodeploymentconfig.kb.io, sideEffects=None, admissionReviewVersions=v1;v1alpha1

var _ webhook.Validator = &AKODeploymentConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AKODeploymentConfig) ValidateCreate() error {
	akoDeploymentConfigLog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList
	selector, err := metav1.LabelSelectorAsSelector(&r.Spec.ClusterSelector)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ClusterSelector"),
			r.Spec.ClusterSelector,
			err.Error()))
	}
	// non default ADC (a.k.a name is not install-ako-for-all), should have non-empty cluster selector
	if r.ObjectMeta.Name != WorkloadClusterAkoDeploymentConfig && selector.Empty() {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ClusterSelector"),
			r.Spec.ClusterSelector,
			"field should not be empty for non-default ADC"))
	}

	allErrs = append(allErrs, r.validateAVI()...)
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(GroupVersion.WithKind("AKODeploymentConfig").GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AKODeploymentConfig) ValidateUpdate(old runtime.Object) error {
	akoDeploymentConfigLog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList
	oldADC, ok := old.(*AKODeploymentConfig)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a AKODeploymentConfig but got a %T", old))
	}

	if oldADC != nil {
		if (oldADC.Spec.ControlPlaneNetwork.CIDR != r.Spec.ControlPlaneNetwork.CIDR) ||
			(oldADC.Spec.ControlPlaneNetwork.Name != r.Spec.ControlPlaneNetwork.Name) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ControlPlaneNetwork"),
				r.Spec.ControlPlaneNetwork,
				"field should not be changed"))
		}

		if oldADC.Spec.ClusterSelector.String() != r.Spec.ClusterSelector.String() {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ClusterSelector"),
				r.Spec.ClusterSelector,
				"field should not be changed"))
		}
	}

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(GroupVersion.WithKind("AKODeploymentConfig").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AKODeploymentConfig) ValidateDelete() error {
	akoDeploymentConfigLog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *AKODeploymentConfig) validateAVI() field.ErrorList {
	var allErrs field.ErrorList
	// following fields are already required fileds in CRD, no need to validate here
	// - adminCredentialRef
	// - certificateAuthorityRef
	// - cloudName
	// - controller
	// - dataNetwork
	// - serviceEngineGroup
	adminCredential := &corev1.Secret{}
	if err := kclient.Get(context.Background(), client.ObjectKey{
		Name:      r.Spec.AdminCredentialRef.Name,
		Namespace: r.Spec.AdminCredentialRef.Namespace,
	}, adminCredential); err != nil {
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("spec", "adminCredentialRef"),
					r.Spec.AdminCredentialRef,
					"can't find admin credential secret"))
		} else {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("spec", "adminCredentialRef"),
					r.Spec.AdminCredentialRef,
					"failed to find admin credential secret:"+err.Error()))
		}
	}

	aviControllerCA := &corev1.Secret{}
	if err := kclient.Get(context.Background(), client.ObjectKey{
		Name:      r.Spec.CertificateAuthorityRef.Name,
		Namespace: r.Spec.CertificateAuthorityRef.Name,
	}, aviControllerCA); err != nil {
		if apierrors.IsNotFound(err) {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("spec", "certificateAuthorityRef"),
					r.Spec.CertificateAuthorityRef,
					"can't find certificate secret"))
		} else {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("spec", "certificateAuthorityRef"),
					r.Spec.CertificateAuthorityRef,
					"failed to find certificate secret:"+err.Error()))
		}
	}

	if len(allErrs) != 0 {
		return allErrs
	}

	username := string(adminCredential.Data["username"][:])
	password := string(adminCredential.Data["password"][:])
	certificate := string(aviControllerCA.Data["certificateAuthorityData"][:])

	var transport *http.Transport
	if certificate != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(certificate))
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
	}
	options := []func(*session.AviSession) error{
		session.SetPassword(password),
		session.SetTransport(transport),
		session.SetVersion(r.Spec.ControllerVersion),
	}
	aviClient, err := clients.NewAviClient(r.Spec.Controller, username, options...)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "controller"),
			r.Spec.Controller,
			"failed to init avi client:"+err.Error()))
		return allErrs
	}

	allErrs = append(allErrs, r.validateAviCloud(*aviClient))
	allErrs = append(allErrs, r.validateAviServiceEngineGroup(*aviClient))
	allErrs = append(allErrs, r.validateAviControlPlaneNetworks(*aviClient)...)
	allErrs = append(allErrs, r.validateAviDataNetworks(*aviClient)...)
	if len(allErrs) != 0 {
		return allErrs
	}
	return nil
}

func (r *AKODeploymentConfig) validateAviCloud(aviClient clients.AviClient) *field.Error {
	if cloud, err := aviClient.Cloud.GetByName(r.Spec.CloudName); err != nil {
		return field.Invalid(field.NewPath("spec", "cloudName"), r.Spec.CloudName,
			"failed to get cloud from avi controller:"+err.Error())
	} else if cloud.IPAMProviderRef == nil {
		return field.Invalid(field.NewPath("spec", "cloudName"), r.Spec.CloudName,
			"this cloud doesn't have any ipam profile configured")
	}
	return nil
}

func (r *AKODeploymentConfig) validateAviServiceEngineGroup(aviClient clients.AviClient) *field.Error {
	if _, err := aviClient.ServiceEngineGroup.GetByName(r.Spec.ServiceEngineGroup); err != nil {
		return field.Invalid(field.NewPath("spec", "serviceEngineGroup"), r.Spec.ServiceEngineGroup,
			"failed to get service engine group from avi controller:"+err.Error())
	}
	return nil
}

func (r *AKODeploymentConfig) validateAviControlPlaneNetworks(aviClient clients.AviClient) field.ErrorList {
	var allErrs field.ErrorList
	// check data network name
	if _, err := aviClient.Network.GetByName(r.Spec.ControlPlaneNetwork.Name); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "controlPlaneNetwork", "name"),
			r.Spec.ControlPlaneNetwork.Name,
			"failed to get data plane network "+r.Spec.ControlPlaneNetwork.Name+" from avi controller:"+err.Error()))
	}
	// check network cidr validate or not
	_, _, err := net.ParseCIDR(r.Spec.ControlPlaneNetwork.CIDR)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "controlPlaneNetwork", "cidr"),
			r.Spec.ControlPlaneNetwork.CIDR,
			"data plane network cidr "+r.Spec.ControlPlaneNetwork.CIDR+" is not valid:"+err.Error()))
	}
	return allErrs
}

func (r *AKODeploymentConfig) validateAviDataNetworks(aviClient clients.AviClient) field.ErrorList {
	var allErrs field.ErrorList
	// check data network name
	if _, err := aviClient.Network.GetByName(r.Spec.DataNetwork.Name); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "name"),
			r.Spec.DataNetwork.Name,
			"failed to get data plane network "+r.Spec.DataNetwork.Name+" from avi controller:"+err.Error()))
	}
	// check network cidr validate or not
	_, cidr, err := net.ParseCIDR(r.Spec.DataNetwork.CIDR)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "cidr"),
			r.Spec.DataNetwork.CIDR,
			"data plane network cidr "+r.Spec.DataNetwork.CIDR+" is not valid:"+err.Error()))
	}
	// check data network ip pools
	for _, ipPool := range r.Spec.DataNetwork.IPPools {
		ipStart := net.ParseIP(ipPool.Start)
		ipEnd := net.ParseIP(ipPool.End)
		if ipStart == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "ipPools"),
				r.Spec.DataNetwork.IPPools,
				"ip pool address"+ipPool.Start+" is not valid"))
		}
		if ipEnd == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "ipPools"),
				r.Spec.DataNetwork.IPPools,
				"ip pool address"+ipPool.End+" is not valid"))
		}
		if !cidr.Contains(ipStart) || !cidr.Contains(ipEnd) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "ipPools"),
				r.Spec.DataNetwork.IPPools,
				"Range ["+ipPool.Start+","+ipPool.End+"] is not in cidr"))
		}
		if bytes.Compare(ipStart, ipEnd) > 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "ipPools"),
				r.Spec.DataNetwork.IPPools,
				ipPool.Start+"is greater than"+ipPool.End))
		}
		// TODO:(xudongl) will wait for AKO support v6 address to uncomment this
		// if ippool.Type != addrType {
		// 	return field.Invalid(field.NewPath("spec", "dataNetwork", "ipPools"),
		// 		r.Spec.DataNetwork.IPPools,
		// 		"data plane network ip pools type is not aligned with cidr")
		// }
	}
	return allErrs
}
