// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako_operator

import (
	"context"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateExistingAKODeploymentConfig(mgr ctrl.Manager) error {
	log := ctrl.Log.WithName("one-time-process").WithName("ADCScanner")

	log.Info("update existing ADCs controllerVersion")

	c := mgr.GetClient()
	ctx := context.Background()

	// get existing ADC
	var adcs akoov1alpha1.AKODeploymentConfigList
	if err := c.List(ctx, &adcs, []client.ListOption{}...); err != nil {
		log.Error(err, "failed to list all AKODeploymentConfig objects")
		return err
	}

	if len(adcs.Items) == 0 {
		log.Info("no existing ADCs, skip")
		return nil
	}

	var (
		aviClient aviclient.Client
		err       error
		version   string
	)
	for _, adc := range adcs.Items {
		// initialize client once since only single Controller is supported
		if aviClient == nil {
			aviClient, err = aviclient.NewAviClientFromSecrets(c, ctx, log, adc.Spec.Controller,
				adc.Spec.AdminCredentialRef.Name, adc.Spec.AdminCredentialRef.Namespace,
				adc.Spec.CertificateAuthorityRef.Name, adc.Spec.CertificateAuthorityRef.Namespace,
				adc.Spec.ControllerVersion)
			if err != nil {
				log.Error(err, "Cannot init AVI clients from ADC", adc.Namespace, adc.Name)
				return err
			}
			log.Info("AVI Client initialized successfully")

			version, err = aviClient.GetControllerVersion()
			if err != nil {
				return field.Invalid(field.NewPath("spec", "Controller"), adc.Spec.Controller, "failed to get avi controller version:"+err.Error())
			}
		}

		if adc.Spec.ControllerVersion != version {
			// patch the adc if the version doesn't match
			patch := client.MergeFrom(adc.DeepCopy())
			oldVersion := adc.Spec.ControllerVersion
			adc.Spec.ControllerVersion = version

			if err := c.Patch(ctx, &adc, patch); err != nil {
				log.Error(err, "failed to patch ADC", adc.Namespace, adc.Name)
				return err
			}
			log.Info("patched ADC's controller version", adc.Namespace, adc.Name, "from", version, "to", oldVersion)
		}

	}

	return nil
}
