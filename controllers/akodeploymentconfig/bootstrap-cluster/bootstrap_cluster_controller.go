// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package bootstrap_cluster

import (
	"bytes"
	"context"
	"os"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako"
	ako_operator "gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako-operator"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/cluster-api/util/resource"
	"sigs.k8s.io/cluster-api/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewReconciler initializes a BootstrapClusterReconciler
func NewReconciler(c client.Client, log logr.Logger, scheme *runtime.Scheme) *BootstrapClusterReconciler {
	return &BootstrapClusterReconciler{
		Client: c,
		Log:    log,
		Scheme: scheme,
	}
}

type BootstrapClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *BootstrapClusterReconciler) DeployAKO(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}
	if obj.Name != akoov1alpha1.ManagementClusterAkoDeploymentConfig {
		log.Info("not bootstrap cluster akodeploymentconfig, skip")
		return res, nil
	}
	log.Info("Start reconciling akodeploymentconfig")
	// convert from akodeploymentconfig to ako deployment components
	components, err := r.convertToAKODeploymentYaml(obj)
	if err != nil {
		log.Error(err, "Failed to populate values from akodeploymentconfig to ako deployment")
		return res, err
	}
	sortedComponents := resource.SortForCreate(components)
	var errList []error
	for i := range sortedComponents {
		if err := ApplyUnstructured(ctx, r.Client, &sortedComponents[i]); err != nil {
			log.Error(err, "Failed to apply ako deployment components")
			errList = append(errList, err)
		}
	}
	return res, kerrors.NewAggregate(errList)
}

func (r *BootstrapClusterReconciler) DeployAKOSecret(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}
	adminCredential := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      obj.Spec.AdminCredentialRef.Name,
		Namespace: obj.Spec.AdminCredentialRef.Namespace,
	}, adminCredential); err != nil {
		log.Error(err, "Failed to find referenced AdminCredential Secret")
		return res, err
	}
	aviControllerCA := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      obj.Spec.CertificateAuthorityRef.Name,
		Namespace: obj.Spec.CertificateAuthorityRef.Namespace,
	}, aviControllerCA); err != nil {
		log.Error(err, "Failed to get avi controller ca")
		return res, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      akoov1alpha1.AviSecretName,
			Namespace: akoov1alpha1.AviNamespace,
		},
		Type: akoov1alpha1.AviClusterSecretType,
		Data: map[string][]byte{
			"username":                     adminCredential.Data["username"],
			"password":                     adminCredential.Data["password"],
			akoov1alpha1.AviCertificateKey: aviControllerCA.Data[akoov1alpha1.AviCertificateKey],
		},
	}
	err := r.Client.Create(ctx, secret)
	if apierrors.IsAlreadyExists(err) {
		log.Info("avi secret already exist")
		return res, nil
	}
	return res, err
}

func (r *BootstrapClusterReconciler) DeleteAKO(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{
		Name: akoov1alpha1.AviNamespace,
	}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			log.V(3).Info("avi namespace is already deleted")
			return res, nil
		}
		log.Error(err, "Failed to get avi namespace, requeue")
		return res, err
	}
	deletePolicy := metav1.DeletePropagationForeground
	err := r.Client.Delete(ctx, ns, &client.DeleteOptions{PropagationPolicy: &deletePolicy})
	return res, err
}

func (r *BootstrapClusterReconciler) convertToAKODeploymentYaml(obj *akoov1alpha1.AKODeploymentConfig) ([]unstructured.Unstructured, error) {
	tmpl, err := template.New("deployment").Parse(ako.AkoDeploymentYamlTemplate)
	if err != nil {
		return nil, err
	}
	managementClusterName := os.Getenv(ako_operator.ManagementClusterName)
	values, err := ako.PopulateValues(obj, akoov1alpha1.TKGSystemNamespace+"-"+managementClusterName)
	if err != nil {
		return nil, err
	}
	r.modifyAKODeploymentForBootstrapCluster(&values)
	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, map[string]interface{}{"Values": values}); err != nil {
		return nil, err
	}
	return yaml.ToUnstructured(buf.Bytes())
}

func (r *BootstrapClusterReconciler) modifyAKODeploymentForBootstrapCluster(values *ako.Values) {
	// change to lower values since we don't need much resource in bootstrap cluster
	values.Resources.Requests = ako.Requests{
		Cpu:    "50m",
		Memory: "20Mi",
	}
}

func ApplyUnstructured(ctx context.Context, c client.Client, obj *unstructured.Unstructured) error {
	// Create the object on the API server.
	// TODO: Errors are only logged. If needed, exponential backoff or requeuing could be used here for remedying connection glitches etc.
	if err := c.Create(ctx, obj); err != nil {
		// The create call is idempotent, so if the object already exists
		// then do not consider it to be an error.
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(
				err,
				"failed to create object %s %s/%s",
				obj.GroupVersionKind(),
				obj.GetNamespace(),
				obj.GetName())
		}
	}
	return nil
}
