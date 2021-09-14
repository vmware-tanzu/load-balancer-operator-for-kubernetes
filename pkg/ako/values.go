// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
)

// StripLast strips the last token seperated by the separator
func StripLast(repository, separator string) (string, []string, error) {
	splits := strings.Split(repository, separator)
	if len(splits) == 0 {
		return "", nil, errors.New("cannot strip last, incorrect format")
	}
	return strings.Join(splits[:len(splits)-1], separator), splits, nil
}

// Values defines the structures of an Ako addon secret string data
// this constructs the payload (string data) of the corev1.Secret
type Values struct {
	ImageInfo                     ImageInfo                     `yaml:"imageInfo"`
	LoadBalancerAndIngressService LoadBalancerAndIngressService `yaml:"loadBalancerAndIngressService"`
}

// NewValues creates a new Values
// given AKODeploymentConfig and clusterNameSpacedName
func NewValues(obj *akoov1alpha1.AKODeploymentConfig, clusterNameSpacedName string) (*Values, error) {
	if obj == nil {
		return nil, errors.New("provided AKODeploymentConfig is nil")
	}
	repository, repositorySplits, err := StripLast(obj.Spec.ExtraConfigs.Image.Repository, "/")
	if err != nil {
		return nil, err
	}
	akoSettings := NewAKOSettings(
		clusterNameSpacedName,
		obj.Spec.ExtraConfigs.CniPlugin,
		obj.Spec.ExtraConfigs.DisableStaticRouteSync)
	networkSettings, err := NewNetworkSettings(obj.Spec.DataNetwork, obj.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList)
	if err != nil {
		return nil, err
	}
	controllerSettings := NewControllerSettings(
		obj.Spec.CloudName,
		obj.Spec.Controller,
		obj.Spec.ServiceEngineGroup,
	)
	l7Settings := NewL7Settings(obj.Spec.ExtraConfigs.IngressConfigs)
	l4Settings := DefaultL4Settings()
	nodePortSelector := DefaultNodePortSelector()
	resources := DefaultResources()
	rbac := NewRbac(obj.Spec.ExtraConfigs.Rbac)

	return &Values{
		ImageInfo: ImageInfo{
			ImageRepository: repository,
			ImagePullPolicy: obj.Spec.ExtraConfigs.Image.PullPolicy,
			Images: ImageInfoImages{
				LoadBalancerAndIngressServiceImage: LoadBalancerAndIngressServiceImage{
					ImagePath: repositorySplits[len(repositorySplits)-1],
					Tag:       obj.Spec.ExtraConfigs.Image.Version,
				},
			},
		},
		LoadBalancerAndIngressService: LoadBalancerAndIngressService{
			Name:      "ako-" + clusterNameSpacedName,
			Namespace: akoov1alpha1.AviNamespace,
			Config: Config{
				IsClusterService:      "",
				ReplicaCount:          1,
				AKOSettings:           akoSettings,
				NetworkSettings:       networkSettings,
				L7Settings:            l7Settings,
				L4Settings:            l4Settings,
				ControllerSettings:    controllerSettings,
				NodePortSelector:      nodePortSelector,
				Resources:             resources,
				Rbac:                  rbac,
				PersistentVolumeClaim: obj.Spec.ExtraConfigs.Log.PersistentVolumeClaim,
				MountPath:             obj.Spec.ExtraConfigs.Log.MountPath,
				LogFile:               obj.Spec.ExtraConfigs.Log.LogFile,
			},
		},
	}, nil
}

// NewValuesFromBytes unmarshalls a byte array
// into an instance of Values
func NewValuesFromBytes(data []byte) (*Values, error) {
	var values Values
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return &values, nil
}

// YttYaml converts the AkoAddonSecretData to a Ytt Yaml template string,
// return any unmarshall error occurs
func (v *Values) YttYaml() (string, error) {
	header := akoov1alpha1.TKGDataValueFormatString
	buf, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return header + string(buf), nil
}

