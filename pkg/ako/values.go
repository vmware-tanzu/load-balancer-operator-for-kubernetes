// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
)

// Values defines the structures of an Ako addon secret string data
// this constructs the payload (string data) of the corev1.Secret
type Values struct {
	LoadBalancerAndIngressService LoadBalancerAndIngressService `yaml:"loadBalancerAndIngressService"`
}

// NewValues creates a new Values
// given AKODeploymentConfig and clusterNameSpacedName
func NewValues(obj *akoov1alpha1.AKODeploymentConfig, clusterNameSpacedName string) (*Values, error) {
	if obj == nil {
		return nil, errors.New("provided AKODeploymentConfig is nil")
	}
	akoSettings := NewAKOSettings(clusterNameSpacedName, obj)
	networkSettings, err := NewNetworkSettings(obj)
	if err != nil {
		return nil, err
	}
	controllerSettings := NewControllerSettings(
		obj.Spec.CloudName,
		obj.Spec.Controller,
		obj.Spec.ControllerVersion,
		obj.Spec.ServiceEngineGroup,
	)
	l7Settings := NewL7Settings(&obj.Spec.ExtraConfigs.IngressConfigs)
	l4Settings := NewL4Settings(&obj.Spec.ExtraConfigs.L4Configs)
	nodePortSelector := NewNodePortSelector(&obj.Spec.ExtraConfigs.NodePortSelector)
	resources := DefaultResources()
	rbac := NewRbac(obj.Spec.ExtraConfigs.Rbac)

	return &Values{
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

// LoadBalancerAndIngressService describes the load balancer and ingress service
type LoadBalancerAndIngressService struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Config    Config `yaml:"config"`
}

// Config consists of different configurations for Values that includes settings of
// AKO, networking, L4, L7, Rbac etc
type Config struct {
	IsClusterService      string              `yaml:"is_cluster_service"`
	ReplicaCount          int                 `yaml:"replica_count"`
	AKOSettings           *AKOSettings        `yaml:"ako_settings"`
	NetworkSettings       *NetworkSettings    `yaml:"network_settings"`
	L7Settings            *L7Settings         `yaml:"l7_settings"`
	L4Settings            *L4Settings         `yaml:"l4_settings"`
	ControllerSettings    *ControllerSettings `yaml:"controller_settings"`
	NodePortSelector      *NodePortSelector   `yaml:"nodeport_selector"`
	Resources             *Resources          `yaml:"resources"`
	Rbac                  *Rbac               `yaml:"rbac"`
	PersistentVolumeClaim string              `yaml:"persistent_volume_claim"`
	MountPath             string              `yaml:"mount_path"`
	LogFile               string              `yaml:"log_file"`
	Avicredentials        Avicredentials      `yaml:"avi_credentials"`
}

// NamespaceSelector contains label key and value used for namespace migration.
// Same label has to be present on namespace/s which needs migration/sync to AKO
type NamespaceSelector struct {
	LabelKey   string `yaml:"label_key"`
	LabelValue string `yaml:"label_value"`
}

// AKOSettings provides the settings for AKO
type AKOSettings struct {
	LogLevel               string            `yaml:"log_level"`
	FullSyncFrequency      string            `yaml:"full_sync_frequency"`       // This frequency controls how often AKO polls the Avi controller to update itself with cloud configurations.
	ApiServerPort          int               `yaml:"api_server_port"`           // Specify the port for the API server, default is set as 8080 // EmptyAllowed: false
	DeleteConfig           string            `yaml:"delete_config"`             // Has to be set to true in configmap if user wants to delete AKO created objects from AVI
	DisableStaticRouteSync string            `yaml:"disable_static_route_sync"` // If the POD networks are reachable from the Avi SE, set this knob to true.
	ClusterName            string            `yaml:"cluster_name"`              // A unique identifier for the kubernetes cluster, that helps distinguish the objects for this cluster in the avi controller. // MUST-EDIT
	CniPlugin              string            `yaml:"cni_plugin"`                // Set the string if your CNI is calico or openshift. enum: calico|canal|flannel|openshift
	SyncNamespace          string            `yaml:"sync_namespace"`
	EnableEVH              string            `yaml:"enable_EVH"`   // This enables the Enhanced Virtual Hosting Model in Avi Controller for the Virtual Services
	Layer7Only             string            `yaml:"layer_7_only"` // If this flag is switched on, then AKO will only do layer 7 loadbalancing
	ServicesAPI            string            `yaml:"services_api"` // Flag that enables AKO in services API mode. Currently implemented only for L4.
	IstioEnabled           string            `yaml:"istio_enabled"`
	VIPPerNamespace        string            `yaml:"vip_per_namespace"`
	NamespaceSector        NamespaceSelector `yaml:"namespace_selector"`
}

// DefaultAKOSettings returns the default AKOSettings
func DefaultAKOSettings() *AKOSettings {
	return &AKOSettings{
		LogLevel:               "INFO",
		ApiServerPort:          8080,
		DeleteConfig:           "false",
		DisableStaticRouteSync: "true",
		FullSyncFrequency:      "1800",
		NamespaceSector:        NamespaceSelector{},
		// CniPlugin: don't set, use default value in AKO
		// ClusterName: populate in runtime
	}
}

// NewAKOSettings returns a new AKOSettings,
// allow users to set CniPlugin, ClusterName and DisableStaticRouteSync in runtime
func NewAKOSettings(clusterName string, obj *akoov1alpha1.AKODeploymentConfig) (settings *AKOSettings) {
	settings = DefaultAKOSettings()
	settings.ClusterName = clusterName
	if obj.Spec.ExtraConfigs.Log.LogLevel != "" {
		settings.LogLevel = obj.Spec.ExtraConfigs.Log.LogLevel
	}
	if obj.Spec.ExtraConfigs.FullSyncFrequency != "" {
		settings.FullSyncFrequency = obj.Spec.ExtraConfigs.FullSyncFrequency
	}
	if obj.Spec.ExtraConfigs.ApiServerPort != nil {
		settings.ApiServerPort = *obj.Spec.ExtraConfigs.ApiServerPort
	}
	if obj.Spec.ExtraConfigs.DisableStaticRouteSync != nil {
		settings.DisableStaticRouteSync = strconv.FormatBool(*obj.Spec.ExtraConfigs.DisableStaticRouteSync)
	}
	if obj.Spec.ExtraConfigs.CniPlugin != "" {
		settings.CniPlugin = obj.Spec.ExtraConfigs.CniPlugin
	}
	if obj.Spec.ExtraConfigs.EnableEVH != nil {
		settings.EnableEVH = strconv.FormatBool(*obj.Spec.ExtraConfigs.EnableEVH)
	}
	if obj.Spec.ExtraConfigs.Layer7Only != nil {
		settings.EnableEVH = strconv.FormatBool(*obj.Spec.ExtraConfigs.Layer7Only)
	}
	if obj.Spec.ExtraConfigs.ServicesAPI != nil {
		settings.ServicesAPI = strconv.FormatBool(*obj.Spec.ExtraConfigs.ServicesAPI)
	}
	if obj.Spec.ExtraConfigs.IstioEnabled != nil {
		settings.IstioEnabled = strconv.FormatBool(*obj.Spec.ExtraConfigs.IstioEnabled)
	}
	if obj.Spec.ExtraConfigs.VIPPerNamespace != nil {
		settings.VIPPerNamespace = strconv.FormatBool(*obj.Spec.ExtraConfigs.VIPPerNamespace)
	}
	if obj.Spec.ExtraConfigs.NamespaceSelector.LabelKey != "" {
		settings.NamespaceSector.LabelKey = obj.Spec.ExtraConfigs.NamespaceSelector.LabelKey
	}
	if obj.Spec.ExtraConfigs.NamespaceSelector.LabelValue != "" {
		settings.NamespaceSector.LabelValue = obj.Spec.ExtraConfigs.NamespaceSelector.LabelValue
	}
	return
}

// NetworkSettings outlines the network settings for virtual services.
type NetworkSettings struct {
	SubnetIP            string                 `yaml:"subnet_ip"`     // Subnet IP of the vip network
	SubnetPrefix        string                 `yaml:"subnet_prefix"` // Subnet Prefix of the vip network
	NetworkName         string                 `yaml:"network_name"`  // Network Name of the vip network
	NodeNetworkList     []v1alpha1.NodeNetwork `yaml:"-"`             // This list of network and cidrs are used in pool placement network for vcenter cloud.
	NodeNetworkListJson string                 `yaml:"node_network_list"`
	VIPNetworkList      []map[string]string    `yaml:"-"` // Network information of the VIP network. Multiple networks allowed only for AWS Cloud.
	VIPNetworkListJson  string                 `yaml:"vip_network_list"`
	EnableRHI           string                 `yaml:"enable_rhi"` // This is a cluster wide setting for BGP peering.
	NsxtT1LR            string                 `yaml:"nsxt_t1_lr"`
	BGPPeerLabels       []string               `yaml:"-"` // Select BGP peers using bgpPeerLabels, for selective VsVip advertisement.
	BGPPeerLabelsJson   string                 `yaml:"bgp_peer_labels"`
}

// DefaultNetworkSettings returns default NetworkSettings
func DefaultNetworkSettings() *NetworkSettings {
	return &NetworkSettings{
		// SubnetIP: don't set, populate in runtime
		// SubnetPrefix: don't set, populate in runtime
		// NetworkName: don't set, populate in runtime
		// NodeNetworkList: don't set, use default value in AKO
		// NodeNetworkListJson: don't set, use default value in AKO
	}
}

// NewNetworkSettings returns a new NetworkSettings
// allow user to set NetworkName, SubnetIP, SubnetPrefix, NodeNetworkList and VIPNetworkList at runtime
func NewNetworkSettings(obj *akoov1alpha1.AKODeploymentConfig) (*NetworkSettings, error) {
	settings := DefaultNetworkSettings()
	settings.NetworkName = obj.Spec.DataNetwork.Name
	ip, ipNet, err := net.ParseCIDR(obj.Spec.DataNetwork.CIDR)
	if err != nil {
		return &NetworkSettings{}, err
	}
	settings.SubnetIP = ip.String()
	ones, _ := ipNet.Mask.Size()
	settings.SubnetPrefix = strconv.Itoa(ones)

	settings.NodeNetworkList = obj.Spec.ExtraConfigs.IngressConfigs.NodeNetworkList
	settings.VIPNetworkList = []map[string]string{{"networkName": obj.Spec.DataNetwork.Name}}

	if len(settings.NodeNetworkList) != 0 {
		jsonBytes, err := json.Marshal(settings.NodeNetworkList)
		if err != nil {
			return &NetworkSettings{}, err
		}
		settings.NodeNetworkListJson = string(jsonBytes)
	}
	if len(settings.VIPNetworkList) != 0 {
		jsonBytes, err := json.Marshal(settings.VIPNetworkList)
		if err != nil {
			return &NetworkSettings{}, err
		}
		settings.VIPNetworkListJson = string(jsonBytes)
	}
	if obj.Spec.ExtraConfigs.NetworksConfig.EnableRHI != nil {
		settings.EnableRHI = strconv.FormatBool(*obj.Spec.ExtraConfigs.NetworksConfig.EnableRHI)
	}
	settings.NsxtT1LR = obj.Spec.ExtraConfigs.NetworksConfig.NsxtT1LR
	settings.BGPPeerLabels = obj.Spec.ExtraConfigs.NetworksConfig.BGPPeerLabels
	if len(settings.BGPPeerLabels) != 0 {
		jsonBytes, err := json.Marshal(settings.BGPPeerLabels)
		if err != nil {
			return &NetworkSettings{}, err
		}
		settings.BGPPeerLabelsJson = string(jsonBytes)
	}
	return settings, nil
}

// L7Settings outlines all the knobs used to control Layer 7 load balancing settings in AKO.
type L7Settings struct {
	DisableIngressClass  bool   `yaml:"disable_ingress_class"`
	DefaultIngController bool   `yaml:"default_ing_controller"`
	L7ShardingScheme     string `yaml:"l7_sharding_scheme"`
	ServiceType          string `yaml:"service_type"`           // enum NodePort|ClusterIP|NodePortLocal
	ShardVSSize          string `yaml:"shard_vs_size"`          // Use this to control the layer 7 VS numbers. This applies to both secure/insecure VSes but does not apply for passthrough. ENUMs: LARGE, MEDIUM, SMALL
	PassthroughShardSize string `yaml:"pass_through_shardsize"` // Control the passthrough virtualservice numbers using this ENUM. ENUMs: LARGE, MEDIUM, SMALL
	NoPGForSNI           bool   `yaml:"no_pg_for_SNI"`
}

// DefaultL7Settings returns the default L7Settings
func DefaultL7Settings() *L7Settings {
	return &L7Settings{
		DefaultIngController: false,
		ServiceType:          "NodePort",
		ShardVSSize:          "SMALL",
		// L7ShardingScheme: don't set, use default value in AKO
		// PassthroughShardSize: don't set, use default value in AKO
	}
}

// NewL7Settings returns a customized L7Settings after parsing the v1alpha1.AKOIngressConfig
// it only modifies ServiceType and ShardVSSize when instructed by the ingressConfig
func NewL7Settings(config *akoov1alpha1.AKOIngressConfig) *L7Settings {
	settings := DefaultL7Settings()
	settings.DisableIngressClass = config.DisableIngressClass
	settings.DefaultIngController = config.DefaultIngressController
	if config.ShardVSSize != "" {
		settings.ShardVSSize = config.ShardVSSize
	}
	if config.ServiceType != "" {
		settings.ServiceType = config.ServiceType
	}
	if config.PassthroughShardSize != "" {
		settings.PassthroughShardSize = config.PassthroughShardSize
	}
	if config.NoPGForSNI != nil {
		settings.NoPGForSNI = *config.NoPGForSNI
	}
	return settings
}

// L4Settings outlines all the knobs  used to control Layer 4 loadbalancing settings in AKO.
type L4Settings struct {
	AdvancedL4    string `yaml:"advanced_l4"`    // Use this knob to control the settings for the services API usage.
	DefaultDomain string `yaml:"default_domain"` // If multiple sub-domains are configured in the cloud, use this knob to set the default sub-domain to use for L4 VSes.
	AutoFQDN      string `yaml:"auto_fqdn"`      // ENUM: default(<svc>.<ns>.<subdomain>), flat (<svc>-<ns>.<subdomain>), "disabled"
}

// DefaultL4Settings returns the default L4Settings
func DefaultL4Settings() *L4Settings {
	return &L4Settings{
		// DefaultDomain: don't set, use default value in AKO
	}
}

// NewL4Settings returns a customized L4Settings after parsing the v1alpha1.AKOL4Config
func NewL4Settings(config *akoov1alpha1.AKOL4Config) *L4Settings {
	settings := DefaultL4Settings()
	if config.AdvancedL4 != nil {
		settings.AdvancedL4 = strconv.FormatBool(*config.AdvancedL4)
	}
	if config.DefaultDomain != "" {
		settings.DefaultDomain = config.DefaultDomain
	}
	if config.AutoFQDN != "" {
		settings.AutoFQDN = config.AutoFQDN
	}
	return settings
}

// ControllerSettings outlines settings on the Avi controller that affects AKO's functionality.
type ControllerSettings struct {
	ServiceEngineGroupName string `yaml:"service_engine_group_name"` // Name of the ServiceEngine Group.
	ControllerVersion      string `yaml:"controller_version"`        // The controller API version
	CloudName              string `yaml:"cloud_name"`                // The configured cloud name on the Avi controller.
	ControllerIP           string `yaml:"controller_ip"`
}

// DefaultControllerSettings return the default ControllerSettings
func DefaultControllerSettings() *ControllerSettings {
	return &ControllerSettings{
		// ServiceEngineGroupName: populate in runtime
		// CloudName: populate in runtime
		// ControllerIP: populate in runtime
		// ControllerVersion: don't set, depend on AKO to autodetect,
		// also because we don't consider version skew in Calgary
	}
}

// NewControllerSettings returns a ControllerSettings from default,
// allow setting CloudName, ControllerIP, ControllerVersion and ServiceEngineGroupName
func NewControllerSettings(cloudName, controllerIP, controllerVersion, serviceEngineGroup string) (setting *ControllerSettings) {
	setting = DefaultControllerSettings()
	setting.CloudName = cloudName
	setting.ControllerIP = controllerIP
	setting.ServiceEngineGroupName = serviceEngineGroup
	if controllerVersion != "" {
		setting.ControllerVersion = controllerVersion
	}
	return
}

// NodePortSelector is only applicable if serviceType is NodePort
type NodePortSelector struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// DefaultNodePortSelector returns the default NodePortSelector
func DefaultNodePortSelector() *NodePortSelector {
	return &NodePortSelector{
		// Key: don't set, use default value in AKO
		// Value: don't set, use default value in AKO
	}
}

// NewNodePortSelector returns the NodePortSelector defined in AKODeploymentConfig
func NewNodePortSelector(nodePortSelector *akoov1alpha1.NodePortSelector) *NodePortSelector {
	selector := DefaultNodePortSelector()
	if nodePortSelector.Key != "" {
		selector.Key = nodePortSelector.Key
	}
	if nodePortSelector.Value != "" {
		selector.Key = nodePortSelector.Value
	}
	return selector
}

type Resources struct {
	Limits   Limits   `yaml:"limits"`
	Requests Requests `yaml:"request"`
}

// DefaultResources returns the default configuration for Resources
func DefaultResources() *Resources {
	return &Resources{
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
func NewRbac(config v1alpha1.AKORbacConfig) *Rbac {
	return &Rbac{
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
