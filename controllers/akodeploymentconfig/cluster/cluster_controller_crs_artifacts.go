// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cluster

var (
	akoDeploymentYamlTemplate = `
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
  serviceType:  "{{ .Values.L7Settings.ServiceType }}"
  defaultIngController: "{{ .Values.L7Settings.DefaultIngController }}"
  deleteConfig: "{{ .Values.AKOSettings.DeleteConfig }}"
  {{ if .Values.NetworkSettings.NodeNetworkListJson }}
  nodeNetworkList: |-
    {{ .Values.NetworkSettings.NodeNetworkListJson }}
  {{ end }}
  {{/* The following fields in .Values.ControllerSettings are omitted:
          1. controllerVersion: because we don't consider backward compatibility in Calgary so
	     there is no explicit intention to set it;
             controllerVersion: "{{ .Values.ControllerSettings.ControllerVersion }}"
  */}}
  {{/* The following fields in .Values.AKOSettings are omitted:
          1. cniPlugin
	     cniPlugin: "{{ .Values.AKOSettings.CniPlugin }}"
	  2. fullSyncFrequency
	     fullSyncFrequency: "{{ .Values.AKOSettings.FullSyncFrequency }}"
       The following fields are used:
          1. disableStaticRouteSync
	  2. deleteConfig
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
	  5. ShardVSSize;
	     shardVSSize: "{{ .Values.L7Settings.ShardVSSize }}"
       The following fiels are used:
	  1. serviceType
	  2. defaultIngController
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
    verbs: ['get', 'watch', 'list']
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
    resources: ["hostrules", "hostrules/status", "httprules", "httprules/status"]
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
          image: "{{ .Values.Image.Repository }}:{{ .Values.Image.Version }}"
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
{{ if not .Values.DisableIngressClass }}
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
`
)
