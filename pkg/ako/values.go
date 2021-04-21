// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ako

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
)

type Values struct {
	Name             string
	Namespace        string
	IsClusterService bool

	ReplicaCount       int
	Image              Image
	AKOSettings        AKOSettings
	NetworkSettings    NetworkSettings
	L7Settings         L7Settings
	L4Settings         L4Settings
	ControllerSettings ControllerSettings
	NodePortSelector   NodePortSelector
	Resources          Resources
	// PodSecurityContext    PodSecurityContext
	Rbac                  Rbac
	Avicredentials        Avicredentials
	PersistentVolumeClaim string
	MountPath             string
	LogFile               string
}

type Image struct {
	Repository string
	PullPolicy string // enum: INFO|DEBUG|WARN|ERROR
	Version    string
}

type AKOSettings struct {
	LogLevel               string
	FullSyncFrequency      string // This frequency controls how often AKO polls the Avi controller to update itself with cloud configurations.
	ApiServerPort          int    // Specify the port for the API server, default is set as 8080 // EmptyAllowed: false
	DeleteConfig           bool   // Has to be set to true in configmap if user wants to delete AKO created objects from AVI
	DisableStaticRouteSync bool   // If the POD networks are reachable from the Avi SE, set this knob to true.
	ClusterName            string // A unique identifier for the kubernetes cluster, that helps distinguish the objects for this cluster in the avi controller. // MUST-EDIT
	CniPlugin              string // Set the string if your CNI is calico or openshift. enum: calico|canal|flannel|openshift
	SyncNamespace          string
}

// This section outlines the network settings for virtualservices.
type NetworkSettings struct {
	SubnetIP            string // Subnet IP of the vip network
	SubnetPrefix        string // Subnet Prefix of the vip network
	NetworkName         string // Network Name of the vip network
	NodeNetworkList     []v1alpha1.NodeNetwork
	NodeNetworkListJson string
}

// This section outlines all the knobs  used to control Layer 7 loadbalancing settings in AKO.
type L7Settings struct {
	DisableIngressClass  bool
	DefaultIngController bool
	L7ShardingScheme     string
	ServiceType          string // enum NodePort|ClusterIP
	ShardVSSize          string // Use this to control the layer 7 VS numbers. This applies to both secure/insecure VSes but does not apply for passthrough. ENUMs: LARGE, MEDIUM, SMALL
	PassthroughShardSize string // Control the passthrough virtualservice numbers using this ENUM. ENUMs: LARGE, MEDIUM, SMALL
}

// This section outlines all the knobs  used to control Layer 4 loadbalancing settings in AKO.
type L4Settings struct {
	DefaultDomain string // If multiple sub-domains are configured in the cloud, use this knob to set the default sub-domain to use for L4 VSes.
}

// This section outlines settings on the Avi controller that affects AKO's functionality.
type ControllerSettings struct {
	ServiceEngineGroupName string // Name of the ServiceEngine Group.
	ControllerVersion      string // The controller API version
	CloudName              string // The configured cloud name on the Avi controller.
	ControllerIP           string
}

// Only applicable if serviceType is NodePort
type NodePortSelector struct {
	Key   string
	Value string
}

type Resources struct {
	Limits   Limits
	Requests Requests
}

type Limits struct {
	Cpu    string
	Memory string
}

type Requests struct {
	Cpu    string
	Memory string
}

// type podSecurityContext struct{}

// Creates the pod security policy if set to true
type Rbac struct {
	PspEnabled          bool
	PspPolicyApiVersion string
}

type Avicredentials struct {
	Username string
	Password string
}

func (v Values) GetName() string {
	return "ako-" + strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000), 10)
	//Be aware that during upgrades, templates are re-executed. When a template run generates data that differs from the last run, that will trigger an update of that resource.
}

