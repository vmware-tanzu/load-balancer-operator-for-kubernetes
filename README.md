# AKO Operator

An Cluster API speaking operator for AKO(AVI Kubernetes Operator), which

1. manages lifecycles of AKO
   1. See [Getting started with AKO](https://confluence.eng.vmware.com/display/TKG/Getting+started+with+AKO) for the manual steps required nowadays for EACH TKG workload cluster;
   2. AKO will be deployed automatically to the selected group of clusters

2. cleans up left behind resources on AVI when a workload cluster is deleted from the management cluster by "tkg delete cluster"
   1. This is necessary because AKO will be deleted together immediately with the cluster by the above command
   2. With AKO Operator in place, the resource cleanup will be gracefully handled so no dangling Virtual Service left behind

3. automates infrastructure-related resource management on behalf of the platform operator(or VI Admin), eg:
   1. DVPG creation/deletion/configuration in vCenter
      1. See [How to create a Distributed vSwitch on vSphere](https://confluence.eng.vmware.com/display/TKG/How+to+create+a+Distributed+vSwitch+on+vSphere) for the manual steps to provision a Data Network in vSphere;
   2. IP Pool configuration in AVI
      1. See [How to deploy AKO in a separate Data Network](https://confluence.eng.vmware.com/display/TKG/How+to+deploy+AKO+in+a+separate+Data+Network) for the manual steps required nowadays to consume the Data Network in AVI

## Coming Next

### Multitenancy

When AVI is upgraded to Enterprise version, AKO Operator could:

1. automates the tenant management by automatically creates user accounts, tenant object in AVI, and inject scoped user credentials to TKG workload clusters to achieve better:
   1. resource isolation
   2. security
   3. Note: AKO is agnostic of all these changes

## Getting Started

### Install AKO Operator in TKGm

```bash
make deploy-ako-operator
```

## Local Development

### Setup

Creating a CAPD based testing environment

```bash
make ytt
hack/e2e.sh -u
```

This will create a management cluster and a workload cluster locally in Docker
for you.

### Run AKOO against the mangement cluster

```bash
# Set current kubectl context to the local management cluster
kubectl config use-context kind-tkg-lcp

# Install AKODeploymentConfig CR
make install

# Build the AKO Operator binary
go build -o bin/manager main.go

# Run AKO Operator in the local management cluster
./bin/manager
```

### Run controller tests

```bash
make integration-test
```

### Run e2e test in kind

```bash
# Create a management cluster and a workload cluster
make ytt
./hack/e2e.sh -u

# Set aliases for accessing both clusters
alias kk="kubectl --kubeconfig=$PWD/tkg-lcp.kubeconfig"
alias kw="kubectl --kubeconfig=$PWD/workload-cls.kubeconfig"

# Set the default kubeconfig to the management cluster
export KUBECONFIG=$PWD/tkg-lcp.kubeconfig

# Build docker image for the AKO Operator
make docker-build

# Load the AKO Operator docker image into the management cluster
kind load docker-image --name tkg-lcp harbor-pks.vmware.com/tkgextensions/tkg-networking/tanzu-ako-operator:dev

# Deploy the AKO Operator in the management cluster
make deploy

# Make sure AKO Operator is up and running
➜ git: ✗ kk get pods -n akoo-system
NAME                                       READY   STATUS    RESTARTS   AGE
akoo-controller-manager-757949b86c-6wwn7   2/2     Running   0          3s

# Checking the operator's log

➜ git: ✗ kk logs akoo-controller-manager-757949b86c-6wwn7 -c manager -n akoo-system | tail -n 10
{"level":"info","ts":1604639438.7660556,"logger":"controllers.Cluster","msg":"cluster doesn't have AVI enabled, skip reconciling","Cluster":"default/workload-cls"}
{"level":"info","ts":1604639438.7642214,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"machine","source":"kind source: /, Kind="}
{"level":"info","ts":1604639438.7675326,"logger":"controller-runtime.controller","msg":"Starting Controller","controller":"machine"}
{"level":"info","ts":1604639438.7678108,"logger":"controller-runtime.controller","msg":"Starting workers","controller":"machine","worker count":1}
{"level":"info","ts":1604639438.769301,"logger":"controllers.Machine","msg":"Cluster doesn't have AVI enabled, skip reconciling","Machine":"default/workload-cls-worker-0-85c7655bb4-vq6c9","Cluster":"default/workload-cls"}
{"level":"info","ts":1604639438.7707927,"logger":"controllers.Machine","msg":"Cluster doesn't have AVI enabled, skip reconciling","Machine":"default/workload-cls-controlplane-0-4bsrd","Cluster":"default/workload-cls"}
{"level":"info","ts":1604639438.7641554,"logger":"controller-runtime.controller","msg":"Starting Controller","controller":"akodeploymentconfig"}
{"level":"info","ts":1604639438.7752495,"logger":"controller-runtime.controller","msg":"Starting workers","controller":"akodeploymentconfig","worker count":1}

# Open another terminal to watch on AKO Operator's log
➜ git: ✗ kk logs akoo-controller-manager-757949b86c-6wwn7 -c manager -f -n akoo-system

# Enable AVI in the workload cluster
➜ git: ✗ kk label cluster workload-cls cluster-service.network.tkg.tanzu.vmware.com/avi=""
cluster.cluster.x-k8s.io/workload-cls labeled

# Making sure AKO is deployed into the workload cluster
➜ git: ✗ kw get pods  ako-0
NAME    READY   STATUS    RESTARTS   AGE
ako-0   1/1     Running   0          40s

➜ git: ✗ kw get configmap
NAME             DATA   AGE
avi-k8s-config   23     77s

# Making sure AKO Operator adds the finalizer on the cluster
➜  ako-operator git:(update-readme) ✗ kk get cluster workload-cls -o yaml  | head
apiVersion: cluster.x-k8s.io/v1alpha3
kind: Cluster
metadata:
...
  finalizers:
  - cluster.cluster.x-k8s.io
  - ako-operator.network.tkg.tanzu.vmware.com

# Making sure the pre-terminate hook is added to the workload cluster Machines
➜ git: ✗ kk get machine -o yaml | grep terminate
      pre-terminate.delete.hook.machine.cluster.x-k8s.io/avi-cleanup: ako-operator
      pre-terminate.delete.hook.machine.cluster.x-k8s.io/avi-cleanup: ako-operator

# Try to delete the workload cluster. This will be a blocking operation, so hit
Ctrl+C to exit
➜ git:(update-readme) ✗ kk delete cluster workload-cls
cluster.cluster.x-k8s.io "workload-cls" deleted

# You should see something similar in the log
{"level":"info","ts":1604640295.9056501,"logger":"controllers.Cluster","msg":"Handling deleted Cluster","Cluster":"default/workload-cls"}
{"level":"info","ts":1604640296.3605769,"logger":"controllers.Cluster","msg":"Found AKO Configmap","Cluster":"default/workload-cls","deleteConfig":"false"}
{"level":"info","ts":1604640296.3606339,"logger":"controllers.Cluster","msg":"Updating deleteConfig in AKO's ConfigMap","Cluster":"default/workload-cls"}
{"level":"info","ts":1604640296.3698053,"logger":"controllers.Cluster","msg":"AKO finished cleanup, updating Cluster condition","Cluster":"default/workload-cls"}
{"level":"info","ts":1604640296.3698819,"logger":"controllers.Cluster","msg":"Removing finalizer","Cluster":"default/workload-cls","finalizer":"ako-operator.network.tkg.tanzu.vmware.com"}

# Check if the cluster is deleted successfully
➜ git: ✗ kk get cluster
No resources found in default namespace.
```
