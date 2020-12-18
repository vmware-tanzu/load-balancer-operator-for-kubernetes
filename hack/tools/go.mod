module gitlab.eng.vmware.com/core-build/tkg-connectivity/hack/tools

go 1.15

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/drone/envsubst v1.0.3-0.20200709231038-aa43e1c1a629
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/golangci/golangci-lint v1.27.0
	github.com/joelanford/go-apidiff v0.0.0-20191206194835-106bcff5f060
	github.com/k14s/ytt v0.28.0
	github.com/onsi/ginkgo v1.12.0
	github.com/raviqqe/liche v0.0.0-20200229003944-f57a5d1c5be4
	golang.org/x/tools v0.0.0-20200502202811-ed308ab3e770
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
	k8s.io/code-generator v0.18.0
	k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29 // indirect
	sigs.k8s.io/controller-tools v0.2.9
	sigs.k8s.io/kind v0.7.0 // indirect
	sigs.k8s.io/kubebuilder/docs/book/utils v0.0.0-20200226075303-ed8438ec10a4
	sigs.k8s.io/kustomize/kustomize/v3 v3.5.4
	sigs.k8s.io/testing_frameworks v0.1.2
)

replace go.starlark.net => github.com/k14s/starlark-go v0.0.0-20200522161834-8a7b2030a110 // ytt branch
