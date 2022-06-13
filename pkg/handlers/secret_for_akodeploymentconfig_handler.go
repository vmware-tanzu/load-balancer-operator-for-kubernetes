// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func AkoDeploymentConfigForSecret(c client.Client, log logr.Logger) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		ctx := context.Background()
		secret, ok := o.(*corev1.Secret)
		if !ok {
			log.Error(errors.New("invalid type"),
				"Expected to receive Cluster resource",
				"actualType", fmt.Sprintf("%T", o))
			return nil
		}
		logger := log.WithValues("Secret", secret.Namespace+"/"+secret.Name)

		if secret.Name != akoov1alpha1.AviCredentialName && secret.Name != akoov1alpha1.AviCAName {
			return []reconcile.Request{}
		}

		var akoDeploymentConfigs akoov1alpha1.AKODeploymentConfigList
		if err := c.List(ctx, &akoDeploymentConfigs, []client.ListOption{}...); err != nil {
			logger.Error(err, "Couldn't read ADCs")
			return []reconcile.Request{}
		}

		var requests []ctrl.Request
		for _, akoDeploymentConfig := range akoDeploymentConfigs.Items {
			if akoDeploymentConfig.Spec.CertificateAuthorityRef.Name == secret.Name &&
				akoDeploymentConfig.Spec.CertificateAuthorityRef.Namespace == secret.Namespace {
				requests = append(requests, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Namespace: akoDeploymentConfig.Namespace,
						Name:      akoDeploymentConfig.Name,
					},
				})
			}
		}

		logger.Info("Generating requests", "requests", requests)
		// Return reconcile requests for the AKODeploymentConfig resources.
		return requests
	}
}
