// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	"github.com/avinetworks/sdk/go/models"
	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/aviclient"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/controllers/remote"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AkoUserReconciler reconcile avi user related resources
type AkoUserReconciler struct {
	client.Client
	aviClient *aviclient.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// NewReconciler returns AKOUserReconciler object.
func NewProvider(client client.Client,
	aviClient *aviclient.Client,
	logger logr.Logger,
	scheme *runtime.Scheme) *AkoUserReconciler {
	return &AkoUserReconciler{Client: client,
		aviClient: aviClient,
		Log:       logger,
		Scheme:    scheme}
}

// ReconcileAviUser: reconcile akodeploymentconfig clusters' avi user
func (r *AkoUserReconciler) ReconcileAviUser(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	if cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		return ctrl.Result{}, nil
	}
	log.V(1).Info("Start reconciling workload cluster avi credentials")
	if !cluster.GetDeletionTimestamp().IsZero() {
		log.Info("reconcile deleting workload cluster avi user resource")
		return r.ReconcileAviUserDelete(ctx, log, cluster, obj)
	}

	return r.reconcileAviUserNormal(ctx, log, obj, cluster)
}

// ReconcileAviUserDelete clean up all avi user account related resources when workload cluster delete or
// choose to disable avi
// Note: only resources in the management cluster will be cleaned up
func (r *AkoUserReconciler) ReconcileAviUserDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	mcSecretName, mcSecretNamespace := r.mcAVISecretNameNameSpace(cluster.Name, cluster.Namespace, obj.Spec.WorkloadCredentialRef)

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      mcSecretName,
		Namespace: mcSecretNamespace,
	}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get AKO secret in the management cluster, requeue")
			return res, err
		} else {
			log.Info("AKO secret in the management cluster is already gone")
			return res, nil
		}
	}

	if err := r.aviClient.User.DeleteByName(string(secret.Data["username"])); err != nil {
		log.Error(err, "Failed to delete avi user account in avi controller, requeue")
		return res, err
	}

	if err := r.Client.Delete(ctx, secret); err != nil {
		if !apierrors.IsGone(err) {
			log.Error(err, "Failed to delete avi secret in the management cluster, requeue")
			return res, err
		}
	}
	return res, nil
}

// reconcileAviUserNormal ensure each workload cluster has an independent avi user
func (r *AkoUserReconciler) reconcileAviUserNormal(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
	cluster *clusterv1.Cluster,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	// ensures the AVI Controller Certificate Authority exists
	aviControllerCASecret, err := r.getAVIControllerCA(ctx, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Avi controller ca not found, requeue the request")
		} else {
			log.Error(err, "Failed to get avi controller ca")
		}
		return res, err
	}

	aviCA := string(aviControllerCASecret.Data[akoov1alpha1.AviCertificateKey][:])
	// Ensures the management cluster Secret exists
	mcSecret := &corev1.Secret{}
	mcSecretName, mcSecretNamespace := r.mcAVISecretNameNameSpace(cluster.Name, cluster.Namespace, obj.Spec.WorkloadCredentialRef)
	// Secret in the management cluster acts as the source of truth so we
	// avoid generating the password multiple times
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      mcSecretName,
		Namespace: mcSecretNamespace,
	}, mcSecret); err != nil {
		if apierrors.IsNotFound(err) {

			aviUsername := cluster.Name + "-" + cluster.Namespace + "-ako-user"
			// This can only happen once no matter how many times we
			// enter the reconciliation
			aviPassword := utils.GenereatePassword(10, true, true, true, true)

			managementClusterSecret := r.createAviUserSecret(
				mcSecretName,
				mcSecretNamespace,
				aviUsername,
				aviPassword,
				aviCA,
				obj,
				false,
			)
			log.Info("No AVI Secret found for cluster in the management cluster, start the creation")
			if err := r.Client.Create(ctx, managementClusterSecret); err != nil {
				log.Error(err, "Failed to create AVI secret for Cluster in the management cluster, requeue")
				return res, err
			}
		} else {
			log.Error(err, "Failed to get cluster avi user secret, requeue")
			return res, err
		}
	}

	// Use the value from the management cluster
	aviUsername := string(mcSecret.Data["username"][:])
	aviPassword := string(mcSecret.Data["password"][:])

	// ensures the AVI User exists and matches the mc secret
	if _, err = r.createOrUpdateAviUser(aviUsername, aviPassword, obj.Spec.Tenant.Name); err != nil {
		log.Error(err, "Failed to create/update cluster avi user")
		return res, err
	} else {
		log.Info("Successfully created/updated AVI User in AVI Controller")
	}

	if res, err = r.createOrUpdateWorkloadClusterSecret(ctx, log, cluster, obj, aviUsername, aviPassword, aviCA); err != nil {
		return res, err
	}

	return res, nil
}

