# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0
IMAGE_REGISTRY ?= harbor-repo.vmware.com/tanzu_ako_operator_dev
IMAGE_TAG ?= $(shell git log -1 --format=%h)
CACHE_IMAGE_REGISTRY ?= harbor-repo.vmware.com/dockerhub-proxy-cache
# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_REGISTRY)/tkg-networking/tanzu-ako-operator:$(IMAGE_TAG)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"

# Gobuild
PUBLISH?=publish
BUILD_VERSION ?= $(shell git describe --always --match "v*" | sed 's/v//')

# TKG Version
TKG_VERSION ?= v1.10.0+vmware.2

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

BIN_DIR       := bin
TOOLS_DIR := hack/tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
export PATH := $(abspath $(BIN_DIR)):$(abspath $(TOOLS_BIN_DIR)):$(PATH)
export KUBEBUILDER_ASSETS := $(abspath $(TOOLS_BIN_DIR))

ifdef GITHUB_ACTIONS
export GOPROXY := direct
else
export GOPROXY := https://build-artifactory.eng.vmware.com/gocenter.io,direct
endif


CONTROLLER_GEN     := $(TOOLS_BIN_DIR)/controller-gen
GOLANGCI_LINT      := $(TOOLS_BIN_DIR)/golangci-lint
KUSTOMIZE          := $(TOOLS_BIN_DIR)/kustomize
GINKGO             := $(TOOLS_BIN_DIR)/ginkgo
KUBE_APISERVER     := $(TOOLS_BIN_DIR)/kube-apiserver
KUBEBUILDER        := $(TOOLS_BIN_DIR)/kubebuilder
KUBECTL            := $(TOOLS_BIN_DIR)/kubectl
ETCD               := $(TOOLS_BIN_DIR)/etcd
KIND               := $(TOOLS_BIN_DIR)/kind
JQ                 := $(TOOLS_BIN_DIR)/jq
YTT := $(abspath $(TOOLS_BIN_DIR)/ytt)

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

cover: test
	go tool cover -func=cover.out -o coverage.txt
	go tool cover -html=cover.out -o coverage.html

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Run go fmt against code
fmt: header-check
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Run header check against code
header-check:
	./hack/header-check.sh

# Generate code
generate: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build:
ifdef GITHUB_ACTIONS
	docker build . -t ${IMG} -f Dockerfile-for-github-ci
else
	docker build . -t ${IMG}
endif

# Push the docker image
docker-push:
	docker push ${IMG}

.PHONY: integration-test
# TODO:(xudongl) This is used to silence the ginkgo complain, can be removed once upgrade ginkgo to v2
export ACK_GINKGO_DEPRECATIONS=1.16.4
integration-test: $(GINKGO) $(ETCD)
	$(GINKGO) -v controllers/akodeploymentconfig/phases -- -enable-integration-tests -enable-unit-tests=false -root-dir="../../.."
	$(GINKGO) -v controllers/akodeploymentconfig/user -- -enable-integration-tests -enable-unit-tests=false -root-dir="../../.."
	$(GINKGO) -v controllers/akodeploymentconfig -- -enable-integration-tests -enable-unit-tests=false
	$(GINKGO) -v controllers/machine -- -enable-integration-tests -enable-unit-tests=false
	$(GINKGO) -v controllers/cluster -- -enable-integration-tests -enable-unit-tests=false
	$(GINKGO) -v controllers/tests/cluster_for_akodeploymentconfig/default_adc -- -enable-integration-tests -enable-unit-tests=false -root-dir="../../../.."
	$(GINKGO) -v controllers/tests/cluster_for_akodeploymentconfig/default_adc_non_empty_selectors -- -enable-integration-tests -enable-unit-tests=false -root-dir="../../../.."

.PHONY: kind-e2e-test
kind-e2e-test: $(KUSTOMIZE) $(KIND) $(KUBECTL) $(JQ) $(YTT)
	./hack/test-e2e.sh

.PHONY: ytt
ytt: $(YTT)

$(YTT): $(TOOLS_DIR)/go.mod # Build ytt from tools folder.
	cd $(TOOLS_DIR); go build -tags=tools -o $(BIN_DIR)/ytt github.com/k14s/ytt/cmd/ytt

## --------------------------------------
## AKO Operator
## --------------------------------------

# Deploy AKO Operator
.PHONY: deploy-ako-operator
deploy-ako-operator: $(YTT)
	$(YTT) -v imageTag=$(BUILD_VERSION) -f config/ytt/ako-operator.yaml -f config/ytt/static.yaml -f config/ytt/values.yaml | kubectl apply -f -

# Delete AKO Operator
.PHONY: delete-ako-operator
delete-ako-operator: $(YTT)
	$(YTT) -v imageTag=$(BUILD_VERSION) -f config/ytt/ako-operator.yaml -f config/ytt/static.yaml -f config/ytt/values.yaml | kubectl delete -f -

## --------------------------------------
## Manifests and Specs
## --------------------------------------

# Install CRDs into a cluster
install: manifests
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Uninstalls controller in the configured Kubernetes cluster in ~/.kube/config
remove: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: $(CONTROLLER_GEN) $(KUSTOMIZE)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(KUSTOMIZE) build config/kustomize-to-ytt > config/ytt/static.yaml

