// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

// AKODeploymentConfigSpec defines the desired state of an AKODeploymentConfig
// AKODeploymentConfig describes the shared configurations for AKO deployments across a set
// of Clusters.
type AKODeploymentConfigSpec struct {
	// CloudName speficies the AVI Cloud AKO will be deployed with
	CloudName string `json:"cloudName"`

	// Controller is the AVI Controller endpoint to which AKO talks to
	// provision Load Balancer resources
	// The format is [scheme://]address[:port]
	// * scheme                     http or https, defaults to https if not
	//                              specified
	// * address                    IP address of the AVI Controller
	//                              specified
	// * port                       if not specified, use default port for
	//                              the corresponding scheme
	Controller string `json:"controller"`

	// ServiceEngineGroup is the group name of Service Engine that's to be used by the set
	// of AKO Deployments
	ServiceEngineGroup string `json:"serviceEngineGroup"`

	// Label selector for Clusters. The Clusters that are
	// selected by this will be the ones affected by this
	// AKODeploymentConfig.
	// It must match the Cluster labels. This field is immutable.
	// +optional
	ClusterSelector metav1.LabelSelector `json:"clusterSelector,omitempty"`

	// WorkloadCredentialRef points to a Secret resource which includes the username
	// and password to access and configure the Avi Controller.
	//
	// * username                   Username used with basic authentication for
	//                              the Avi REST API
	// * password                   Password used with basic authentication for
	//                              the Avi REST API
	//
	// This field is optional. When it's not specified, username/password
	// will be automatically generated for each Cluster and Tenant needs to
	// be non-nil in this case.
	// +optional
	WorkloadCredentialRef SecretReference `json:"workloadCredentialRef,omitempty"`

	// AdminCredentialRef points to a Secret resource which includes the username
	// and password to access and configure the Avi Controller.
	//
	// * username                   Username used with basic authentication for
	//                              the Avi REST API
	// * password                   Password used with basic authentication for
	//                              the Avi REST API
	//
	// This credential needs to be bound with admin tenant and will be used
	// by AKO Operator to automate configurations and operations.
	AdminCredentialRef SecretReference `json:"adminCredentialRef"`

	// CertificateAuthorityRef points to a Secret resource that includes the
	// AVI Controller's CA
	//
	// * certificateAuthorityData   PEM-encoded certificate authority
	//                              certificates
	//
	CertificateAuthorityRef SecretReference `json:"certificateAuthorityRef"`

	// The AVI tenant for the current AKODeploymentConfig
	// This field is optional.
	// +optional
	Tenant AVITenant `json:"tenant,omitempty"`

	// DataNetworks describes the Data Networks the AKO will be deployed
	// with.
	// This field is immutable.
	DataNetwork DataNetwork `json:"dataNetwork"`

	// ControlPlaneNetwork describes the control plane network of the clusters selected by an akoDeploymentConfig
	//
	// +optional
	ControlPlaneNetwork ControlPlaneNetwork `json:"controlPlaneNetwork"`

	// ExtraConfigs contains extra configurations for AKO Deployment
	//
	// +optional
	ExtraConfigs ExtraConfigs `json:"extraConfigs,omitempty"`
}