// createOrUpdateWorkloadClusterSecret ensure the AKO Secret exist in the target
// workload cluster
func (r *AkoUserReconciler) createOrUpdateWorkloadClusterSecret(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
	aviUsername string,
	aviPassword string,
	aviCA string,
) (ctrl.Result, error) {
	var found bool
	var res ctrl.Result

	remoteClient, err := r.createWorkloadClusterClient(ctx, cluster.Name, cluster.Namespace)
	if err != nil {
		log.Error(err, "Failed to create client for cluster, requeue")
		return res, err
	}

	// ensures the workload cluster Secret exists and matches the mc secret
	wcSecret := &corev1.Secret{}
	if err := remoteClient.Get(ctx, client.ObjectKey{
		Name:      akoov1alpha1.AviSecretName,
		Namespace: akoov1alpha1.AviNamespace,
	}, wcSecret); err != nil {
		// ignore notfound error
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get AKO Secret in the workload cluster, requeue")
			return res, err
		}
	} else {
		found = true
	}

	if !found {
		wcSecret = r.createAviUserSecret(
			akoov1alpha1.AviSecretName,
			akoov1alpha1.AviNamespace,
			aviUsername,
			aviPassword,
			aviCA,
			obj,
			true,
		)
	} else {
		// Update secret Data
		wcSecret.Data["username"] = []byte(aviUsername)
		wcSecret.Data["password"] = []byte(aviPassword)
		wcSecret.Data[akoov1alpha1.AviCertificateKey] = []byte(aviCA)
	}

	if !found {
		log.Info("No AKO Secret found in th workload cluster, start the creation")
		if err := remoteClient.Create(ctx, wcSecret); err != nil {
			log.Error(err, "Failed to create AKO Secret in the workload cluster, requeue")
			return res, err
		}
	} else {
		log.Info("Updating AKO Secret in the workload cluster")
		if err := remoteClient.Update(ctx, wcSecret); err != nil {
			log.Error(err, "Failed to update AKO Secret in the workload cluster, requeue")
			return res, err
		}
	}
	return ctrl.Result{}, nil
}

// listAkoDeplymentConfigDeployedClusters list all clusters enabled current akodeploymentconfig
func (r *AkoUserReconciler) listAkoDeplymentConfigDeployedClusters(ctx context.Context, obj *akoov1alpha1.AKODeploymentConfig) (*clusterv1.ClusterList, error) {
	selector, err := metav1.LabelSelectorAsSelector(&obj.Spec.ClusterSelector)
	if err != nil {
		return nil, err
	}
	listOptions := []client.ListOption{
		client.InNamespace(obj.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}
	var clusters clusterv1.ClusterList
	if err := r.Client.List(ctx, &clusters, listOptions...); err != nil {
		return nil, err
	}
	return &clusters, nil
}

// getAVIControllerCA get avi certificateAuthority secret
func (r *AkoUserReconciler) getAVIControllerCA(ctx context.Context, obj *akoov1alpha1.AKODeploymentConfig) (*corev1.Secret, error) {
	aviControllerCA := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{
		Name:      obj.Spec.CertificateAuthorityRef.Name,
		Namespace: obj.Spec.CertificateAuthorityRef.Namespace,
	}, aviControllerCA)
	return aviControllerCA, err
}

