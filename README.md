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

# Getting Started

## Setup

Creating a CAPD based testing environment

```bash
make ytt
hack/e2e.sh -u
```

This will create a management cluster and a workload cluster locally in Docker
for you.

## Run AKOO against the mangement cluster
```bash
go build -o bin/manager main.go
./bin/manager -kubeconfig tkg-lcp.kubeconfig
```