// ImageInfo describes the image information for the add-on secret
type ImageInfo struct {
	ImageRepository string          `yaml:"imageRepository"`
	ImagePullPolicy string          `yaml:"imagePullPolicy"`
	Images          ImageInfoImages `yaml:"images"`
}

type ImageInfoImages struct {
	LoadBalancerAndIngressServiceImage LoadBalancerAndIngressServiceImage `yaml:"loadBalancerAndIngressServiceImage"`
}

// LoadBalancerAndIngressServiceImage describes the LoadBalancerAndIngressServiceImage
type LoadBalancerAndIngressServiceImage struct {
	ImagePath string `yaml:"imagePath"`
	Tag       string `yaml:"tag"`
}

// LoadBalancerAndIngressService describes the load balancer and ingress service
type LoadBalancerAndIngressService struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Config    Config `yaml:"config"`
}

// Config consists of different configurations for Values that includes settings of
// AKO, networking, L4, L7, Rbac etc
type Config struct {
	IsClusterService      string             `yaml:"is_cluster_service"`
	ReplicaCount          int                `yaml:"replica_count"`
	AKOSettings           AKOSettings        `yaml:"ako_settings"`
	NetworkSettings       NetworkSettings    `yaml:"network_settings"`
	L7Settings            L7Settings         `yaml:"l7_settings"`
	L4Settings            L4Settings         `yaml:"l4_settings"`
	ControllerSettings    ControllerSettings `yaml:"controller_settings"`
	NodePortSelector      NodePortSelector   `yaml:"nodeport_selector"`
	Resources             Resources          `yaml:"resources"`
	Rbac                  Rbac               `yaml:"rbac"`
	PersistentVolumeClaim string             `yaml:"persistent_volume_claim"`
	MountPath             string             `yaml:"mount_path"`
	LogFile               string             `yaml:"log_file"`
	Avicredentials        Avicredentials     `yaml:"avi_credentials"`
}

// AKOSettings provides the settings for AKO
type AKOSettings struct {
	LogLevel               string `yaml:"log_level"`
	FullSyncFrequency      string `yaml:"full_sync_frequency"`       // This frequency controls how often AKO polls the Avi controller to update itself with cloud configurations.
	ApiServerPort          int    `yaml:"api_server_port"`           // Specify the port for the API server, default is set as 8080 // EmptyAllowed: false
	DeleteConfig           string `yaml:"delete_config"`             // Has to be set to true in configmap if user wants to delete AKO created objects from AVI
	DisableStaticRouteSync string `yaml:"disable_static_route_sync"` // If the POD networks are reachable from the Avi SE, set this knob to true.
	ClusterName            string `yaml:"cluster_name"`              // A unique identifier for the kubernetes cluster, that helps distinguish the objects for this cluster in the avi controller. // MUST-EDIT
	CniPlugin              string `yaml:"cni_plugin"`                // Set the string if your CNI is calico or openshift. enum: calico|canal|flannel|openshift
	SyncNamespace          string `yaml:"sync_namespace"`
}

// DefaultAKOSettings returns the default AKOSettings
func DefaultAKOSettings() AKOSettings {
	return AKOSettings{
		LogLevel:               "INFO",
		ApiServerPort:          8080,
		DeleteConfig:           "false",
		DisableStaticRouteSync: "true",
		FullSyncFrequency:      "1800",
		// CniPlugin: don't set, use default value in AKO
		// SyncNamespace: don't set, use default value in AKO
		// ClusterName: populate in runtime
	}
}

// NewAKOSettings returns a new AKOSettings,
// allow users to set CniPlugin, ClusterName and DisableStaticRouteSync in runtime
func NewAKOSettings(clusterName, cniPlugin string, disableStaticRouteSync bool) (settings AKOSettings) {
	settings = DefaultAKOSettings()
	settings.ClusterName = clusterName
	settings.CniPlugin = cniPlugin
	settings.DisableStaticRouteSync = strconv.FormatBool(disableStaticRouteSync)
	return
}

