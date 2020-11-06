module gitlab.eng.vmware.com/core-build/tkg-connectivity/hack/tools

go 1.13

require (
	github.com/golangci/golangci-lint v1.32.2 // indirect
	github.com/k14s/ytt v0.28.0
	github.com/onsi/ginkgo v1.14.1
	k8s.io/code-generator v0.18.2
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect

	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/kustomize/kustomize/v3 v3.5.4
)

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200522161834-8a7b2030a110 // ytt branch
