// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
)

// log is for logging in this package.
var akoDeploymentConfigLog = logf.Log.WithName("akodeploymentconfig-resource")
var kclient client.Client
var aviClient aviclient.Client
var runTest bool

const controllerVersionRegex = `^\d+(\.\d+)*$`

func (r *AKODeploymentConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	kclient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-networking-tkg-tanzu-vmware-com-v1alpha1-akodeploymentconfig,mutating=true,failurePolicy=fail,groups=networking.tkg.tanzu.vmware.com,resources=akodeploymentconfigs,versions=v1alpha1,name=vakodeploymentconfig.kb.io,sideEffects=None,admissionReviewVersions=v1;v1alpha1
//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-networking-tkg-tanzu-vmware-com-v1alpha1-akodeploymentconfig,mutating=false,failurePolicy=fail,groups=networking.tkg.tanzu.vmware.com,resources=akodeploymentconfigs,versions=v1alpha1,name=vakodeploymentconfig.kb.io, sideEffects=None, admissionReviewVersions=v1;v1alpha1

var _ webhook.Validator = &AKODeploymentConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AKODeploymentConfig) ValidateCreate() (admission.Warnings, error) {
	akoDeploymentConfigLog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList
	allErrs = append(allErrs, r.validateClusterSelector(nil)...)
	allErrs = append(allErrs, r.validateAVI(nil)...)
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("AKODeploymentConfig").GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AKODeploymentConfig) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	akoDeploymentConfigLog.Info("validate update", "name", r.Name)
	oldADC, ok := old.(*AKODeploymentConfig)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a AKODeploymentConfig but got a %T", old))
	}
	var allErrs field.ErrorList
	if oldADC != nil {
		allErrs = append(allErrs, r.validateClusterSelector(oldADC)...)
		allErrs = append(allErrs, r.validateAVI(oldADC)...)
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("AKODeploymentConfig").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AKODeploymentConfig) ValidateDelete() (admission.Warnings, error) {
	akoDeploymentConfigLog.Info("validate delete", "name", r.Name)
	// should not delete the akodeploymentconfig selects management cluster
	if r.Name == ManagementClusterAkoDeploymentConfig {
		return nil, field.Invalid(field.NewPath("spec", "ClusterSelector"),
			r.Spec.ClusterSelector,
			"can't delete akodeploymentconfig object for management cluster")
	}
	return nil, nil
}

// validateClusterSelector checks AKODeploymentConfig object's cluster selector field input is valid or not
// when old is nil, it is used for AKODeploymentConfig object create, otherwise it is used for AKODeploymentConfig
// object update
func (r *AKODeploymentConfig) validateClusterSelector(old *AKODeploymentConfig) field.ErrorList {
	var allErrs field.ErrorList
	// when update AKODeploymentConfig object, cluster selector should be immutable
	if old != nil {
		if old.Spec.ClusterSelector.String() != r.Spec.ClusterSelector.String() {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ClusterSelector"),
				r.Spec.ClusterSelector,
				"field should not be changed"))
			return allErrs
		}
	}
	// convert cluster selector to label selector
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
	return allErrs
}

