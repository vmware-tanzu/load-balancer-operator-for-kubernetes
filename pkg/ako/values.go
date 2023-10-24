// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako

import (
	"encoding/json"
	"errors"
	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/utils"
	"math/rand"
	"net"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	akoov1alpha1 "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/api/v1alpha1"
	ako_operator "github.com/vmware-tanzu/load-balancer-operator-for-kubernetes/pkg/ako-operator"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
		obj.Spec.Tenant.Name,
	)
	l7Settings := NewL7Settings(&obj.Spec.ExtraConfigs.IngressConfigs)
	l4Settings := NewL4Settings(&obj.Spec.ExtraConfigs.L4Configs)
	nodePortSelector := NewNodePortSelector(&obj.Spec.ExtraConfigs.NodePortSelector)
	rbac := NewRbac(obj.Spec.ExtraConfigs.Rbac)
	featureGates := NewFeatureGates(obj.Spec.ExtraConfigs.FeatureGates)

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
				Rbac:                  rbac,
				PersistentVolumeClaim: obj.Spec.ExtraConfigs.Log.PersistentVolumeClaim,
				MountPath:             obj.Spec.ExtraConfigs.Log.MountPath,
				LogFile:               obj.Spec.ExtraConfigs.Log.LogFile,
				FeatureGates:          featureGates,
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
func (v *Values) YttYaml(cluster *clusterv1.Cluster) (string, error) {
	buf, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	if ako_operator.IsClusterClassBasedCluster(cluster) {
		return string(buf), nil
	} else {
		header := akoov1alpha1.TKGDataValueFormatString
		return header + string(buf), nil
	}
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
	TkgClusterRole        string              `yaml:"tkg_cluster_role"`
	IsClusterService      string              `yaml:"is_cluster_service"`
	ReplicaCount          int                 `yaml:"replica_count"`
	AKOSettings           *AKOSettings        `yaml:"ako_settings"`
	NetworkSettings       *NetworkSettings    `yaml:"network_settings"`
	L7Settings            *L7Settings         `yaml:"l7_settings"`
	L4Settings            *L4Settings         `yaml:"l4_settings"`
	ControllerSettings    *ControllerSettings `yaml:"controller_settings"`
	NodePortSelector      *NodePortSelector   `yaml:"nodeport_selector"`
	Rbac                  *Rbac               `yaml:"rbac"`
	PersistentVolumeClaim string              `yaml:"persistent_volume_claim"`
	MountPath             string              `yaml:"mount_path"`
	LogFile               string              `yaml:"log_file"`
	Avicredentials        Avicredentials      `yaml:"avi_credentials"`
	FeatureGates          *FeatureGates       `yaml:"feature_gates"`
}

// NamespaceSelector contains label key and value used for namespace migration.
// Same label has to be present on namespace/s which needs migration/sync to AKO
type NamespaceSelector struct {
	LabelKey   string `yaml:"label_key"`
	LabelValue string `yaml:"label_value"`
}

// AKOSettings provides the settings for AKO
type AKOSettings struct {
	PrimaryInstance          string            `yaml:"primary_instance"` // Defines AKO instance is primary or not. Value `true` indicates that AKO instance is primary.
	LogLevel                 string            `yaml:"log_level"`
	FullSyncFrequency        string            `yaml:"full_sync_frequency"`       // This frequency controls how often AKO polls the Avi controller to update itself with cloud configurations.
	ApiServerPort            int               `yaml:"api_server_port"`           // Specify the port for the API server, default is set as 8080 // EmptyAllowed: false
	DeleteConfig             string            `yaml:"delete_config"`             // Has to be set to true in configmap if user wants to delete AKO created objects from AVI
	DisableStaticRouteSync   string            `yaml:"disable_static_route_sync"` // If the POD networks are reachable from the Avi SE, set this knob to true.
	ClusterName              string            `yaml:"cluster_name"`              // A unique identifier for the kubernetes cluster, that helps distinguish the objects for this cluster in the avi controller. // MUST-EDIT
	CniPlugin                string            `yaml:"cni_plugin"`                // Set the string if your CNI is calico or openshift. enum: antrea|calico|canal|flannel|openshift
	SyncNamespace            string            `yaml:"sync_namespace"`
	EnableEVH                string            `yaml:"enable_EVH"`   // This enables the Enhanced Virtual Hosting Model in Avi Controller for the Virtual Services
	Layer7Only               string            `yaml:"layer_7_only"` // If this flag is switched on, then AKO will only do layer 7 loadbalancing
	ServicesAPI              string            `yaml:"services_api"` // Flag that enables AKO in services API mode. Currently implemented only for L4.
	VIPPerNamespace          string            `yaml:"vip_per_namespace"`
	NamespaceSector          NamespaceSelector `yaml:"namespace_selector"`
	EnableEvents             string            `yaml:"enable_events"` // Enables/disables Event broadcasting via AKO
	IstioEnabled             string            `yaml:"istio_enabled"`
	BlockedNamespaceList     []string          `yaml:"-"`
	BlockedNamespaceListJson string            `yaml:"blocked_namespace_list"`
	IpFamily                 string            `yaml:"ip_family"`
	UseDefaultSecretsOnly    string            `yaml:"use_default_secrets_only"`
}

