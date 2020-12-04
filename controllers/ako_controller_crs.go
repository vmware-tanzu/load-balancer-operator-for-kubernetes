// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/go-logr/logr"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	clustereaddonv1alpha3 "sigs.k8s.io/cluster-api/exp/addons/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CrsWorkloadClusterResourceName = "ako-deployment"
)

func getAKODeploymentYaml() (string, error) {
	tmpl, err := template.New("deployment").Parse(akoDeploymentYaml)
	if err != nil {
		return "", err
	}

	values := Values{}

	// default setting
	values.AKOSettings.ApiServerPort = 8080
	values.NetworkSettings.NodeNetworkList = make([]NodeNetwork, 0)

	// TODO(lxinqi): use the real value defined by user
	values.ReplicaCount = 1

	values.Image.Repository = "avinetworks/ako"
	values.Image.PullPolicy = "IfNotPresent"

	values.AKOSettings.LogLevel = "INFO"
	values.AKOSettings.FullSyncFrequency = "1800"
	values.AKOSettings.ApiServerPort = 8080
	values.AKOSettings.DeleteConfig = false
	values.AKOSettings.DisableStaticRouteSync = false
	values.AKOSettings.ClusterName = "workload-cls"
	values.AKOSettings.CniPlugin = "calico"
	values.AKOSettings.SyncNamespace = ""

	values.NetworkSettings.SubnetIP = "10.182.0.0"
	values.NetworkSettings.SubnetPrefix = "20"
	values.NetworkSettings.NetworkName = "VM Network"

	values.L7Settings.DefaultIngController = true
	values.L7Settings.L7ShardingScheme = "hostname"
	values.L7Settings.ServiceType = "NodePort"
	values.L7Settings.ShardVSSize = "LARGE"
	values.L7Settings.PassthroughShardSize = "SMALL"

	values.L4Settings.DefaultDomain = "tkgm.test.vmware.com"

	values.ControllerSettings.ServiceEngineGroupName = "Default-Group"
	values.ControllerSettings.ControllerVersion = "20.1.2"
	values.ControllerSettings.CloudName = "Default-Cloud"
	values.ControllerSettings.ControllerIP = "10.182.15.127"

	values.NodePortSelector.Key = ""
	values.NodePortSelector.Value = ""

	values.Resources.Limits.Cpu = "250m"
	values.Resources.Limits.Memory = "300Mi"
	values.Resources.Requests.Cpu = "100m"
	values.Resources.Requests.Memory = "200Mi"

	values.Rbac.PspEnable = false

	values.Avicredentials.Username = "admin"
	values.Avicredentials.Password = "Admin!23"

	values.Service.Type = "ClusterIP"
	values.Service.Port = 80

	values.PersistentVolumeClaim = ""
	values.MountPath = "/log"
	values.LogFile = "avi.log"

	values.NameOverride = ""
	values.Name = values.GetName(values.NameOverride)
	values.AppVersion = "1.2.1"
	values.ChartName = "ako"
	values.PsppolicyApiVersion = "policy/v1beta1" // or "extensions/v1beta1"

	// preprocessing
	nodeNetworkListJson, jsonerr := json.Marshal(values.NetworkSettings.NodeNetworkList)
	if jsonerr != nil {
		fmt.Println("Can't convert network setting into json. Error: ", jsonerr)
	}
	values.NetworkSettings.NodeNetworkListJson = string(nodeNetworkListJson)

	values.Avicredentials.Username = base64.StdEncoding.EncodeToString([]byte(values.Avicredentials.Username))
	values.Avicredentials.Password = base64.StdEncoding.EncodeToString([]byte(values.Avicredentials.Password))

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, map[string]interface{}{
		"Values": values,
	})

	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func getWorkloadClusterDeploymentSecret(namespace string) (*corev1.Secret, error) {
	workloadClusterDeploymentSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CrsWorkloadClusterResourceName,
			Namespace: namespace,
		},
		Type: clustereaddonv1alpha3.ClusterResourceSetSecretType,
		Data: make(map[string][]byte),
	}
	akoDeploymentYaml, err := getAKODeploymentYaml()
	if err != nil {
		return nil, err
	} else {
		workloadClusterDeploymentSecret.Data["ako-deployment"] = []byte(akoDeploymentYaml)
	}
	return workloadClusterDeploymentSecret, nil
}

var (
	workloadClusterCRS = &clustereaddonv1alpha3.ClusterResourceSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: CrsWorkloadClusterResourceName,
		},
		Spec: clustereaddonv1alpha3.ClusterResourceSetSpec{
			ClusterSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					akoov1alpha1.AviClusterLabel: "",
				},
			},
			Resources: []clustereaddonv1alpha3.ResourceRef{
				{
					Name: CrsWorkloadClusterResourceName,
					Kind: "Secret",
				},
			},
		},
	}
)

// reconcileCRS creates the CRS for AKO deployment in workload clusters
func (r *ClusterReconciler) reconcileCRS(
	ctx context.Context,
	log logr.Logger,
	obj *clusterv1.Cluster,
) (ctrl.Result, error) {
	log.Info("Starts reconciling ClusterResourceSet", "cluster", obj.Namespace+"/"+obj.Name)
	ns := obj.Namespace

	res := ctrl.Result{}
	var errs []error
	s := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      CrsWorkloadClusterResourceName,
		Namespace: ns,
	}, s); err == nil {
		// Update Secret since it already exists
		s, err = getWorkloadClusterDeploymentSecret(ns)
		log.Info("Update existing secret")
		if err != nil {
			errs = append(errs, err)
			log.Error(err, "Cann't get secret")
		} else {
			if err := r.Update(ctx, s); err != nil {
				errs = append(errs, err)
				log.Error(err, "Cann't update secret")
			}
		}
	} else {
		if apierrors.IsNotFound(err) {
			s, err = getWorkloadClusterDeploymentSecret(ns)

			if err != nil {
				errs = append(errs, err)
			} else {
				if err := r.Create(ctx, s); err != nil {
					errs = append(errs, err)
					log.Error(err, "Cann't create secret")
				} else {
					log.Info("Creating secret")
				}
			}
		} else {
			errs = append(errs, err)
		}
	}

	crs := &clustereaddonv1alpha3.ClusterResourceSet{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      workloadClusterCRS.Name,
		Namespace: ns,
	}, crs); err != nil {
		log.Info("Ready to get crs")
		if !apierrors.IsNotFound(err) {
			errs = append(errs, err)
			log.Error(err, "Cann't get crs")
		} else {
			crs = workloadClusterCRS.DeepCopy()
			crs.Namespace = ns
			if err := r.Create(ctx, crs); err != nil {
				errs = append(errs, err)
				log.Error(err, "Cann't create crs")
			} else {
				log.Info("CRS created")
			}
		}
	} else {
		log.Info("CRS already exists")
	}
	return res, kerrors.NewAggregate(errs)
}
