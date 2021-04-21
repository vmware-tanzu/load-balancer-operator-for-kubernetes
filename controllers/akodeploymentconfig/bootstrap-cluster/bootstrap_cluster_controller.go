// Copyright (c) 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package bootstrap_cluster

import (
	"bytes"
	"context"
	"os"
	"text/template"

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

	if obj.Name != akoov1alpha1.ManagementClusterAkoDeploymentConfig || !ako_operator.IsBootStrapCluster() {
		log.Info("not bootstrap cluster akodeploymentconfig, skip")
		return res, nil
	}

	log.Info("Start reconciling akodeploymentconfig")
	// convert from akodeploymentconfig to ako deployment components
	components, err := ConvertToAKODeploymentYaml(obj)
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

func ConvertToAKODeploymentYaml(obj *akoov1alpha1.AKODeploymentConfig) ([]unstructured.Unstructured, error) {
	tmpl, err := template.New("deployment").Parse(ako.AkoDeploymentYamlTemplate)
	if err != nil {
		return nil, err
	}

	managementClusterName := os.Getenv(ako_operator.ManagementClusterName)
	values, err := ako.PopulateValues(obj, akoov1alpha1.TKGSystemNamespace+"-"+managementClusterName)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, map[string]interface{}{
		"Values": values,
	})

	if err != nil {
		return nil, err
	}
	return yaml.ToUnstructured(buf.Bytes())
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