// validateAVI checks all NSX Advanced Load Balancer related fields are valid or not
// when old is nil, it is used for AKODeploymentConfig object create, otherwise it is used for AKODeploymentConfig
// object update. Following fields are already required fileds in CRD, so no need to check if those fields are empty.
// - adminCredentialRef
// - certificateAuthorityRef
// - cloudName
// - controller
// - dataNetwork
// - serviceEngineGroup
func (r *AKODeploymentConfig) validateAVI(old *AKODeploymentConfig) field.ErrorList {
	var allErrs field.ErrorList

	// check avi related secret
	adminCredential := &corev1.Secret{}
	if err := r.validateAviSecret(adminCredential, r.Spec.AdminCredentialRef); err != nil {
		allErrs = append(allErrs, err)
	}

	aviControllerCA := &corev1.Secret{}
	if err := r.validateAviSecret(aviControllerCA, r.Spec.CertificateAuthorityRef); err != nil {
		allErrs = append(allErrs, err)
	}

	if len(allErrs) != 0 {
		return allErrs
	}

	// check avi controller version format
	_, err := r.validateAviControllerVersion()
	if err != nil {
		allErrs = append(allErrs, err)
	}

	if !runTest {
		username := string(adminCredential.Data["username"][:])
		password := string(adminCredential.Data["password"][:])
		certificate := string(aviControllerCA.Data["certificateAuthorityData"][:])

		client, fieldErr := r.validateAviAccount(username, password, certificate, "")
		if fieldErr != nil {
			allErrs = append(allErrs, fieldErr)
			return allErrs
		}
		// get actual avi controller version
		version, err := client.GetControllerVersion()
		if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ControllerVersion"), r.Spec.Controller, "failed to get avi controller version:"+err.Error()))
			return allErrs
		}

		// update controller version
		r.Spec.ControllerVersion = version
		// reinit client with the real controller version
		client, fieldErr = r.validateAviAccount(username, password, certificate, version)
		if fieldErr != nil {
			allErrs = append(allErrs, fieldErr)
			return allErrs
		}
		aviClient = client
	}

	if old == nil {
		// when old is nil, it is creating a new AKODeploymentConfig object, check following fields
		if err := r.validateAviCloud(); err != nil {
			allErrs = append(allErrs, err)
		}
		if err := r.validateAviServiceEngineGroup(); err != nil {
			allErrs = append(allErrs, err)
		}
		if err := r.validateAviControlPlaneNetworks(); err != nil {
			allErrs = append(allErrs, err...)
		}
		if err := r.validateAviDataNetworks(); err != nil {
			allErrs = append(allErrs, err...)
		}
	} else {
		// when old is not nil, it is updating an existing AKODeploymentConfig object,
		// only check changed fields
		if old.Spec.CloudName != r.Spec.CloudName {
			if err := r.validateAviCloud(); err != nil {
				allErrs = append(allErrs, err)
			}
		}
		if old.Spec.ServiceEngineGroup != r.Spec.ServiceEngineGroup {
			if err := r.validateAviServiceEngineGroup(); err != nil {
				allErrs = append(allErrs, err)
			}
		}
		// control plane network should be immutable since cluster control plane endpoint
		// can't be updated
		if (old.Spec.ControlPlaneNetwork.Name != r.Spec.ControlPlaneNetwork.Name) ||
			(old.Spec.ControlPlaneNetwork.CIDR != r.Spec.ControlPlaneNetwork.CIDR) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ControlPlaneNetwork"),
				r.Spec.ControlPlaneNetwork,
				"field should not be changed"))
		}
		if (old.Spec.DataNetwork.Name != r.Spec.DataNetwork.Name) ||
			(old.Spec.DataNetwork.CIDR != r.Spec.DataNetwork.CIDR) {
			if err := r.validateAviDataNetworks(); err != nil {
				allErrs = append(allErrs, err...)
			}
		}
	}
	return allErrs
}

// validateAviSecret checks NSX Advanced Load Balancer related credentials or certificate secret is valid or not
func (r *AKODeploymentConfig) validateAviSecret(secret *corev1.Secret, secretRef SecretReference) *field.Error {
	if err := kclient.Get(context.Background(), client.ObjectKey{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return field.Invalid(field.NewPath("spec", secretRef.Namespace+"/"+secretRef.Name),
				secretRef.Name,
				"can't find secret")
		} else {
			return field.Invalid(field.NewPath("spec", secretRef.Namespace+"/"+secretRef.Name),
				secretRef.Name,
				"failed to find secret:"+err.Error())
		}
	}
	return nil
}

// validateAviControllerVersion checks NSX Advanced Load Balancer controller version valid or not
func (r *AKODeploymentConfig) validateAviControllerVersion() (string, *field.Error) {
	controllerVersion := ""
	if r.Spec.ControllerVersion != "" {
		controllerVersion := r.Spec.ControllerVersion
		var re = regexp.MustCompile(controllerVersionRegex)
		if !re.MatchString(controllerVersion) {
			return controllerVersion, field.Invalid(field.NewPath("spec", "ControllerVersion"),
				r.Spec.ControllerVersion,
				"invalid controller version format, example valid controller version: 21.1.4")
		}
	}
	return controllerVersion, nil
}

