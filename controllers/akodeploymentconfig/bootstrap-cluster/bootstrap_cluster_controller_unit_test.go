// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bootstrap_cluster_test

import (
	"bytes"
	"os"
	"text/template"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	akoov1alpha1 "gitlab.eng.vmware.com/core-build/ako-operator/api/v1alpha1"
	"gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako"
	ako_operator "gitlab.eng.vmware.com/core-build/ako-operator/pkg/ako-operator"
	"k8s.io/utils/pointer"
)

const expectedYamlUnstructured = `
apiVersion: v1
kind: Namespace
metadata:
  name: avi-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ako-sa
  namespace: avi-system
 
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: avi-k8s-config
  namespace: avi-system
data:
  controllerIP: "10.23.122.1"
  serviceEngineGroupName: "Default-SEG"
  cloudName: "test-cloud"
  clusterName: "tkg-system-"
  apiServerPort: "8080"
  subnetIP: "10.0.0.0"
  subnetPrefix: "24"
  networkName: "test-akdc"
  disableStaticRouteSync: "true"
  fullSyncFrequency: "1800"
  serviceType:  "NodePort"
  defaultIngController: "true"
  shardVSSize: "MEDIUM"
  deleteConfig: "false"
  vipNetworkList: |-
    [{"networkName":"test-akdc"}]
  
  nodeNetworkList: |-
    [{"networkName":"test-node-network-1","cidrs":["10.0.0.0/24","192.168.0.0/24"]}]
  
  
  
  
  
  
  

---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: ako-cr
rules:
  - apiGroups: [""]
    resources: ["*"]
    verbs: ['get', 'watch', 'list', 'patch']
  - apiGroups: ["apps"]
    resources: ["statefulsets"]
    verbs: ["get","watch","list"]
  - apiGroups: ["apps"]
    resources: ["statefulsets/status"]
    verbs: ["get","watch","list","patch", "update"]
  - apiGroups: ["extensions", "networking.k8s.io"]
    resources: ["ingresses", "ingresses/status"]
    verbs: ["get","watch","list","patch", "update"]
  - apiGroups: [""]
    resources: ["services/status"]
    verbs: ["get","watch","list","patch", "update"]
  - apiGroups: ["crd.projectcalico.org"]
    resources: ["blockaffinities"]
    verbs: ["get", "watch", "list"]
  - apiGroups: ["network.openshift.io"]
    resources: ["hostsubnets"]
    verbs: ["get", "watch", "list"]
  - apiGroups: ["route.openshift.io"]
    resources: ["routes", "routes/status"]
    verbs: ["get", "watch", "list", "patch", "update"]
  - apiGroups: ["ako.vmware.com"]
    resources: ["aviinfrasettings", "aviinfrasettings/status", "hostrules", "hostrules/status", "httprules", "httprules/status"]
    verbs: ["get","watch","list","patch", "update"]
  - apiGroups: ["networking.x-k8s.io"]
    resources: ["gateways", "gateways/status", "gatewayclasses", "gatewayclasses/status"]
    verbs: ["get","watch","list","patch", "update"]
  - apiGroups:
    - policy
    - extensions
    resources:
    - podsecuritypolicies
    verbs:
    - use
    resourceNames:
    - ako-tkg-system-

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ako-crb
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ako-cr
subjects:
- kind: ServiceAccount
  name: ako-sa
  namespace: avi-system

---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ako
  namespace: avi-system
  labels:
    app.kubernetes.io/name: ako-tkg-system-
spec:
  replicas: 1
  serviceName: ako
  selector:
    matchLabels:
      app.kubernetes.io/name: ako-tkg-system-
  template:
    metadata:
      labels:
        app.kubernetes.io/name: ako-tkg-system-
    spec:
      serviceAccountName: ako-sa
      securityContext: {}
      
      volumes:
      - name: ako-pv-storage
        persistentVolumeClaim:
          claimName: true
      
      containers:
        - name: ako-tkg-system-
          securityContext: null
          
          volumeMounts:
          - mountPath: /var/log
            name: ako-pv-storage
          
          env:
          - name: CTRL_USERNAME
            valueFrom:
              secretKeyRef:
                name: avi-secret
                key: username
          - name: CTRL_PASSWORD
            valueFrom:
              secretKeyRef:
                name: avi-secret
                key: password
          - name: FULL_SYNC_INTERVAL
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: fullSyncFrequency
          - name: CTRL_IPADDRESS
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: controllerIP
          - name: CLOUD_NAME
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: cloudName
          - name: CLUSTER_NAME
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: clusterName
          - name: DISABLE_STATIC_ROUTE_SYNC
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: disableStaticRouteSync
           
          - name: NODE_NETWORK_LIST
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: nodeNetworkList
           
           
          - name: SUBNET_IP
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: subnetIP
          - name: SUBNET_PREFIX
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: subnetPrefix
          - name: VIP_NETWORK_LIST
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: vipNetworkList
          - name: DEFAULT_ING_CONTROLLER
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: defaultIngController
          - name: NETWORK_NAME
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: networkName
          - name: SEG_NAME
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: serviceEngineGroupName
          - name: SERVICE_TYPE
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: serviceType
          
          - name: USE_PVC
            value: "true"
          
          - name: LOG_FILE_PATH
            value: /var/log
          - name: LOG_FILE_NAME
            value: test-avi.log
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: POD_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          resources:
            limits:
              cpu: 250m
              memory: 300Mi
            requests:
              cpu: 100m
              memory: 200Mi
          livenessProbe:
            httpGet:
              path: /api/status
              port:  8080
            initialDelaySeconds: 5
            periodSeconds: 10

---
apiVersion: test/1.2 
kind: PodSecurityPolicy
metadata:
  name: ako-tkg-system-
  labels:
    app.kubernetes.io/name: ako-tkg-system-
spec:
  privileged: false
  # Required to prevent escalations to root.
  allowPrivilegeEscalation: false
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    # Require the container to run without root privileges.
    rule: 'RunAsAny'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'MustRunAs'
    ranges:
      # Forbid adding the root group.
      - min: 1
        max: 65535
  fsGroup:
    rule: 'MustRunAs'
    ranges:
      # Forbid adding the root group.
      - min: 1
        max: 65535
  readOnlyRootFilesystem: false
---

---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: hostrules.ako.vmware.com
spec:
  conversion:
    strategy: None
  group: ako.vmware.com
  names:
    kind: HostRule
    listKind: HostRuleList
    plural: hostrules
    shortNames:
    - hostrule
    - hr
    singular: hostrule
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              virtualhost:
                properties:
                  analyticsProfile:
                    type: string
                  applicationProfile:
                    type: string
                  enableVirtualHost:
                    type: boolean
                  errorPageProfile:
                    type: string
                  fqdn:
                    type: string
                  datascripts:
                    items:
                      type: string
                    type: array
                  httpPolicy:
                    properties:
                      overwrite:
                        type: boolean
                      policySets:
                        items:
                          type: string
                        type: array
                    type: object
                  tls:
                    properties:
                      sslProfile:
                        type: string
                      sslKeyCertificate:
                        properties:
                          name:
                            type: string
                          type:
                            enum:
                            - ref
                            type: string
                        required:
                        - name
                        - type
                        type: object
                      termination:
                        enum:
                        - edge
                        type: string
                    required:
                    - sslKeyCertificate
                    type: object
                  wafPolicy:
                    type: string
                required:
                - fqdn
                type: object
            required:
            - virtualhost
            type: object
          status:
            properties:
              error:
                type: string
              status:
                type: string
            type: object
        type: object
    additionalPrinterColumns:
    - description: virtualhost for which the hostrule is valid
      jsonPath: .spec.virtualhost.fqdn
      name: Host
      type: string
    - description: status of the hostrule object
      jsonPath: .status.status
      name: Status
      type: string
    - jsonPath: .metadata.creationTimestamp
      name: Age
      type: date
    served: true
    storage: true
    subresources:
      status: {}
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: httprules.ako.vmware.com
spec:
  group: ako.vmware.com
  names:
    plural: httprules
    singular: httprule
    listKind: HTTPRuleList
    kind: HTTPRule
    shortNames:
    - httprule
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              fqdn:
                type: string
              paths:
                items:
                  properties:
                    loadBalancerPolicy:
                      properties:
                        algorithm:
                          enum:
                          - LB_ALGORITHM_CONSISTENT_HASH
                          - LB_ALGORITHM_CORE_AFFINITY
                          - LB_ALGORITHM_FASTEST_RESPONSE
                          - LB_ALGORITHM_FEWEST_SERVERS
                          - LB_ALGORITHM_LEAST_CONNECTIONS
                          - LB_ALGORITHM_LEAST_LOAD
                          - LB_ALGORITHM_ROUND_ROBIN
                          type: string
                        hash:
                          enum:
                          - LB_ALGORITHM_CONSISTENT_HASH_CALLID
                          - LB_ALGORITHM_CONSISTENT_HASH_SOURCE_IP_ADDRESS
                          - LB_ALGORITHM_CONSISTENT_HASH_SOURCE_IP_ADDRESS_AND_PORT
                          - LB_ALGORITHM_CONSISTENT_HASH_URI
                          - LB_ALGORITHM_CONSISTENT_HASH_CUSTOM_HEADER
                          - LB_ALGORITHM_CONSISTENT_HASH_CUSTOM_STRING
                          type: string
                        hostHeader:
                          type: string
                      type: object
                    target:
                      pattern: ^\/.*$
                      type: string
                    healthMonitors:
                      items:
                        type: string
                      type: array
                    tls:
                      properties:
                        destinationCA:
                          type: string
                        sslProfile:
                          type: string
                        type:
                          enum:
                          - reencrypt
                          type: string
                      required:
                      - type
                      type: object
                  required:
                  - target
                  type: object
                type: array
            required:
            - fqdn
            type: object
          status:
            properties:
              error:
                type: string
              status:
                type: string
            type: object
        type: object
    additionalPrinterColumns:
    - description: fqdn associated with the httprule
      jsonPath: .spec.fqdn
      name: HOST
      type: string
    - description: status of the httprule object
      jsonPath: .status.status
      name: Status
      type: string
    - jsonPath: .metadata.creationTimestamp
      name: Age
      type: date
    served: true
    storage: true
    subresources:
      status: {}
---
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: aviinfrasettings.ako.vmware.com
spec:
  conversion:
    strategy: None
  group: ako.vmware.com
  names:
    kind: AviInfraSetting
    listKind: AviInfraSettingList
    plural: aviinfrasettings
    singular: aviinfrasetting
  scope: Cluster
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: AviInfraSetting is used to select specific Avi controller infra attributes.
        properties:
          spec:
            properties:
              network:
                properties:
                  names:
                    items:
                      type: string
                    type: array
                  enableRhi:
                    type: boolean
                type: object
                required:
                - names
              seGroup:
                properties:
                  name:
                    type: string
                type: object
                required:
                - name
              l7Settings:
                properties:
                  shardSize:
                    enum:
                    - SMALL
                    - MEDIUM
                    - LARGE
                    - DEDICATED
                    type: string
                type: object
                required:
                - shardSize
            type: object
          status:
            properties:
              error:
                type: string
              status:
                type: string
            type: object
        type: object
    additionalPrinterColumns:
    - description: status of the nas object
      jsonPath: .status.status
      name: Status
      type: string
    - jsonPath: .metadata.creationTimestamp
      name: Age
      type: date
    served: true
    storage: true
    subresources:
      status: {}
`