// ExtraConfigs contains extra configurations for AKO Deployment
type ExtraConfigs struct {
	// Log specifies the configuration for AKO logging
	// +optional
	Log AKOLogConfig `json:"log,omitempty"`

	// FullSyncFrequency controls how often AKO polls the Avi controller to update itself
	// with cloud configurations. Default value is 1800
	// +optional
	FullSyncFrequency string `json:"fullSyncFrequency,omitempty"`

	// ApiServerPort specifies Internal port for AKO's API server for the liveness probe of the AKO pod
	// default port is 8080
	// +optional
	ApiServerPort *int `json:"apiServerPort,omitempty"`

	// DisableStaticRouteSync describes ako should sync static routing or not.
	// If the POD networks are reachable from the Avi SE, this should be to true.
	// Otherwise, it should be false.
	// It would be true by default.
	// +optional
	DisableStaticRouteSync *bool `json:"disableStaticRouteSync,omitempty"`

	// CniPlugin describes which cni plugin cluster is using.
	// default value is antrea, set this string if cluster cni is other type.
	// AKO supported CNI: antrea|calico|canal|flannel|openshift|ncp
	// +kubebuilder:validation:Enum=antrea;calico;canal;flannel;openshift;ncp
	// +optional
	CniPlugin string `json:"cniPlugin,omitempty"`

	// EnableEVH specifies if you want to enable the Enhanced Virtual Hosting Model
	// in Avi Controller for the Virtual Services, default value is false
	// +optional
	EnableEVH *bool `json:"enableEVH,omitempty"`

	// Layer7Only specifies if you want AKO only to do layer 7 load balancing.
	// default value is false
	// +optional
	Layer7Only *bool `json:"layer7Only,omitempty"`

	// NameSpaceSelector contains label key and value used for namespace migration.
	// Same label has to be present on namespace/s which needs migration/sync to AKO
	// +optional
	NamespaceSelector NamespaceSelector `json:"namespaceSelector,omitempty"`

	// ServicesAPI specifies if enables AKO in services API mode: https://kubernetes-sigs.github.io/service-apis/.
	// Currently, implemented only for L4. This flag uses the upstream GA APIs which are not backward compatible
	// with the advancedL4 APIs which uses a fork and a version of v1alpha1pre1
	// default value is false
	// +optional
	ServicesAPI *bool `json:"servicesAPI,omitempty"`

	// This flag indicates to AKO that it should listen on Istio resources.
	// default value is false
	// +optional
	IstioEnabled *bool `json:"istioEnabled,omitempty"`

	// Enabling this flag would tell AKO to create Parent VS per Namespace in EVH mode
	// default value is false
	// +optional
	VIPPerNamespace *bool `json:"vipPerNamespace,omitempty"`

	// NetworksConfig specifies the network configurations for virtual services.
	// +optional
	NetworksConfig NetworksConfig `json:"networksConfig,omitempty"`

	// IngressConfigs specifies ingress configuration for ako
	// +optional
	IngressConfigs AKOIngressConfig `json:"ingress,omitempty"`

	// IngressConfigs specifies L4 load balancer configuration for ako
	// +optional
	L4Configs AKOL4Config `json:"l4Config,omitempty"`

	// NodePortSelector only applicable if serviceType is NodePort
	// +optional
	NodePortSelector NodePortSelector `json:"nodePortSelector,omitempty"`

	// Rbac specifies the configuration for AKO Rbac
	// +optional
	Rbac AKORbacConfig `json:"rbac,omitempty"`
}

// NameSpaceSelector contains label key and value used for namespace migration
type NamespaceSelector struct {
	LabelKey   string `json:"labelKey,omitempty"`
	LabelValue string `json:"labelValue,omitempty"`
}

type NetworksConfig struct {
	// EnableRHI specifies cluster wide setting for BGP peering.
	// default value is false
	// +optional
	EnableRHI *bool `json:"enableRHI,omitempty"`

	// BGPPeerLabels specifies BGP peers, this is used for selective VsVip advertisement.
	// +optional
	BGPPeerLabels []string `json:"bgpPeerLabels,omitempty"`

	// T1 Logical Segment mapping for backend network. Only applies to NSX-T cloud.
	// +optional
	NsxtT1LR string `json:"nsxtT1LR,omitempty"`

	// VipNetworkList specifies Network information of the VIP network.
	// Multiple networks allowed only for AWS Cloud.
	// default will be the networks specified in Data Networks
	// +optional
	// VipNetworkList []NodeNetwork `json:"vipNetworkList,omitempty"`
}

// AKOIngressConfig contains ingress configurations for AKO Deployment
type AKOIngressConfig struct {
	// DisableIngressClass will prevent AKO Operator to install AKO
	// IngressClass into workload clusters for old version of K8s
	//
	// +optional
	DisableIngressClass bool `json:"disableIngressClass,omitempty"`

	// DefaultIngressController bool describes ako is the default
	// ingress controller to use
	//
	// +optional
	DefaultIngressController bool `json:"defaultIngressController,omitempty"`

	// ServiceType string describes ingress methods for a service
	// Valid value should be NodePort, ClusterIP and NodePortLocal
	// +kubebuilder:validation:Enum=NodePort;ClusterIP;NodePortLocal
	// +optional
	ServiceType string `json:"serviceType,omitempty"`

	// ShardVSSize describes ingress shared virtual service size
	// Valid value should be SMALL, MEDIUM, LARGE or DEDICATED, default value is SMALL
	// +kubebuilder:validation:Enum=SMALL;MEDIUM;LARGE;DEDICATED
	// +optional
	ShardVSSize string `json:"shardVSSize,omitempty"`

	// PassthroughShardSize controls the passthrough virtualservice numbers
	// Valid value should be SMALL, MEDIUM or LARGE, default value is SMALL
	// +kubebuilder:validation:Enum=SMALL;MEDIUM;LARGE
	// +optional
	PassthroughShardSize string `json:"passthroughShardSize,omitempty"`

	// NodeNetworkList describes the details of network and CIDRs
	// are used in pool placement network for vcenter cloud. Node Network details
	// are not needed when in NodePort mode / static routes are disabled / non vcenter clouds.
	// +optional
	NodeNetworkList []NodeNetwork `json:"nodeNetworkList,omitempty"`

	// NoPGForSNI describes if you want to get rid of poolgroups from SNI VSes.
	// Do not use this flag, if you don't want http caching, default value is false.
	// +optional
	NoPGForSNI *bool `json:"noPGForSNI,omitempty"`
}

