// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package ako

var (
	AkoDeploymentYamlTemplate = `
apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Values.Namespace }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ako-sa
  namespace: {{ .Values.Namespace }}
 
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: avi-k8s-config
  namespace: {{ .Values.Namespace }}
data:
  controllerIP: "{{ .Values.ControllerSettings.ControllerIP }}"
  serviceEngineGroupName: "{{ .Values.ControllerSettings.ServiceEngineGroupName }}"
  cloudName: "{{ .Values.ControllerSettings.CloudName }}"
  clusterName: "{{ .Values.AKOSettings.ClusterName }}"
  apiServerPort: "{{ .Values.AKOSettings.ApiServerPort }}"
  subnetIP: "{{ .Values.NetworkSettings.SubnetIP }}"
  subnetPrefix: "{{ .Values.NetworkSettings.SubnetPrefix }}"
  networkName: "{{ .Values.NetworkSettings.NetworkName }}"
  disableStaticRouteSync: "{{ .Values.AKOSettings.DisableStaticRouteSync }}"
  fullSyncFrequency: "{{ .Values.AKOSettings.FullSyncFrequency }}"
  serviceType:  "{{ .Values.L7Settings.ServiceType }}"
  defaultIngController: "{{ .Values.L7Settings.DefaultIngController }}"
  shardVSSize: "{{ .Values.L7Settings.ShardVSSize }}"
  deleteConfig: "{{ .Values.AKOSettings.DeleteConfig }}"
  vipNetworkList: |-
    {{ .Values.NetworkSettings.VIPNetworkListJson }}
  {{ if .Values.NetworkSettings.NodeNetworkListJson }}
  nodeNetworkList: |-
    {{ .Values.NetworkSettings.NodeNetworkListJson }}
  {{ end }}
  {{ if .Values.AKOSettings.CniPlugin }}
  cniPlugin: "{{ .Values.AKOSettings.CniPlugin }}"
  {{ end }}
  {{/* The following fields in .Values.ControllerSettings are omitted:
          1. controllerVersion: because we don't consider backward compatibility in Calgary so
	     there is no explicit intention to set it;
             controllerVersion: "{{ .Values.ControllerSettings.ControllerVersion }}"
  */}}
  {{/* The following fields in .Values.AKOSettings are used:
      1. disableStaticRouteSync
	  2. deleteConfig
	  3. fullSyncFrequency
  */}}
  {{/* The following fields in .Values.L4Settingsare omitted:
          1. defaultDomain
	     defaultDomain: "{{ .Values.L4Settings.DefaultDomain }}"
  */}}
  {{/* The following fields in .Values.L7Settings are omitted:
          1. l7ShardingScheme;
	     l7ShardingScheme: "{{ .Values.L7Settings.L7ShardingScheme }}"
	  2. nodeKey;
	  3. nodeValue;
	     {{ if eq .Values.L7Settings.ServiceType "NodePort" }}
             nodeKey: "{{ .Values.NodePortSelector.Key }}"
	     nodeValue: "{{ .Values.NodePortSelector.Value }}"
             {{ end }}
	  4. PassthroughShardSize;
	     passthroughShardSize: "{{ .Values.L7Settings.PassthroughShardSize }}"
       The following fiels are used:
          1. serviceType
          2. defaultIngController
      	  3. shardVSSize
  */}}
  {{ if .Values.AKOSettings.SyncNamespace }}
  syncNamespace: {{ .Values.AKOSettings.SyncNamespace }}
  {{ end }}

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
{{- if .Values.Rbac.PspEnabled }}
  - apiGroups:
    - policy
    - extensions
    resources:
    - podsecuritypolicies
    verbs:
    - use
    resourceNames:
    - {{ .Values.Name }}
{{- end }}

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
  namespace: {{ .Values.Namespace }}

---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ako
  namespace: {{ .Values.Namespace }}
  labels:
    app.kubernetes.io/name: {{ .Values.Name }}
    app.kubernetes.io/version: "{{ .Values.Image.Version }}"
spec:
  replicas: {{ .Values.ReplicaCount }}
  serviceName: ako
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ .Values.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ .Values.Name }}
    spec:
      serviceAccountName: ako-sa
      securityContext: {}
      {{ if .Values.PersistentVolumeClaim }}
      volumes:
      - name: ako-pv-storage
        persistentVolumeClaim:
          claimName: {{ .Values.PersistentVolumeClaim }}
      {{ end }}
      containers:
        - name: {{ .Values.Name }}
          securityContext: null
          {{ if .Values.PersistentVolumeClaim }}
          volumeMounts:
          - mountPath: {{ .Values.MountPath }}
            name: ako-pv-storage
          {{ end }}
          image: "{{ .Values.Image.Repository }}/{{ .Values.Image.Path }}:{{ .Values.Image.Version }}"
          imagePullPolicy: {{ .Values.Image.PullPolicy }}
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
           {{ if .Values.NetworkSettings.NodeNetworkListJson }}
          - name: NODE_NETWORK_LIST
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: nodeNetworkList
           {{ end }}
           {{ if .Values.AKOSettings.SyncNamespace  }}
          - name: SYNC_NAMESPACE
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: syncNamespace
          {{ end }}
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
          {{ if .Values.PersistentVolumeClaim }}
          - name: USE_PVC
            value: "true"
          {{ end }}
          - name: LOG_FILE_PATH
            value: {{ .Values.MountPath }}
          - name: LOG_FILE_NAME
            value: {{ .Values.LogFile }}
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
              cpu: {{ .Values.Resources.Limits.Cpu }}
              memory: {{ .Values.Resources.Limits.Memory }}
            requests:
              cpu: {{ .Values.Resources.Requests.Cpu }}
              memory: {{ .Values.Resources.Requests.Memory }}
          livenessProbe:
            httpGet:
              path: /api/status
              port:  {{ .Values.AKOSettings.ApiServerPort }}
            initialDelaySeconds: 5
            periodSeconds: 10

---
{{- if .Values.Rbac.PspEnabled }}
apiVersion: {{ .Values.Rbac.PspPolicyApiVersion }} 
kind: PodSecurityPolicy
metadata:
  name: {{ .Values.Name }}
  labels:
    app.kubernetes.io/name: {{ .Values.Name }}
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
{{- end }}
---
{{ if not .Values.L7Settings.DisableIngressClass }}
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: avi-lb
  {{ if .Values.L7Settings.DefaultIngController }}
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
  {{ end }}
spec:
  controller: ako.vmware.com/avi-lb
  parameters:
    apiGroup: ako.vmware.com
    kind: IngressParameters
    name: external-lb
{{ end }}
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
)