# Generate manifests from ytt for AKO Operator Deployment
.PHONY: ytt-manifests
ytt-manifests: $(YTT)
	@$(YTT) -v imageTag=$(BUILD_VERSION) -f config/ytt/ako-operator.yaml -f config/ytt/static.yaml -f config/ytt/values.yaml

## --------------------------------------
## Linting and fixing linter errors
## --------------------------------------

.PHONY: lint
lint: ## Run all the lint targets
	$(MAKE) lint-go-full
	$(MAKE) lint-markdown
	$(MAKE) lint-shell

GOLANGCI_LINT_FLAGS ?= --fast=true
.PHONY: lint-go
lint-go: | $(GOLANGCI_LINT) ## Lint codebase
ifdef GITHUB_ACTIONS
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_FLAGS) --timeout 30m ## Allow more time for Github Action, otherwise timeout errors is likely to occur
else
	$(GOLANGCI_LINT) run -v $(GOLANGCI_LINT_FLAGS)
endif

.PHONY: lint-go-full
lint-go-full: GOLANGCI_LINT_FLAGS = --fast=false
lint-go-full: lint-go ## Run slower linters to detect possible issues

.PHONY: lint-markdown
lint-markdown: ## Lint the project's markdown
ifdef GITHUB_ACTIONS
	markdownlint -c md-config.json .
else
	docker run -i --rm -v "$$(pwd)":/work $(CACHE_IMAGE_REGISTRY)/tmknom/markdownlint -c /work/md-config.json .
endif

.PHONY: lint-shell
lint-shell: ## Lint the project's shell scripts
ifdef GITHUB_ACTIONS
	shellcheck hack/*.sh
else
	## Lint the project's shell in Github Action. We can assume shellcheck is in PATH
	docker run --rm -v "$$(pwd):/mnt" $(CACHE_IMAGE_REGISTRY)/koalaman/shellcheck:stable  hack/*.sh
endif

.PHONY: fix
fix: GOLANGCI_LINT_FLAGS = --fast=false --fix
fix: lint-go ## Tries to fix errors reported by lint-go-full target

## --------------------------------------
## Tooling Binaries
## --------------------------------------

.PHONY: $(TOOLING_BINARIES)
TOOLING_BINARIES := $(CONTROLLER_GEN) $(GOLANGCI_LINT) $(KUSTOMIZE) \
                    $(KUBE_APISERVER) $(KUBEBUILDER) $(KUBECTL) \
                    $(ETCD) $(GINKGO) $(KIND) $(JQ)
tools: $(TOOLING_BINARIES) ## Build tooling binaries
$(TOOLING_BINARIES):
	cd $(TOOLS_DIR) && $(MAKE) $(@F)

## --------------------------------------
## Binaries
## --------------------------------------

.PHONY: $(MANAGER)
manager: $(MANAGER) ## Build the controller-manager binary
$(MANAGER): generate-go
	go build -o $@ -ldflags '-extldflags -static -w -s' .

## --------------------------------------
## GoBuild
## --------------------------------------

#.PHONY:	gobuild-config
#gobuild-config: ## Configure gitlab access

.PHONY: gobuild-fetch-unshallow
gobuild-fetch-unshallow:
	-git fetch --unshallow --tags

.PHONY: gobuild-docker-images
gobuild-docker-images: export GOPATH=$(HOME)/go
gobuild-docker-images: export GOROOT=$(GOBUILD_CAYMAN_GO_ROOT)
gobuild-docker-images: export PATH=$(GOPATH)/bin:$(GOBUILD_CAYMAN_GO_ROOT)/bin:$(shell printenv PATH)
gobuild-docker-images: export CGO_ENABLED=0
gobuild-docker-images: export GOPROXY=https://build-artifactory.eng.vmware.com/gocenter.io,direct
gobuild-docker-images: export GOSUMDB=off
gobuild-docker-images: export GOPRIVATE=gitlab.eng.vmware.com
gobuild-docker-images: export IMAGE_REGISTRY=vmware.io
gobuild-docker-images: export VERSION=$(TKG_VERSION)
gobuild-docker-images:
	hack/gobuild/build.sh

# Publish the ako operator docker image to gobuild publish directory
# PUBLISH_DIR environment variable will be set by GoBuild
.PHONY: gobuild-publish
gobuild-publish: export PUBLISH_DIR=$(PUBLISH)
gobuild-publish: export IMAGE_PUBLISH_DIR=$(PUBLISH_DIR)/lin64/ako-operator/images
gobuild-publish: export MANIFESTS_PUBLISH_DIR=$(PUBLISH_DIR)/lin64/ako-operator/manifests
gobuild-publish: export IMAGE_REGISTRY=vmware.io
gobuild-publish: export VERSION=$(TKG_VERSION)
gobuild-publish:
	hack/gobuild/publish.sh
	hack/gobuild/manifests.sh


# Entrypoint for gobuild builds
.PHONY: gobuild
gobuild: gobuild-docker-images gobuild-publish
