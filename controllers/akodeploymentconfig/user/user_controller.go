// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/vmware/alb-sdk/go/models"
	"golang.org/x/mod/semver"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/aviclient"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/utils"
)

// AkoUserReconciler reconcile avi user related resources
type AkoUserReconciler struct {
	client.Client
	aviClient aviclient.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

// NewProvider returns AKOUserReconciler object.
func NewProvider(client client.Client,
	aviClient aviclient.Client,
	logger logr.Logger,
	scheme *runtime.Scheme,
) *AkoUserReconciler {
	return &AkoUserReconciler{
		Client:    client,
		aviClient: aviClient,
		Log:       logger,
		Scheme:    scheme,
	}
}

// ReconcileAviUser reconcile akodeploymentconfig clusters' avi user
func (r *AkoUserReconciler) ReconcileAviUser(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Start reconciling workload cluster avi credentials")
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
	// cluster existing, don't delete avi user
	if cluster.GetDeletionTimestamp().IsZero() {
		log.Info("workload cluster existing, don't delete avi user")
		return ctrl.Result{}, nil
	}
	// Check if there is a cleanup condition in the Cluster status, if not, update it
	if !conditions.Has(cluster, akoov1alpha1.AviUserCleanupSucceededCondition) {
		conditions.MarkFalse(cluster, akoov1alpha1.AviUserCleanupSucceededCondition, akoov1alpha1.AviResourceCleanupReason, clusterv1.ConditionSeverityInfo, "Cleaning up the AVI load balancing user credentials before deletion")
		log.Info("Trigger the Avi user cleanup in the target Cluster and set Cluster condition", "condition", akoov1alpha1.AviUserCleanupSucceededCondition)
	}

	if conditions.IsTrue(cluster, akoov1alpha1.AviResourceCleanupSucceededCondition) {
		log.Info("Cluster avi resources deleted, start deleting avi user credentials")
		return r.reconcileAviUserDelete(ctx, log, cluster, obj)
	}

	log.Info("Wait until AVI resource deletion for cluster finishes, requeue", "condition", akoov1alpha1.AviResourceCleanupSucceededCondition)
	return ctrl.Result{}, errors.Errorf("requeue to wait AVI resource deletion for cluster: %s/%s", cluster.Namespace, cluster.Name)
}

// reconcileAviUserDelete clean up all avi user account related resources when workload cluster delete or
// choose to disable avi
// Note: only resources in the management cluster will be cleaned up
func (r *AkoUserReconciler) reconcileAviUserDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	res := ctrl.Result{}

	if cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		log.Info("No need to clean admin user, skip")
		conditions.MarkTrue(cluster, akoov1alpha1.AviUserCleanupSucceededCondition)
		return res, nil
	}

	if conditions.IsTrue(cluster, akoov1alpha1.AviUserCleanupSucceededCondition) {
		log.Info("AVI user credentails were cleaned up before, skip")
		return res, nil
	}

	if obj.Spec.WorkloadCredentialRef != nil {
		log.Info("AVI user credentials managed by customers, no need to delete, skip")
		conditions.MarkTrue(cluster, akoov1alpha1.AviUserCleanupSucceededCondition)
		return res, nil
	}

	mcSecretName, mcSecretNamespace := r.mcAVISecretNameNameSpace(cluster.Name, cluster.Namespace)

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
		}
	}

	if err := r.aviClient.UserDeleteByName(string(secret.Data["username"])); err != nil {
		log.Error(err, "Failed to delete avi user account in avi controller, requeue")
		return res, err
	}

	if err := r.Client.Delete(ctx, secret); err != nil {
		if !apierrors.IsGone(err) {
			log.Error(err, "Failed to delete avi secret in the management cluster, requeue")
			return res, err
		}
	}

	log.Info("AVI User credentials finished cleanup, updating Cluster condition")
	conditions.MarkTrue(cluster, akoov1alpha1.AviUserCleanupSucceededCondition)
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

	if cluster.Namespace == akoov1alpha1.TKGSystemNamespace {
		err = r.deployManagementClusterSecret(cluster, ctx, log, obj, aviControllerCASecret)
		if err != nil {
			log.Error(err, "Failed to generate avi-secret in management cluster")
		}
		return res, err
	}

	// Ensures the management cluster Secret exists
	mcSecret := &corev1.Secret{}
	if obj.Spec.WorkloadCredentialRef != nil {
		log.Info("AVI user credentials managed by customers")
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      obj.Spec.WorkloadCredentialRef.Name,
			Namespace: obj.Spec.WorkloadCredentialRef.Namespace,
		}, mcSecret); err != nil {
			log.Error(err, "Failed to get cluster avi user secret, requeue")
			return res, err
		}
	} else {
		log.Info("AVI user credentials managed by tkg system")
		mcSecretName, mcSecretNamespace := r.mcAVISecretNameNameSpace(cluster.Name, cluster.Namespace)
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

				mcSecret = r.createAviUserSecret(
					mcSecretName,
					mcSecretNamespace,
					aviUsername,
					aviPassword,
					aviCA,
					obj,
					false,
				)
				log.Info("No AVI Secret found for cluster in the management cluster, start the creation")
				if err := r.Client.Create(ctx, mcSecret); err != nil {
					log.Error(err, "Failed to create AVI secret for Cluster in the management cluster, requeue")
					return res, err
				}
			} else {
				log.Error(err, "Failed to get cluster avi user secret, requeue")
				return res, err
			}
		} else {
			// controller certificate can be updated by the user
			mcSecret.Data[akoov1alpha1.AviCertificateKey] = []byte(aviCA)
			if err := r.Client.Update(ctx, mcSecret); err != nil {
				log.Error(err, "Failed to update avi-credentials secret, requeue")
				return res, err
			}
		}

		// Use the value from the management cluster
		aviUsername := string(mcSecret.Data["username"][:])
		aviPassword := string(mcSecret.Data["password"][:])

		// ensures the AVI User exists and matches the mc secret
		if err = r.createOrUpdateAviUser(log, aviUsername, aviPassword, obj.Spec.Tenant.Name); err != nil {
			log.Error(err, "Failed to create/update cluster avi user")
			return res, err
		} else {
			log.Info("Successfully created/updated AVI User in AVI Controller")
		}
	}

	return res, nil
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
func (r *AkoUserReconciler) createOrUpdateAviUser(log logr.Logger, aviUsername, aviPassword, tenantName string) error {
	version, err := r.aviClient.GetControllerVersion()
	if err != nil {
		return err
	}
	// Add v prefix if not present so semver can parse it
	if len(version) > 0 && version[0] != 'v' {
		version = "v" + version
	}

	aviUser, err := r.aviClient.UserGetByName(aviUsername)
	// user not found, create one
	if aviclient.IsAviUserNonExistentError(err) {
		log.Info("AVI User not found, creating a new user", "user", aviUsername)
		// for avi essential version the default tenant is admin
		if tenantName == "" {
			tenantName = "admin"
		}
		tenant, err := r.aviClient.TenantGet(tenantName)
		if err != nil {
			return err
		}
		role, err := r.getOrCreateAkoUserRole(log, tenant.URL, version)
		if err != nil {
			return err
		}
		aviUser = &models.User{
			Name:             &aviUsername,
			Password:         &aviPassword,
			DefaultTenantRef: tenant.URL,
			Access: []*models.UserRole{
				{
					AllTenants: ptr.To(false),
					RoleRef:    role.URL,
					TenantRef:  tenant.URL,
				},
			},
		}
		// since v30.0.0, there is only enterprise edition
		if semver.Compare(version, akoov1alpha1.AVIControllerEnterpriseOnlyVersion) >= 0 {
			aviUser.Username = &aviUsername
		}
		if _, err := r.aviClient.UserCreate(aviUser); err != nil {
			return err
		}
		return nil
	} else if err != nil {
		log.Info("Failed to get AVI User", "user", aviUsername, "error", err)
		return err
	}

	// ensure user's role align with latest essential permission when user found
	if _, err := r.ensureAkoUserRole(log, version); err != nil {
		return err
	}
	// Update the password when user found, this is needed when the AVI user was
	// created before the mc Secret. And this operation will sync
	// the User's password to be the same as mc Secret's
	if aviUser.Password == nil || *aviUser.Password != aviPassword {
		log.Info("AVI User found, updating the password")
		aviUser.Password = &aviPassword
		if _, err := r.aviClient.UserUpdate(aviUser); err != nil {
			return err
		}
	}
	return nil
}

