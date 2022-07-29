module github.com/vmware-tanzu/load-balancer-operator-for-kubernetes

go 1.17

require (
	github.com/bitly/go-simplejson v0.5.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/go-logr/logr v1.2.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/pkg/errors v0.9.1
	github.com/satori/go.uuid v1.2.0
	github.com/vmware/alb-sdk v0.0.0-20210721142023-8e96475b833b
	github.com/vmware/load-balancer-and-ingress-services-for-kubernetes v0.0.0-20211102041403-f2ed902e4706
	go.uber.org/zap v1.19.1
	gopkg.in/yaml.v3 v3.0.0
	k8s.io/api v0.24.2
	k8s.io/apimachinery v0.24.2
	k8s.io/client-go v0.24.2
	k8s.io/klog/v2 v2.60.1
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/cluster-api v1.2.0
	sigs.k8s.io/controller-runtime v0.12.3
)
