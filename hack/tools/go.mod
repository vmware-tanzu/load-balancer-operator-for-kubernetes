module gitlab.eng.vmware.com/core-build/tkg-connectivity/hack/tools

go 1.15

require (
	github.com/k14s/ytt v0.28.0
	github.com/mattn/go-isatty v0.0.11 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0 // indirect
	k8s.io/code-generator v0.18.2
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect
	sigs.k8s.io/controller-tools v0.4.0
	sigs.k8s.io/kustomize/kustomize/v3 v3.8.8
)

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200522161834-8a7b2030a110 // ytt branch
