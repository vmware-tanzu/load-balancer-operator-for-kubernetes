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
	kerrors "k8s.io/apimachinery/pkg/util/errors"
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

// ReconcileAviUsers: reconcile akodeploymentconfig clusters' avi user
func (r *AkoUserReconciler) ReconcileAviUsers(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) (ctrl.Result, error) {
	log.Info("Start reconciling workload cluster avi credentials")
	// select all clusters deployed by current AKODeploymentConfig
	clusters, err := r.listAkoDeplymentConfigDeployedClusters(ctx, obj)
	if err != nil {
		log.Error(err, "Fail to list clusters deployed by current AKODeploymentConfig")
		return ctrl.Result{}, err
	}
	var errs []error
	for _, cluster := range clusters.Items {
		if _, exist := cluster.Labels[akoov1alpha1.AviClusterLabel]; !exist {
			log.Info("Cluster doesn't have AVI enabled, skip reconciling")
		} else if !cluster.GetDeletionTimestamp().IsZero() {
			log.Info("reconcile deleting workload cluster avi user resource")
			if err := r.reconcileAviUserDelete(ctx, log, &cluster, obj); err != nil {
				log.Error(err, "Fail to reconcile delete cluster deployed by current AKODeploymentConfig")
				errs = append(errs, err)
			}
		} else if _, err := r.reconcileAviUserNormal(ctx, log, obj, &cluster); err != nil {
			log.Error(err, "Fail to reconcile cluster deployed by current AKODeploymentConfig")
			errs = append(errs, err)
		}
	}
	return ctrl.Result{}, kerrors.NewAggregate(errs)
}

func (r *AkoUserReconciler) ReconcileAviUsersDelete(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
) error {
	clusters, err := r.listAkoDeplymentConfigDeployedClusters(ctx, obj)
	if err != nil {
		log.Error(err, "Fail to list clusters deployed by current AKODeploymentConfig")
		return err
	}
	var errs []error
	// clean up each clusters' avi user account resources
	for _, cluster := range clusters.Items {
		if err := r.reconcileAviUserDelete(ctx, log, &cluster, obj); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return kerrors.NewAggregate(errs)
	}
	return nil
}

// reconcileAviUserDelete clean up all avi user account related resources when workload cluster delete or
// choose to disable avi
func (r *AkoUserReconciler) reconcileAviUserDelete(
	ctx context.Context,
	log logr.Logger,
	cluster *clusterv1.Cluster,
	obj *akoov1alpha1.AKODeploymentConfig,
) error {
	mcSecretName := r.getAviSecretName(cluster.Name, cluster.Namespace)
	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: mcSecretName, Namespace: obj.Namespace}, secret); apierrors.IsNotFound(err) {
		log.Info("Can't find avi account secret")
		return nil
	} else if err != nil {
		log.Error(err, "Failed to get avi user account secret")
		return err
	}
	if err := r.aviClient.User.Delete(string(secret.Data["uuid"])); err != nil {
		log.Error(err, "Failed to delete avi user account in avi controller")
		return err
	} else if err := r.Client.Delete(ctx, secret); err != nil {
		log.Error(err, "Failed to delete avi secret in management cluster")
		return err
	} else if remoteClient, err := r.createWorkloadClusterClient(ctx, cluster.Name, cluster.Namespace); err != nil {
		log.Error(err, "Failed to get workload cluster client")
		return err
	} else if err := remoteClient.Get(ctx, client.ObjectKey{Name: akoov1alpha1.AviSecretName, Namespace: akoov1alpha1.AviNamespace}, secret); apierrors.IsNotFound(err) {
		log.Info("Can't find avi account secret, requeue request")
		return nil
	} else if err != nil {
		log.Error(err, "Failed to get avi secret in workload cluster")
		return err
	} else if err := remoteClient.Delete(ctx, secret); err != nil {
		log.Error(err, "Failed to delete avi user account workload cluster")
		return err
	}
	return nil
}

