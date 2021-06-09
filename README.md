# Load Balancer Operator for Kubernetes

## Useful links

- [Quick Start](./docs/quick-start.md)

Load Balancer Operator for Kubernetes is a Cluster API speaking operator for AKO([AVI Kubernetes Operator](https://github.com/vmware/load-balancer-and-ingress-services-for-kubernetes)), which

1. manages lifecycles of AKO
   1. AKO will be deployed automatically to the selected group of clusters
   2. AKO will also be upgraded automatically when AKO Operator is upgraded

2. cleans up left behind resources on AVI when a workload cluster is deleted from the management cluster by "tkg delete cluster"
   1. This is necessary because AKO will be deleted together immediately with the cluster by the above command
   2. With AKO Operator in place, the resource cleanup will be gracefully handled so no dangling Virtual Service left behind

3. reconciles Cluster API objects and provisions Service type LoadBalancer for control plane Machines

4. automates user account and tenant creation, and inject user credentials to TKG workload clusters to achieve better:
   1. resource isolation
   2. security
   3. Note: AKO is agnostic of all these changes

## Contributing

We welcome new contributors to our repository. Following are the pre-requisties that should help
you get started:

- Before contributing, please get familiar with our
[Code of Conduct](CODE-OF-CONDUCT.md).
- Check out our [Contributor Guide](CONTRIBUTING.md) for information
about setting up your development environment and our contribution workflow.

## License

Load Balancer Operator for Kubernetes is licensed under the [Apache License, version 2.0](LICENSE.txt)