// NetworkSettings outlines the network settings for virtual services.
type NetworkSettings struct {
	SubnetIP            string                 `yaml:"subnet_ip"`     // Subnet IP of the vip network
	SubnetPrefix        string                 `yaml:"subnet_prefix"` // Subnet Prefix of the vip network
	NetworkName         string                 `yaml:"network_name"`  // Network Name of the vip network
	NodeNetworkList     []v1alpha1.NodeNetwork `yaml:"-"`
	NodeNetworkListJson string                 `yaml:"node_network_list"`
	VIPNetworkList      []map[string]string    `yaml:"-"`
	VIPNetworkListJson  string                 `yaml:"vip_network_list"`
}

// DefaultNetworkSettings returns default NetworkSettings
func DefaultNetworkSettings() NetworkSettings {
	return NetworkSettings{
		// SubnetIP: don't set, populate in runtime
		// SubnetPrefix: don't set, populate in runtime
		// NetworkName: don't set, populate in runtime
		// NodeNetworkList: don't set, use default value in AKO
		// NodeNetworkListJson: don't set, use default value in AKO
	}
}

// NewNetworkSettings returns a new NetworkSettings
// allow user to set NetworkName, SubnetIP, SubnetPrefix, NodeNetworkList and VIPNetworkList at runtime
func NewNetworkSettings(dataNetwork v1alpha1.DataNetwork, nodeNetworkList []v1alpha1.NodeNetwork) (NetworkSettings, error) {
	settings := DefaultNetworkSettings()
	settings.NetworkName = dataNetwork.Name
	ip, ipNet, err := net.ParseCIDR(dataNetwork.CIDR)
	if err != nil {
		return NetworkSettings{}, err
	}
	settings.SubnetIP = ip.String()
	ones, _ := ipNet.Mask.Size()
	settings.SubnetPrefix = strconv.Itoa(ones)

	settings.NodeNetworkList = nodeNetworkList
	settings.VIPNetworkList = []map[string]string{{"networkName": dataNetwork.Name}}

	if len(nodeNetworkList) != 0 {
		jsonBytes, err := json.Marshal(nodeNetworkList)
		if err != nil {
			return NetworkSettings{}, err
		}
		settings.NodeNetworkListJson = string(jsonBytes)
	}
	if len(settings.VIPNetworkList) != 0 {
		jsonBytes, err := json.Marshal(settings.VIPNetworkList)
		if err != nil {
			return NetworkSettings{}, err
		}
		settings.VIPNetworkListJson = string(jsonBytes)
	}
	return settings, nil
}

// L7Settings outlines all the knobs used to control Layer 7 load balancing settings in AKO.
type L7Settings struct {
	DisableIngressClass  bool   `yaml:"disable_ingress_class"`
	DefaultIngController bool   `yaml:"default_ing_controller"`
	L7ShardingScheme     string `yaml:"l7_sharding_scheme"`
	ServiceType          string `yaml:"service_type"`           // enum NodePort|ClusterIP
	ShardVSSize          string `yaml:"shard_vs_size"`          // Use this to control the layer 7 VS numbers. This applies to both secure/insecure VSes but does not apply for passthrough. ENUMs: LARGE, MEDIUM, SMALL
	PassthroughShardSize string `yaml:"pass_through_shardsize"` // Control the passthrough virtualservice numbers using this ENUM. ENUMs: LARGE, MEDIUM, SMALL
}

// DefaultL7Settings returns the default L7Settings
func DefaultL7Settings() L7Settings {
	return L7Settings{
		DefaultIngController: false,
		ServiceType:          "NodePort",
		ShardVSSize:          "SMALL",
		// L7ShardingScheme: don't set, use default value in AKO
		// PassthroughShardSize: don't set, use default value in AKO
	}
}

