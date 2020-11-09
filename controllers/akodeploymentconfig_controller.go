/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/aviclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (r *AKODeploymentConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&akoov1alpha1.AKODeploymentConfig{}).
		Complete(r)
}

// AKODeploymentConfigReconciler reconciles a AKODeploymentConfig object
type AKODeploymentConfigReconciler struct {
	client.Client
	aviClient *aviclient.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// +kubebuilder:rbac:groups=network.tanzu.vmware.com,resources=akodeploymentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.tanzu.vmware.com,resources=akodeploymentconfigs/status,verbs=get;update;patch

func (r *AKODeploymentConfigReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	log := r.Log.WithValues("AKODeploymentConfig", req.NamespacedName)
	res := ctrl.Result{}

	// Get the resource for this request.
	obj := &akoov1alpha1.AKODeploymentConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Machine not found, will not reconcile")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Always Patch when exiting this function so changes to the resource are updated on the API server.
	patchHelper, err := patch.NewHelper(obj, r.Client)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to init patch helper for %s %s",
			obj.GroupVersionKind(), req.NamespacedName)
	}
	defer func() {
		if err := patchHelper.Patch(ctx, obj); err != nil {
			if reterr == nil {
				reterr = err
			}
			log.Error(err, "patch failed")
		}
	}()

	if r.aviClient == nil {
		// TODO(fangyuanl): initialize aviClient from fields in
		// AKODeploymentConfig
	}

	// Handle deleted cluster resources.
	if !obj.GetDeletionTimestamp().IsZero() {
		res, err := r.reconcileDelete(ctx, log, obj)
		if err != nil {
			log.Error(err, "failed to reconcile AKODeploymentConfig deletion")
			return res, err
		}
		return res, nil
	}

	// Handle non-deleted resources.
	if res, err := r.reconcileNormal(ctx, log, obj); err != nil {
		log.Error(err, "failed to reconcile AKODeploymentConfig")
		return res, err
	}
	return res, nil
}

func (r *AKODeploymentConfigReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *AKODeploymentConfigReconciler) reconcileNormal(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