// reconcileAviUserNormal ensure each workload cluster has an independent avi user
func (r *AkoUserReconciler) reconcileAviUserNormal(
	ctx context.Context,
	log logr.Logger,
	obj *akoov1alpha1.AKODeploymentConfig,
	cluster *clusterv1.Cluster,
) (ctrl.Result, error) {
	// ensure each cluster has avi account
	res := ctrl.Result{}
	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      r.getAviSecretName(cluster.Name, cluster.Namespace),
		Namespace: obj.Namespace,
	}, secret); err == nil {
		log.Info("Avi user already created in management cluster")
		// check secret in workload cluster
		if err := r.getWorkloadClusterAviSecret(ctx, cluster); apierrors.IsNotFound(err) {
			log.Info("No Avi Secret in workload cluster, create one")
			if err := r.createWorkloadClusterAviSecret(ctx,
				obj,
				cluster,
				string(secret.Data["username"]),
				string(secret.Data["password"]),
				secret.Data[akoov1alpha1.AviCertificateKey]); err != nil {
				log.Error(err, "Failed to create secret in workload cluster")
				return res, err
			}
		} else if err != nil {
			log.Error(err, "failed to get secret in workload cluster")
			return res, nil
		}
		log.Info("Avi user already created in workload cluster")
		return res, nil
	} else if !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to get cluster avi user secret")
		return res, err
	}
	// need to generate an avi user for current cluster
	aviUsername := cluster.Name + "-" + cluster.Namespace + "-ako-user"
	aviPassword := utils.GenereatePassword(10, true, true, true, true)
	if aviUser, err := r.createAviUser(aviUsername, aviPassword); err != nil {
		log.Error(err, "Failed to create cluster avi user")
		return res, err
	} else if aviControllerCASecret, err := r.getAVIControllerCA(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Avi controller ca not found, requeue the request")
		} else {
			log.Error(err, "Failed to get avi controller ca")
		}
		return res, err
	} else {
		managementClusterSecret := r.createAviUserSecret(r.getAviSecretName(cluster.Name, cluster.Namespace),
			cluster.Namespace,
			aviUsername,
			aviPassword,
			*aviUser.UUID,
			aviControllerCASecret.Data[akoov1alpha1.AviCertificateKey],
			obj)
		if err := r.Client.Create(ctx, managementClusterSecret); err != nil {
			log.Error(err, "Failed to create secret in management cluster")
			return res, err
		}

		if err := r.createWorkloadClusterAviSecret(ctx,
			obj,
			cluster,
			aviUsername,
			aviPassword,
			aviControllerCASecret.Data[akoov1alpha1.AviCertificateKey]); err != nil {
			log.Error(err, "Failed to create secret in workload cluster")
			return res, err
		}
	}
	return res, nil
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

// createAviUser create an avi user in avi controller
func (r *AkoUserReconciler) createAviUser(aviUsername, aviPassword string) (*models.User, error) {
	// (xudongl) for avi essential version, there should be only one tenant, which is admin
	// (xudongl) TODO Role of AKO is still TBD
	if tenant, err := r.aviClient.Tenant.Get("admin"); err != nil {
		return nil, err
	} else if role, err := r.aviClient.Role.GetByName("Tenant-Admin"); err != nil {
		return nil, err
	} else if aviUser, err := r.aviClient.User.Create(&models.User{
		Name:             &aviUsername,
		Password:         &aviPassword,
		DefaultTenantRef: tenant.URL,
		Access: []*models.UserRole{
			{
				AllTenants: pointer.BoolPtr(false),
				RoleRef:    role.URL,
				TenantRef:  tenant.URL,
			}}}); err != nil {
		return nil, err
	} else {
		return aviUser, nil
	}
}

// getAviSecretName get avi user secret name in management cluster
func (r *AkoUserReconciler) getAviSecretName(clusterName, clusterNamespace string) string {
	return clusterName + "-" + clusterNamespace + "-" + akoov1alpha1.AviSecretName
}

