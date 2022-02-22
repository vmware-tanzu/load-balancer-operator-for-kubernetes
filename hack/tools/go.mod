module github.com/vmware-samples/load-balancer-operator-for-kubernetes/hack/tools

go 1.16

require (
	github.com/k14s/ytt v0.28.0
	github.com/onsi/ginkgo v1.16.4
	k8s.io/code-generator v0.22.2
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/kustomize/kustomize/v4 v4.4.0
)

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200522161834-8a7b2030a110 // ytt branch