// createOrUpdateAviUser create an avi user in avi controller
func (r *AkoUserReconciler) createOrUpdateAviUser(aviUsername, aviPassword, tenantName string) (*models.User, error) {
	aviUser, err := r.aviClient.User.GetByName(aviUsername)
	// user not found, create one
	if aviclient.IsAviUserNonExistentError(err) {
		// for avi essential version the default tenant is admin
		if tenantName == "" {
			tenantName = "admin"
		}
		tenant, err := r.aviClient.Tenant.Get(tenantName)
		if err != nil {
			return nil, err
		}
		role, err := r.getOrCreateAkoUserRole(tenant.URL)
		if err != nil {
			return nil, err
		}
		aviUser = &models.User{
			Name:             &aviUsername,
			Password:         &aviPassword,
			DefaultTenantRef: tenant.URL,
			Access: []*models.UserRole{
				{
					AllTenants: pointer.BoolPtr(false),
					RoleRef:    role.URL,
					TenantRef:  tenant.URL,
				},
			},
		}
		return r.aviClient.User.Create(aviUser)
	}
	// Update the password when user found, this is needed when the AVI user was
	// created before the mc Secret. And this operation will sync
	// the User's password to be the same as mc Secret's
	if err == nil {
		aviUser.Password = &aviPassword
		return r.aviClient.User.Update(aviUser)
	}
	return nil, err
}

// getOrCreateAkoUserRole get ako user's role, create one if not exist
func (r *AkoUserReconciler) getOrCreateAkoUserRole(roleTenantRef *string) (*models.Role, error) {
	role, err := r.aviClient.Role.GetByName(akoov1alpha1.AkoUserRoleName)
	//not found ako user role, create one
	if aviclient.IsAviRoleNonExistentError(err) {
		role = &models.Role{
			Name:       pointer.StringPtr(akoov1alpha1.AkoUserRoleName),
			Privileges: AkoRolePermission,
			TenantRef:  roleTenantRef,
		}
		return r.aviClient.Role.Create(role)
	}
	// else return role or error
	return role, err
}

// mcAVISecretNameNameSpace get avi user secret name/namespace in management cluster. There is no need to
// encode the cluster namespace as the secret is deployed in the same namespace as
// the cluster
func (r *AkoUserReconciler) mcAVISecretNameNameSpace(clusterName, clusterNamespace string, secretRef akoov1alpha1.SecretReference) (name, namespace string) {
	if secretRef.Name != "" {
		name = secretRef.Name
	} else {
		name = clusterName + "-" + "avi-credentials"
	}
	if secretRef.Namespace != "" {
		namespace = secretRef.Namespace
	} else {
		namespace = clusterNamespace
	}
	return name, namespace
}

// createAviUserSecret create a secret to store avi user credentials
func (r *AkoUserReconciler) createAviUserSecret(name, namespace, username, password string, aviCA string, obj *akoov1alpha1.AKODeploymentConfig, isWorkloadCluster bool) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: akoov1alpha1.AviClusterSecretType,
		Data: make(map[string][]byte),
	}
	if !isWorkloadCluster {
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			{
				UID:                obj.UID,
				Name:               obj.Name,
				Controller:         pointer.BoolPtr(true),
				BlockOwnerDeletion: pointer.BoolPtr(true),
				Kind:               akoov1alpha1.AkoDeploymentConfigKind,
				APIVersion:         akoov1alpha1.AkoDeploymentConfigVersion,
			},
		}

	}
	secret.Data["username"] = []byte(username)
	secret.Data["password"] = []byte(password)
	secret.Data[akoov1alpha1.AviCertificateKey] = []byte(aviCA)
	return secret
}

// createWorkloadClusterClient create workload cluster corresponding client
func (r *AkoUserReconciler) createWorkloadClusterClient(ctx context.Context, clusterName, clusterNamespace string) (client.Client, error) {
	remoteClient, err := remote.NewClusterClient(ctx, r.Client, client.ObjectKey{
		Name:      clusterName,
		Namespace: clusterNamespace,
	}, r.Scheme)
	return remoteClient, err
}