// validateAviAccount checks if using inputs can connect to avi controller or not
func (r *AKODeploymentConfig) validateAviAccount(username, password, certificate, version string) (aviclient.Client, *field.Error) {
	aviClient, err := aviclient.NewAviClient(&aviclient.AviClientConfig{
		ServerIP: r.Spec.Controller,
		Username: username,
		Password: password,
		CA:       certificate,
	}, version)
	if err != nil {
		return nil, field.Invalid(field.NewPath("spec", "Controller"), r.Spec.Controller, "failed to init avi client for controller:"+err.Error())
	}
	return aviClient, nil
}

// validateAviCloud checks input Cloud Name field valid or not
func (r *AKODeploymentConfig) validateAviCloud() *field.Error {
	if cloud, err := aviClient.CloudGetByName(r.Spec.CloudName); err != nil {
		return field.Invalid(field.NewPath("spec", "cloudName"), r.Spec.CloudName,
			"failed to get cloud from avi controller:"+err.Error())
	} else if cloud.IPAMProviderRef == nil {
		return field.Invalid(field.NewPath("spec", "cloudName"), r.Spec.CloudName,
			"this cloud doesn't have any ipam profile configured")
	}
	return nil
}

// validateAviServiceEngineGroup checks input Servcie Engine Group valid or not
func (r *AKODeploymentConfig) validateAviServiceEngineGroup() *field.Error {
	if _, err := aviClient.ServiceEngineGroupGetByName(r.Spec.ServiceEngineGroup, r.Spec.CloudName); err != nil {
		return field.Invalid(field.NewPath("spec", "serviceEngineGroup"), r.Spec.ServiceEngineGroup,
			"failed to get service engine group from avi controller:"+err.Error())
	}
	return nil
}

// validateAviControlPlaneNetworks checks input Control Plane Network name existing or not, CIDR format valid or not
func (r *AKODeploymentConfig) validateAviControlPlaneNetworks() field.ErrorList {
	var allErrs field.ErrorList
	if r.Spec.ControlPlaneNetwork.Name == "" || r.Spec.ControlPlaneNetwork.CIDR == "" {
		return allErrs
	}
	// check control plane network name
	if _, err := aviClient.NetworkGetByName(r.Spec.ControlPlaneNetwork.Name, r.Spec.CloudName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "controlPlaneNetwork", "name"),
			r.Spec.ControlPlaneNetwork.Name,
			"failed to get control plane network "+r.Spec.ControlPlaneNetwork.Name+" from avi controller:"+err.Error()))
	}
	// check network cidr validate or not
	_, _, err := net.ParseCIDR(r.Spec.ControlPlaneNetwork.CIDR)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "controlPlaneNetwork", "cidr"),
			r.Spec.ControlPlaneNetwork.CIDR,
			"control plane network cidr "+r.Spec.ControlPlaneNetwork.CIDR+" is not valid:"+err.Error()))
	}
	return allErrs
}

// validateAviDataNetworks checks input
// Data Plane Network name existing or not
// CIDR format valid or not
// IPPools format valid or not
func (r *AKODeploymentConfig) validateAviDataNetworks() field.ErrorList {
	var allErrs field.ErrorList
	// check data network name
	if _, err := aviClient.NetworkGetByName(r.Spec.DataNetwork.Name, r.Spec.CloudName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "name"),
			r.Spec.DataNetwork.Name,
			"failed to get data plane network "+r.Spec.DataNetwork.Name+" from avi controller:"+err.Error()))
	}
	// check network cidr
	addr, cidr, err := net.ParseCIDR(r.Spec.DataNetwork.CIDR)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "cidr"),
			r.Spec.DataNetwork.CIDR,
			"data plane network cidr "+r.Spec.DataNetwork.CIDR+" is not valid:"+err.Error()))
	}
	addrType := "INVALID"
	if addr.To4() != nil {
		addrType = "V4"
	} else if addr.To16() != nil {
		addrType = "V6"
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
				ipPool.Start+" is greater than "+ipPool.End))
		}
		if ipPool.Type != addrType {
			return append(allErrs, field.Invalid(field.NewPath("spec", "dataNetwork", "ipPools"),
				r.Spec.DataNetwork.IPPools,
				"data plane network ip pools type is not aligned with cidr"))
		}
	}
	return allErrs
}
