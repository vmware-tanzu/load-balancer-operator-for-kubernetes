package controllers

import (
	rand "math/rand"
	"strconv"
	"strings"
	"time"
)

type Values struct {
	Name                string
	NameOverride        string
	Namespace           string
	AppVersion          string
	ChartName           string
	PsppolicyApiVersion string
	IsClusterService    bool

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
	Service               Service
	PersistentVolumeClaim string
	MountPath             string
	LogFile               string
	//nameOverride          string
}

type Image struct {
	Repository string
	PullPolicy string // enum: INFO|DEBUG|WARN|ERROR
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
	NodeNetworkList     []NodeNetwork
	NodeNetworkListJson string
}

type NodeNetwork struct {
	NetworkName string
	Cidrs       []string
}

// This section outlines all the knobs  used to control Layer 7 loadbalancing settings in AKO.
type L7Settings struct {
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
	PspEnable bool
}

type Avicredentials struct {
	Username string
	Password string
}

type Service struct {
	Type string
	Port int
}

func (v Values) GetName(nameOverride string) string {
	if nameOverride != "" {
		if len(nameOverride) > 63 {
			nameOverride = nameOverride[0:62]
		}
		return strings.TrimSuffix(nameOverride, "-")
	} else {
		return "ako-" + strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(10000000000), 10)
		//Be aware that during upgrades, templates are re-executed. When a template run generates data that differs from the last run, that will trigger an update of that resource.
	}
}
