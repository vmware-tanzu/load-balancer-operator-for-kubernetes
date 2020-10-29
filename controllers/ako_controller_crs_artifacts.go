package controllers

var (
  akoDeploymentYaml = `
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ako-sa
  namespace: {{ .Values.Namespace }}  

---
apiVersion: v1
kind: Secret
metadata:
  name: avi-secret
  namespace: {{ .Values.Namespace }}
type: Opaque
data:
  username: {{ .Values.Avicredentials.Username }}
  password: {{ .Values.Avicredentials.Password }}
    
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: avi-k8s-config
  namespace: {{ .Values.Namespace }}
data:
  controllerIP: "{{ .Values.ControllerSettings.ControllerIP }}"
  controllerVersion: "{{ .Values.ControllerSettings.ControllerVersion }}"
  cniPlugin: "{{ .Values.AKOSettings.CniPlugin }}"
  shardVSSize: "{{ .Values.L7Settings.ShardVSSize }}"
  passthroughShardSize: "{{ .Values.L7Settings.PassthroughShardSize }}"
  fullSyncFrequency: "{{ .Values.AKOSettings.FullSyncFrequency }}"
  cloudName: "{{ .Values.ControllerSettings.CloudName }}"
  clusterName: "{{ .Values.AKOSettings.ClusterName }}"
  defaultDomain: "{{ .Values.L4Settings.DefaultDomain }}"
  disableStaticRouteSync: "{{ .Values.AKOSettings.DisableStaticRouteSync }}"
  defaultIngController: "{{ .Values.L7Settings.DefaultIngController }}"
  subnetIP: "{{ .Values.NetworkSettings.SubnetIP }}"
  subnetPrefix: "{{ .Values.NetworkSettings.SubnetPrefix }}"
  networkName: "{{ .Values.NetworkSettings.NetworkName }}"
  l7ShardingScheme: "{{ .Values.L7Settings.L7ShardingScheme }}"
  logLevel: "{{ .Values.AKOSettings.LogLevel }}"
  deleteConfig: "{{ .Values.AKOSettings.DeleteConfig }}"
  {{ if .Values.AKOSettings.SyncNamespace  }}
  syncNamespace: {{ .Values.AKOSettings.SyncNamespace }}
  {{ end }}
  serviceType:  "{{ .Values.L7Settings.ServiceType }}"
  {{ if eq .Values.L7Settings.ServiceType "NodePort" }}
  nodeKey: "{{ .Values.NodePortSelector.Key }}"
  nodeValue: "{{ .Values.NodePortSelector.Value }}"
  {{ end }}
  serviceEngineGroupName: "{{ .Values.ControllerSettings.ServiceEngineGroupName }}"
  nodeNetworkList: |-
    {{ .Values.NetworkSettings.NodeNetworkListJson }}
  apiServerPort: "{{ .Values.AKOSettings.ApiServerPort }}"

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
{{- if .Values.Rbac.PspEnable }}
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
  labels:
    chart: {{ .Values.ChartName }}-{{ .Values.AppVersion }}
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
    app.kubernetes.io/version: "{{ .Values.AppVersion }}"
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
        - name: {{ .Values.ChartName }}
          securityContext: null
          {{ if .Values.PersistentVolumeClaim }}
          volumeMounts:
          - mountPath: {{ .Values.MountPath }}
            name: ako-pv-storage
          {{ end }}
          image: "{{ .Values.Image.Repository }}:{{ .Values.AppVersion }}"
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
          - name: CTRL_VERSION
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: controllerVersion
          - name: CNI_PLUGIN
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: cniPlugin
          - name: SHARD_VS_SIZE
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: shardVSSize
          - name: PASSTHROUGH_SHARD_SIZE
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: passthroughShardSize
          - name: FULL_SYNC_INTERVAL
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: fullSyncFrequency
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
          - name: DEFAULT_DOMAIN
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: defaultDomain
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
          - name: NODE_NETWORK_LIST
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: nodeNetworkList
          - name: SERVICE_TYPE
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: serviceType
          {{ if eq .Values.L7Settings.ServiceType "NodePort" }}
          - name: NODE_KEY
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: nodeKey
          - name: NODE_VALUE
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: nodeValue
          {{ end }}
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
          - name: L7_SHARD_SCHEME
            valueFrom:
              configMapKeyRef:
                name: avi-k8s-config
                key: l7ShardingScheme
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
{{- if .Values.Rbac.PspEnable }}
apiVersion: {{ .Values.PsppolicyApiVersion }} 
kind: PodSecurityPolicy
metadata:
  name: {{ .Values.Name }}
  labels:
    {{- if .Values.IsClusterService }}
    k8s-app: {{ .Values.ChartName }}
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: "AKO"
    {{- else }}
    app.kubernetes.io/name: {{ .Values.Name }}
    {{- end }}
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
`
)
