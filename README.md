# Load Balancer Operator for Kubernetes

## Useful links

- [Quick Start](./docs/quick-start.md)

Load Balancer Operator for Kubernetes is a Cluster API speaking operator for load balancers.

## Features

* It reconciles Cluster API objects and provisions Service type LoadBalancer for control plane Machines to achieve HA.
* It leverages [Carvel Packaging APIs](https://carvel.dev/kapp-controller/docs/latest/packaging) to lifecycle manage load balancer provider operator. Currently, we support AKO([AVI Kubernetes Operator](https://github.com/vmware/load-balancer-and-ingress-services-for-kubernetes)) as a reference implementation.
* It bridges [Cluster API](https://cluster-api.sigs.k8s.io/) and load balancer provider operator to ensure load balancer resources are cleaned up when cluster is deleted.
* For AKO, it also automates the user account creation and injection per cluster.

## Contributing

We welcome new contributors to our repository. Following are the pre-requisties that should help
you get started:

- Before contributing, please get familiar with our
[Code of Conduct](CODE-OF-CONDUCT.md).
- Check out our [Contributor Guide](CONTRIBUTING.md) for information
about setting up your development environment and our contribution workflow.

## License

Load Balancer Operator for Kubernetes is licensed under the [Apache License, version 2.0](LICENSE.txt)
