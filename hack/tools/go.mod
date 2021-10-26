module gitlab.eng.vmware.com/core-build/tkg-connectivity/hack/tools

go 1.16

require (
	github.com/golangci/golangci-lint v1.21.0 // indirect
	github.com/k14s/ytt v0.28.0
	github.com/onsi/ginkgo v1.16.4
	k8s.io/code-generator v0.22.2
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/kind v0.11.1 // indirect
	sigs.k8s.io/kustomize/api v0.10.0 // indirect
	sigs.k8s.io/kustomize/cmd/config v0.10.1 // indirect
	sigs.k8s.io/kustomize/kustomize/v4 v4.4.0 // indirect
)

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200522161834-8a7b2030a110 // ytt branch