// createAviUserSecret create a secret to store workload cluster credentials
func (r *AkoUserReconciler) createAviUserSecret(name, namespace, username, password, uuid string, aviCA []byte, obj *akoov1alpha1.AKODeploymentConfig) *corev1.Secret {
	workloadClusterAviSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					UID:                obj.UID,
					Name:               obj.Name,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
					Kind:               akoov1alpha1.AkoDeploymentConfigKind,
					APIVersion:         akoov1alpha1.AkoDeploymentConfigVersion,
				},
			},
		},
		Type: akoov1alpha1.AviClusterSecretType,
		Data: make(map[string][]byte),
	}
	workloadClusterAviSecret.Data["username"] = []byte(username)
	workloadClusterAviSecret.Data["password"] = []byte(password)
	workloadClusterAviSecret.Data["uuid"] = []byte(uuid)
	workloadClusterAviSecret.Data[akoov1alpha1.AviCertificateKey] = aviCA
	return workloadClusterAviSecret
}

// getWorkloadClusterAviSecret get workload cluster avi user secret
func (r *AkoUserReconciler) getWorkloadClusterAviSecret(ctx context.Context, cluster *clusterv1.Cluster) error {
	if remoteClient, err := r.createWorkloadClusterClient(ctx, cluster.Name, cluster.Namespace); err != nil {
		return nil
	} else if err := remoteClient.Get(ctx, client.ObjectKey{
		Name:      akoov1alpha1.AviSecretName,
		Namespace: akoov1alpha1.AviNamespace,
	}, &corev1.Secret{}); err != nil {
		return err
	}
	return nil
}

// createWorkloadClusterClient create workload cluster corresponding client
func (r *AkoUserReconciler) createWorkloadClusterClient(ctx context.Context, clusterName, clusterNamespace string) (client.Client, error) {
	remoteClient, err := remote.NewClusterClient(ctx, r.Client, client.ObjectKey{
		Name:      clusterName,
		Namespace: clusterNamespace,
	}, r.Scheme)
	return remoteClient, err
}

// createWorkloadClusterAviSecretSpec create a secret to store workload cluster credentials
func (r *AkoUserReconciler) createWorkloadClusterAviSecretSpec(name, username, password string, aviCA []byte, obj *akoov1alpha1.AKODeploymentConfig) *corev1.Secret {
	workloadClusterAviSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: akoov1alpha1.AviNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					UID:                obj.UID,
					Name:               obj.Name,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
					Kind:               akoov1alpha1.AkoDeploymentConfigKind,
					APIVersion:         akoov1alpha1.AkoDeploymentConfigVersion,
				},
			},
		},
		Type: akoov1alpha1.AviClusterSecretType,
		Data: make(map[string][]byte),
	}
	workloadClusterAviSecret.Data["username"] = []byte(username)
	workloadClusterAviSecret.Data["password"] = []byte(password)
	workloadClusterAviSecret.Data[akoov1alpha1.AviCertificateKey] = aviCA
	return workloadClusterAviSecret
}

// createWorkloadClusterAviSecret create avi secret in workload cluster
func (r *AkoUserReconciler) createWorkloadClusterAviSecret(ctx context.Context,
	obj *akoov1alpha1.AKODeploymentConfig,
	cluster *clusterv1.Cluster,
	aviUserName,
	aviPassword string,
	aviCA []byte,
) error {
	workloadClusterSecret := r.createWorkloadClusterAviSecretSpec(akoov1alpha1.AviSecretName, aviUserName, aviPassword, aviCA, obj)
	if remoteClient, err := r.createWorkloadClusterClient(ctx, cluster.Name, cluster.Namespace); err != nil {
		return err
	} else if err := remoteClient.Create(ctx, workloadClusterSecret); err != nil {
		return err
	}
	return nil
}