// NewL7Settings returns a customized L7Settings after parsing the v1alpha1.AKOIngressConfig
// it only modifies ServiceType and ShardVSSize when instructed by the ingressConfig
func NewL7Settings(config v1alpha1.AKOIngressConfig) L7Settings {
	settings := DefaultL7Settings()
	settings.DisableIngressClass = config.DisableIngressClass
	settings.DefaultIngController = config.DefaultIngressController
	if config.ShardVSSize != "" {
		settings.ShardVSSize = config.ShardVSSize
	}
	if config.ServiceType != "" {
		settings.ServiceType = config.ServiceType
	}
	return settings
}

// L4Settings outlines all the knobs  used to control Layer 4 loadbalancing settings in AKO.
type L4Settings struct {
	DefaultDomain string `yaml:"default_domain"` // If multiple sub-domains are configured in the cloud, use this knob to set the default sub-domain to use for L4 VSes.
}

// DefaultL4Settings returns the default L4Settings
func DefaultL4Settings() L4Settings {
	return L4Settings{
		// DefaultDomain: don't set, use default value in AKO
	}
}

// ControllerSettings outlines settings on the Avi controller that affects AKO's functionality.
type ControllerSettings struct {
	ServiceEngineGroupName string `yaml:"service_engine_group_name"` // Name of the ServiceEngine Group.
	ControllerVersion      string `yaml:"controller_version"`        // The controller API version
	CloudName              string `yaml:"cloud_name"`                // The configured cloud name on the Avi controller.
	ControllerIP           string `yaml:"controller_ip"`
}

// DefaultControllerSettings return the default ControllerSettings
func DefaultControllerSettings() ControllerSettings {
	return ControllerSettings{
		// ServiceEngineGroupName: populate in runtime
		// CloudName: populate in runtime
		// ControllerIP: populate in runtime
		// ControllerVersion: don't set, depend on AKO to autodetect,
		// also because we don't consider version skew in Calgary
	}
}

// NewControllerSettings returns a ControllerSettings from default,
// allow setting CloudName, ControllerIP and ServiceEngineGroupName
func NewControllerSettings(cloudName, controllerIP, serviceEngineGroup string) (setting ControllerSettings) {
	setting = DefaultControllerSettings()
	setting.CloudName = cloudName
	setting.ControllerIP = controllerIP
	setting.ServiceEngineGroupName = serviceEngineGroup
	return
}

// NodePortSelector is only applicable if serviceType is NodePort
type NodePortSelector struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// DefaultNodePortSelector returns the default NodePortSelector
func DefaultNodePortSelector() NodePortSelector {
	return NodePortSelector{
		// Key: don't set, use default value in AKO
		// Value: don't set, use default value in AKO
	}
}

type Resources struct {
	Limits   Limits   `yaml:"limits"`
	Requests Requests `yaml:"request"`
}

// DefaultResources returns the default configuration for Resources
func DefaultResources() Resources {
	return Resources{
		Limits: Limits{
			Cpu:    "250m",
			Memory: "300Mi",
		},
		Requests: Requests{
			Cpu:    "100m",
			Memory: "200Mi",
		},
	}
}

type Limits struct {
	Cpu    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

type Requests struct {
	Cpu    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

// Rbac creates the pod security policy if PspEnabled is set to true
type Rbac struct {
	PspEnabled          bool   `yaml:"psp_enabled"`
	PspPolicyApiVersion string `yaml:"psp_policy_api_version"`
}

// NewRbac creates a Rbac from the v1alpha1.AKORbacConfig
func NewRbac(config v1alpha1.AKORbacConfig) Rbac {
	return Rbac{
		PspEnabled:          config.PspEnabled,
		PspPolicyApiVersion: config.PspPolicyAPIVersion,
	}
}

type Avicredentials struct {
	Username                 string `yaml:"username"`
	Password                 string `yaml:"password"`
	CertificateAuthorityData string `yaml:"certificate_authority_data"`
}

func (v Values) GetName() string {
	return "ako-" + strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000), 10)
	//Be aware that during upgrades, templates are re-executed. When a template run generates data that differs from the last run, that will trigger an update of that resource.
}