// getOrCreateAkoUserRole get ako user's role, create one if not exist
func (r *AkoUserReconciler) getOrCreateAkoUserRole(log logr.Logger, roleTenantRef *string, version string) (*models.Role, error) {
	log.Info("Ensure AKO User Role")
	role, err := r.aviClient.RoleGetByName(akoov1alpha1.AkoUserRoleName)
	// not found ako user role, create one
	if aviclient.IsAviRoleNonExistentError(err) {
		log.V(3).Info("Creating AKO User Role since it's not found", "role", akoov1alpha1.AkoUserRoleName)
		log.Info("current avi version", "version", version)
		role = &models.Role{
			Name:       ptr.To(akoov1alpha1.AkoUserRoleName),
			Privileges: filterAkoRolePermissionByVersion(log, AkoRolePermission, version),
			TenantRef:  roleTenantRef,
		}
		return r.aviClient.RoleCreate(role)
	}
	if err == nil {
		return r.ensureAkoUserRole(log, version)
	}
	return role, err
}

// ensureAkoUserRole ensure ako-essential-role has the latest permission
func (r *AkoUserReconciler) ensureAkoUserRole(log logr.Logger, version string) (*models.Role, error) {
	role, err := r.aviClient.RoleGetByName(akoov1alpha1.AkoUserRoleName)
	if err != nil {
		return role, err
	}

	// check if role needs to be synced
	if syncAkoUserRole(role, version) {
		log.Info("Syncing AKO User Role with expected permissions")
		return r.aviClient.RoleUpdate(role)
	}

	return role, nil
}