// AKOL4Config contains L4 load balancer configurations for AKO Deployment
type AKOL4Config struct {
	// AdvancedL4 controls the settings for the services API usage.
	// default to not using services APIs: https://github.com/kubernetes-sigs/service-apis
	// +optional
	AdvancedL4 *bool `json:"advancedL4,omitempty"`

	// DefaultDomain controls the default sub-domain to use for L4 VSes when multiple sub-domains
	// are configured in the cloud.
	// +optional
	DefaultDomain string `json:"defaultDomain,omitempty"`

	// AutoFQDN controls the FQDN generation.
	// Valid value should be default(<svc>.<ns>.<subdomain>), flat (<svc>-<ns>.<subdomain>) or disabled,
	// +kubebuilder:validation:Enum=default;flat;disabled
	// +optional
	AutoFQDN string `json:"autoFQDN,omitempty"`
}

// NodePortSelector is only applicable if serviceType is NodePort
type NodePortSelector struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type NodeNetwork struct {
	// NetworkName is the name of this network
	// +optional
	NetworkName string `json:"networkName,omitempty"`
	// Cidrs represents all the IP CIDRs in this network
	// +optional
	Cidrs []string `json:"cidrs,omitempty"`
}

type AKOLogConfig struct {
	// LogLevel specifies the AKO pod log level
	// Valid value should be INFO, DEBUG, WARN or ERROR, default value is INFO
	// +kubebuilder:validation:Enum=INFO;DEBUG;WARN;ERROR
	// +optional
	LogLevel string `json:"logLevel,omitempty"`

	// PersistentVolumeClaim specifies if a PVC should make for AKO logging
	// +optional
	PersistentVolumeClaim string `json:"persistentVolumeClaim,omitempty"`

	// MountPath specifies the path to mount PVC
	// +optional
	MountPath string `json:"mountPath,omitempty"`

	// LogFile specifies the log file name
	// +optional
	LogFile string `json:"logFile,omitempty"`
}

type AKORbacConfig struct {
	// PspPolicyAPIVersion decides the API version of the PodSecurityPolicy
	PspPolicyAPIVersion string `json:"pspPolicyAPIVersion,omitempty"`

	// PspEnabled enables the deployment of a PodSecurityPolicy that grants
	// AKO the proper role
	// +optional
	PspEnabled bool `json:"pspEnabled,omitempty"`
}

// AVITenant describes settings for an AVI Tenant object
type AVITenant struct {
	// Context is the type of AVI tenant context. Defaults to Provider. This field is immutable.
	// +kubebuilder:validation:Enum=Provider;Tenant
	Context string `json:"context,omitempty"`

	// Name is the name of the tenant. This field is immutable.
	Name string `json:"name"`
}

// DataNetwork describes one AVI Data Network
type DataNetwork struct {
	Name    string   `json:"name"`
	CIDR    string   `json:"cidr"`
	IPPools []IPPool `json:"ipPools,omitempty"`
}

// ControlPlaneNetwork describes the ControlPlane Network of the clusters selected by an akoDeploymentConfig
type ControlPlaneNetwork struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

// IPPool defines a contiguous range of IP Addresses
type IPPool struct {
	// Start represents the starting IP address of the pool.
	Start string `json:"start"`
	// End represents the ending IP address of the pool.
	End string `json:"end"`
	// Type represents the type of IP Address
	// +kubebuilder:validation:Enum=V4;
	Type string `json:"type"`
}

// SecretReference pointer to SecretRef
type SecretReference *SecretRef

// SecretRef references a Kind Secret object in the same kubernetes
// cluster
type SecretRef struct {
	// Name is the name of resource being referenced.
	Name string `json:"name"`
	// Namespace of the resource being referenced.
	Namespace string `json:"namespace"`
}

// AKODeploymentConfigStatus defines the observed state of AKODeploymentConfig
type AKODeploymentConfigStatus struct {
	// ObservedGeneration reflects the generation of the most recently
	// observed AKODeploymentConfig.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions defines current state of the AKODeploymentConfig.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=adc,path=akodeploymentconfigs,scope=Cluster
// +kubebuilder:subresource:status

// AKODeploymentConfig is the Schema for the akodeploymentconfigs API
type AKODeploymentConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AKODeploymentConfigSpec   `json:"spec,omitempty"`
	Status AKODeploymentConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AKODeploymentConfigList contains a list of AKODeploymentConfig
type AKODeploymentConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AKODeploymentConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AKODeploymentConfig{}, &AKODeploymentConfigList{})
}