type CNI string

const (
	Antrea    CNI = "antrea"
	Calico    CNI = "calico"
	Canal     CNI = "canal"
	Flannel   CNI = "flannel"
	Openshift CNI = "openshift"
)

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
	settings.BlockedNamespaceList = obj.Spec.ExtraConfigs.BlockedNamespaceList
	if obj.Spec.ExtraConfigs.PrimaryInstance != nil {
		settings.PrimaryInstance = strconv.FormatBool(*obj.Spec.ExtraConfigs.PrimaryInstance)
	}
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
	if obj.Spec.ExtraConfigs.EnableEvents != nil {
		settings.EnableEVH = strconv.FormatBool(*obj.Spec.ExtraConfigs.EnableEvents)
	}
	if obj.Spec.ExtraConfigs.ServicesAPI != nil {
		settings.ServicesAPI = strconv.FormatBool(*obj.Spec.ExtraConfigs.ServicesAPI)
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
	if obj.Spec.ExtraConfigs.IstioEnabled != nil {
		settings.IstioEnabled = strconv.FormatBool(*obj.Spec.ExtraConfigs.IstioEnabled)
	}
	if obj.Spec.ExtraConfigs.UseDefaultSecretsOnly != nil {
		settings.UseDefaultSecretsOnly = strconv.FormatBool(*obj.Spec.ExtraConfigs.UseDefaultSecretsOnly)
	}
	if obj.Spec.ExtraConfigs.IpFamily != "" {
		settings.IpFamily = obj.Spec.ExtraConfigs.IpFamily
	}
	if len(settings.BlockedNamespaceList) != 0 {
		// json marshal []string can't throw error
		jsonBytes, _ := json.Marshal(settings.BlockedNamespaceList)
		settings.BlockedNamespaceListJson = string(jsonBytes)
	}
	return
}

// NetworkSettings outlines the network settings for virtual services.
type NetworkSettings struct {
	SubnetIP                string                 `yaml:"subnet_ip"`                  // Subnet IP of the vip network
	SubnetPrefix            string                 `yaml:"subnet_prefix"`              // Subnet Prefix of the vip network
	NetworkName             string                 `yaml:"network_name"`               // Network Name of the vip network
	ControlPlaneNetworkName string                 `yaml:"control_plane_network_name"` // Control Plane Network Name of the control plane vip network
	ControlPlaneNetworkCIDR string                 `yaml:"control_plane_network_cidr"` // Control Plane Network Cidr of the control plane vip network
	NodeNetworkList         []v1alpha1.NodeNetwork `yaml:"-"`                          // This list of network and cidrs are used in pool placement network for vcenter cloud.
	NodeNetworkListJson     string                 `yaml:"node_network_list"`
	VIPNetworkList          []v1alpha1.VIPNetwork  `yaml:"-"` // Network information of the VIP network. Multiple networks allowed only for AWS Cloud.
	VIPNetworkListJson      string                 `yaml:"vip_network_list"`
	EnableRHI               string                 `yaml:"enable_rhi"` // This is a cluster wide setting for BGP peering.
	NsxtT1LR                string                 `yaml:"nsxt_t1_lr"`
	BGPPeerLabels           []string               `yaml:"-"` // Select BGP peers using bgpPeerLabels, for selective VsVip advertisement.
	BGPPeerLabelsJson       string                 `yaml:"bgp_peer_labels"`
}