func unitTestConvertToDeploymentYaml() {
	Context("Populate deployment components", func() {
		var (
			akoDeploymentConfig *akoov1alpha1.AKODeploymentConfig
		)

		When("valid AKODeploymentYaml is provided", func() {
			BeforeEach(func() {
				akoDeploymentConfig = &akoov1alpha1.AKODeploymentConfig{
					Spec: akoov1alpha1.AKODeploymentConfigSpec{
						CloudName:          "test-cloud",
						Controller:         "10.23.122.1",
						ServiceEngineGroup: "Default-SEG",
						DataNetwork: akoov1alpha1.DataNetwork{
							Name: "test-akdc",
							CIDR: "10.0.0.0/24",
						},
						ControlPlaneNetwork: akoov1alpha1.ControlPlaneNetwork{
							Name: "integration-test-8ed12g",
							CIDR: "10.1.0.0/24",
						},
						ExtraConfigs: akoov1alpha1.ExtraConfigs{
							Rbac: akoov1alpha1.AKORbacConfig{
								PspEnabled:          true,
								PspPolicyAPIVersion: "test/1.2",
							},
							Log: akoov1alpha1.AKOLogConfig{
								PersistentVolumeClaim: "true",
								MountPath:             "/var/log",
								LogFile:               "test-avi.log",
							},
							IngressConfigs: akoov1alpha1.AKOIngressConfig{
								DisableIngressClass:      true,
								DefaultIngressController: true,
								ShardVSSize:              "MEDIUM",
								ServiceType:              "NodePort",
								NodeNetworkList: []akoov1alpha1.NodeNetwork{
									{
										NetworkName: "test-node-network-1",
										Cidrs:       []string{"10.0.0.0/24", "192.168.0.0/24"},
									},
								},
							},
							DisableStaticRouteSync: pointer.BoolPtr(true),
						},
					},
				}
			})

			It("should generate exact yaml unstructured", func() {
				tmpl, err := template.New("deployment").Parse(ako.AkoDeploymentYamlTemplate)
				Expect(err).Should(BeNil())
				managementClusterName := os.Getenv(ako_operator.ManagementClusterName)
				values, err := ako.NewValues(akoDeploymentConfig, akoov1alpha1.TKGSystemNamespace+"-"+managementClusterName)
				Expect(err).Should(BeNil())

				var buf bytes.Buffer
				err = tmpl.Execute(&buf, map[string]interface{}{"Values": values})
				Expect(err).Should(BeNil())
				Expect(buf.String()).Should(Equal(expectedYamlUnstructured))
			})
		})
	})
}