// syncAkoUserRole checks the Role for the complete/correct
// set of permissions and makes any additions/updates necessary.
// Any additional permissions on the Role that are not part of
// the desired AKO role are left as-is. It returns a bool
// indicating whether the Role was changed.
// It also filters out permissions that are deprecated in the current AVI version.
func syncAkoUserRole(role *models.Role, version string) bool {
	existingResources := sets.New[string]()
	updated := false

	for i, permission := range role.Privileges {
		desiredType, ok := AkoRolePermissionMap[*permission.Resource]
		if !ok {
			// Existing AVI role in AVI Controller has a permission that's not part of
			// the desired AKO role defined in AkoRolePermissionMap: leave it as-is.
			// Since those could come from a new AVI Controller version that AKO-Operator
			// Might not be aware of.
			continue
		}

		existingResources.Insert(*permission.Resource)

		if *permission.Type != desiredType {
			// Existing AVI role has a permission that's part of the
			// desired AKO role, but has the wrong type: update it.
			role.Privileges[i].Type = ptr.To(desiredType)
			updated = true
		}
	}

	for resource, desiredType := range AkoRolePermissionMap {
		if !existingResources.Has(resource) {
			// Filter out permissions that are deprecated in the current AVI version.
			if deprecateVersion, ok := deprecatePermissionMap[resource]; ok && semver.Compare(version, deprecateVersion) >= 0 {
				// Skip adding deprecated permissions to the role
				continue
			}
			// Existing AVI role is missing a permission that's
			// part of the desired AKO role: add it.
			role.Privileges = append(role.Privileges, &models.Permission{
				Resource: ptr.To(resource),
				Type:     ptr.To(desiredType),
			})
			updated = true
		}
	}

	return updated
}

// mcAVISecretNameNameSpace get avi user secret name/namespace in management cluster. There is no need to
// encode the cluster namespace as the secret is deployed in the same namespace as
// the cluster
func (r *AkoUserReconciler) mcAVISecretNameNameSpace(clusterName, clusterNamespace string) (name, namespace string) {
	name = clusterName + "-" + "avi-credentials"
	namespace = clusterNamespace
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
				Controller:         ptr.To(true),
				BlockOwnerDeletion: ptr.To(true),
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

func (r *AkoUserReconciler) deployManagementClusterSecret(
	cluster *clusterv1.Cluster,
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
	aviControllerCA *corev1.Secret,
) error {
	adminCredential := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      obj.Spec.AdminCredentialRef.Name,
		Namespace: obj.Spec.AdminCredentialRef.Namespace,
	}, adminCredential); err != nil {
		log.Error(err, "Failed to find referenced AdminCredential Secret", "secret namespace", obj.Spec.AdminCredentialRef.Namespace, "secret name", obj.Spec.AdminCredentialRef.Name)
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.AVIUserSecretName(cluster),
			Namespace: cluster.Namespace,
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
		log.Info("avi secret already exists, update avi-secret", "secret namespace", secret.Namespace, "secret name", secret.Name)
		return r.Client.Update(ctx, secret)
	}
	return err
}