// DefaultNetworkSettings returns default NetworkSettings
func DefaultNetworkSettings() *NetworkSettings {
	return &NetworkSettings{
		// SubnetIP: don't set, populate in runtime
		// SubnetPrefix: don't set, populate in runtime
		// NetworkName: don't set, populate in runtime
		// ControlPlaneNetworkName: don't set, populate in runtime
		// ControlPlaneNetworkCIDR: don't set, populate in runtime
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
	//V6CIDR will enable the VS networks to use ipv6
	if utils.GetIPFamilyFromCidr(obj.Spec.DataNetwork.CIDR) == "V6" {
		settings.VIPNetworkList = []v1alpha1.VIPNetwork{{NetworkName: obj.Spec.DataNetwork.Name, V6CIDR: obj.Spec.DataNetwork.CIDR}}
	} else {
		settings.VIPNetworkList = []v1alpha1.VIPNetwork{{NetworkName: obj.Spec.DataNetwork.Name, CIDR: obj.Spec.DataNetwork.CIDR}}
	}

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

	if obj.Spec.ControlPlaneNetwork.Name != "" {
		settings.ControlPlaneNetworkName = obj.Spec.ControlPlaneNetwork.Name
		settings.ControlPlaneNetworkCIDR = obj.Spec.ControlPlaneNetwork.CIDR
	} else {
		settings.ControlPlaneNetworkName = obj.Spec.DataNetwork.Name
		settings.ControlPlaneNetworkCIDR = obj.Spec.DataNetwork.CIDR
	}

	if obj.Spec.ExtraConfigs.NetworksConfig.EnableRHI != nil {
		settings.EnableRHI = strconv.FormatBool(*obj.Spec.ExtraConfigs.NetworksConfig.EnableRHI)
	}
	if obj.Spec.ExtraConfigs.NetworksConfig.NsxtT1LR != "" {
		settings.NsxtT1LR = obj.Spec.ExtraConfigs.NetworksConfig.NsxtT1LR
	}
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
	EnableMCI            string `yaml:"enable_MCI"` // Enabling this flag would tell AKO to start processing multi-cluster ingress objects.
}

type ServiceType string

const (
	NodePort      ServiceType = "NodePort"
	ClusterIP     ServiceType = "ClusterIP"
	NodePortLocal ServiceType = "NodePortLocal"
)

// DefaultL7Settings returns the default L7Settings
func DefaultL7Settings() *L7Settings {
	return &L7Settings{
		ServiceType: string(NodePort),
		// DefaultIngController  don't set, populate in runtime
		// ShardVSSize:          don't set, populate in runtime
		// L7ShardingScheme: 	 don't set, populate in runtime
		// PassthroughShardSize: don't set, populate in runtime
	}
}

// NewL7Settings returns a customized L7Settings after parsing the v1alpha1.AKOIngressConfig
// it only modifies ServiceType and ShardVSSize when instructed by the ingressConfig
func NewL7Settings(config *akoov1alpha1.AKOIngressConfig) *L7Settings {
	settings := DefaultL7Settings()
	if config.DisableIngressClass != nil {
		settings.DisableIngressClass = *config.DisableIngressClass
	}
	if config.DefaultIngressController != nil {
		settings.DefaultIngController = *config.DefaultIngressController
	}
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
	if config.EnableMCI != nil {
		settings.EnableMCI = strconv.FormatBool(*config.EnableMCI)
	}
	return settings
}

// L4Settings outlines all the knobs  used to control Layer 4 loadbalancing settings in AKO.
type L4Settings struct {
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
	TenantName             string `yaml:"tenant_name"`
}

// DefaultControllerSettings return the default ControllerSettings
func DefaultControllerSettings() *ControllerSettings {
	return &ControllerSettings{
		// set controller version to the default one
		// ControllerVersion: populate in runtime,
		// ServiceEngineGroupName: populate in runtime
		// CloudName: populate in runtime
		// ControllerIP: populate in runtime
		// TenantName: populate in runtime
	}
}

// NewControllerSettings returns a ControllerSettings from default,
// allow setting CloudName, ControllerIP, ControllerVersion and ServiceEngineGroupName
func NewControllerSettings(cloudName, controllerIP, controllerVersion, serviceEngineGroup, tenantName string) (setting *ControllerSettings) {
	setting = DefaultControllerSettings()
	setting.CloudName = cloudName
	setting.ControllerIP = controllerIP
	setting.ServiceEngineGroupName = serviceEngineGroup
	setting.TenantName = tenantName
	setting.ControllerVersion = controllerVersion
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

// Rbac creates the pod security policy if PspEnabled is set to true
type Rbac struct {
	PspEnabled          bool   `yaml:"psp_enabled"`
	PspPolicyApiVersion string `yaml:"psp_policy_api_version"`
}

// NewRbac creates a Rbac from the v1alpha1.AKORbacConfig
func NewRbac(config v1alpha1.AKORbacConfig) *Rbac {
	pspEnabled := false
	if config.PspEnabled != nil {
		pspEnabled = *config.PspEnabled
	}
	return &Rbac{
		PspEnabled:          pspEnabled,
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

// FeatureGates describes the configuration for AKO features
type FeatureGates struct {
	GatewayAPI string `yaml:"gateway_api"`
}

func NewFeatureGates(config v1alpha1.FeatureGates) *FeatureGates {
	gatewayAPIEnabled := "false"
	if config.GatewayAPI != "" {
		gatewayAPIEnabled = config.GatewayAPI
	}
	return &FeatureGates{
		GatewayAPI: gatewayAPIEnabled,
	}
}