func SetDefaultValues(values *Values) {
	values.AKOSettings = AKOSettings{
		LogLevel:               "INFO",
		ApiServerPort:          8080,
		DeleteConfig:           false,
		DisableStaticRouteSync: true,
		FullSyncFrequency:      "1800",
		// CniPlugin: don't set, use default value in AKO
		// SyncNamespace: don't set, use default value in AKO
		// ClusterName: populate in runtime
	}
	values.ReplicaCount = 1
	values.L7Settings = L7Settings{
		DefaultIngController: false,
		ServiceType:          "NodePort",
		ShardVSSize:          "SMALL",
		// L7ShardingScheme: don't set, use default value in AKO
		// PassthroughShardSize: don't set, use default value in AKO
	}
	values.L4Settings = L4Settings{
		// DefaultDomain: don't set, use default value in AKO
	}
	values.ControllerSettings = ControllerSettings{
		// ServiceEngineGroupName: populate in runtime
		// CloudName: populate in runtime
		// ControllerIP: populate in runtime
		// ControllerVersion: don't set, depend on AKO to autodetect,
		// also because we don't consider version skew in Calgary
	}
	values.NodePortSelector = NodePortSelector{
		// Key: don't set, use default value in AKO
		// Value: don't set, use default value in AKO
	}
	values.Resources = Resources{
		Limits: Limits{
			Cpu:    "250m",
			Memory: "300Mi",
		},
		Requests: Requests{
			Cpu:    "100m",
			Memory: "200Mi",
		},
	}
	values.NetworkSettings = NetworkSettings{
		// SubnetIP: don't set, populate in runtime
		// SubnetPrefix: don't set, populate in runtime
		// NetworkName: don't set, populate in runtime
		// NodeNetworkList: don't set, use default value in AKO
		// NodeNetworkListJson: don't set, use default value in AKO
	}
	if len(values.NetworkSettings.NodeNetworkList) != 0 {
		// preprocessing
		nodeNetworkListJson, jsonerr := json.Marshal(values.NetworkSettings.NodeNetworkList)
		if jsonerr != nil {
			fmt.Println("Can't convert network setting into json. Error: ", jsonerr)
		}
		values.NetworkSettings.NodeNetworkListJson = string(nodeNetworkListJson)
	}
	values.Namespace = akoov1alpha1.AviNamespace
}

func PopulateValues(obj *akoov1alpha1.AKODeploymentConfig, clusterNameSpacedName string) (Values, error) {
	values := Values{}

	SetDefaultValues(&values)

	values.Image.Repository = obj.Spec.ExtraConfigs.Image.Repository
	values.Image.PullPolicy = obj.Spec.ExtraConfigs.Image.PullPolicy
	values.Image.Version = obj.Spec.ExtraConfigs.Image.Version

	values.AKOSettings.ClusterName = clusterNameSpacedName

	values.AKOSettings.DisableStaticRouteSync = obj.Spec.ExtraConfigs.DisableStaticRouteSync

	values.ControllerSettings.CloudName = obj.Spec.CloudName
	values.ControllerSettings.ControllerIP = obj.Spec.Controller
	values.ControllerSettings.ServiceEngineGroupName = obj.Spec.ServiceEngineGroup

	network := obj.Spec.DataNetwork
	values.NetworkSettings.NetworkName = network.Name
	ip, ipnet, err := net.ParseCIDR(network.CIDR)
	if err != nil {
		return values, err
	}
	values.NetworkSettings.SubnetIP = ip.String()
	ones, _ := ipnet.Mask.Size()
	values.NetworkSettings.SubnetPrefix = strconv.Itoa(ones)
	values.NetworkSettings.NodeNetworkList = obj.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList

	if len(values.NetworkSettings.NodeNetworkList) != 0 {
		// preprocessing
		nodeNetworkListJson, jsonerr := json.Marshal(values.NetworkSettings.NodeNetworkList)
		if jsonerr != nil {
			return Values{}, jsonerr
		}
		values.NetworkSettings.NodeNetworkListJson = string(nodeNetworkListJson)
	}

	values.PersistentVolumeClaim = obj.Spec.ExtraConfigs.Log.PersistentVolumeClaim
	values.MountPath = obj.Spec.ExtraConfigs.Log.MountPath
	values.LogFile = obj.Spec.ExtraConfigs.Log.LogFile

	values.L7Settings.DisableIngressClass = obj.Spec.ExtraConfigs.IngressConfigs.DisableIngressClass
	values.L7Settings.DefaultIngController = obj.Spec.ExtraConfigs.IngressConfigs.DefaultIngressController
	if obj.Spec.ExtraConfigs.IngressConfigs.ShardVSSize != "" {
		values.L7Settings.ShardVSSize = obj.Spec.ExtraConfigs.IngressConfigs.ShardVSSize
	}
	if obj.Spec.ExtraConfigs.IngressConfigs.ServiceType != "" {
		values.L7Settings.ServiceType = obj.Spec.ExtraConfigs.IngressConfigs.ServiceType
	}

	values.Name = "ako-" + clusterNameSpacedName

	values.Rbac = Rbac{
		PspEnabled:          obj.Spec.ExtraConfigs.Rbac.PspEnabled,
		PspPolicyApiVersion: obj.Spec.ExtraConfigs.Rbac.PspPolicyAPIVersion,
	}
	return values, nil
}
