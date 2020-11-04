module gitlab.eng.vmware.com/core-build/tkg-connectivity/hack/tools

go 1.13

require (
	github.com/k14s/ytt v0.28.0
	github.com/onsi/ginkgo v1.11.0
	k8s.io/code-generator v0.17.7

	sigs.k8s.io/controller-tools v0.2.9
)

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200522161834-8a7b2030a110 // ytt branch
