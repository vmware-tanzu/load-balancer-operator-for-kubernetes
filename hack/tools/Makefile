# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

# If you update this file, please follow
# https://suva.sh/posts/well-documented-makefiles

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq (,$(strip $(GOPROXY)))
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Active module mode, as we use go modules to manage dependencies
export GO111MODULE := on

# Directories.
BIN_DIR := $(shell pwd)/bin
SRCS := go.mod go.sum

# Versions.
KUBEBUILDER_VERSION=3.14.2
K8S_VERSION=1.29.3

# Host information.
HOST_OS=$(shell go env GOOS)
HOST_ARCH=$(shell go env GOARCH)

# Binaries.
CONTROLLER_GEN  := $(BIN_DIR)/controller-gen
CONVERSION_GEN  := $(BIN_DIR)/conversion-gen
GOLANGCI_LINT   := $(BIN_DIR)/golangci-lint
KUSTOMIZE       := $(BIN_DIR)/kustomize
GINKGO          := $(BIN_DIR)/ginkgo
KUBE_APISERVER  := $(BIN_DIR)/kube-apiserver
KUBEBUILDER     := $(BIN_DIR)/kubebuilder
KUBECTL         := $(BIN_DIR)/kubectl
ETCD            := $(BIN_DIR)/etcd
KIND            := $(BIN_DIR)/kind
JQ              := $(BIN_DIR)/jq
ENVTEST         := $(BIN_DIR)/setup-envtest

## --------------------------------------
## Help
## --------------------------------------

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Binaries
## --------------------------------------

controller-gen: $(CONTROLLER_GEN) $(SRCS)
controller-gen: ## Build controller-gen
$(CONTROLLER_GEN):
	go build -tags=tools -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

golangci-lint: $(GOLANGCI_LINT) $(SRCS)
golangci-lint: ## Build golangci-lint
$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.59.0

kustomize: $(KUSTOMIZE) $(SRCS)
kustomize: ## Build kustomize
$(KUSTOMIZE):
	go build -tags=tools -o $@ sigs.k8s.io/kustomize/kustomize/v5

conversion-gen: $(CONVERSION_GEN) $(SRCS)
conversion-gen: ## Build conversion-gen
$(CONVERSION_GEN):
	go build -tags=tools -o $@ k8s.io/code-generator/cmd/conversion-gen

ginkgo: $(GINKGO) $(SRCS)
ginkgo: ## Build ginkgo
$(GINKGO):
	go build -tags=tools -o $@ github.com/onsi/ginkgo/ginkgo

kind: $(KIND) $(SRCS)
kind: ## Build kind
$(KIND):
	go build -tags=tools -o $@ sigs.k8s.io/kind

jq: $(JQ) ## Install jq
$(JQ):
	curl -o $(@) -L https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 && \
		chmod a+x $(@)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST):
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20240215143116-d0396a3d6f9f

etcd: $(ETCD) ## Install etcd
kube-apiserver: $(KUBE_APISERVER) ## Install kube-apiserver
kubectl: $(KUBECTL) ## Install kubectl
$(ETCD) $(KUBE_APISERVER) $(KUBECTL): envtest
	TEMP_DL_DIR=$(shell mktemp -d) && \
	GOBIN=$(BIN_DIR) $(ENVTEST) use $(K8S_VERSION) --bin-dir "$${TEMP_DL_DIR}" && \
	cp "$${TEMP_DL_DIR}"/k8s/*/* $(@D)

kubebuilder: $(KUBEBUILDER) ## Install kubebuilder
$(KUBEBUILDER):
	@mkdir -p $(@D)
	TEMP_DL_DIR=$$(mktemp -d) && \
	curl -sL https://github.com/kubernetes-sigs/kubebuilder/releases/download/v$(KUBEBUILDER_VERSION)/kubebuilder_$(KUBEBUILDER_VERSION)_$(HOST_OS)_$(HOST_ARCH) -O $(KUBEBUILDER)

## --------------------------------------
## Generate
## --------------------------------------

.PHONY: modules
modules: ## Runs go mod to ensure proper vendoring
	go mod tidy

## --------------------------------------
## Cleanup / Verification
## --------------------------------------

.PHONY: clean
clean: ## Run all the clean targets
	$(MAKE) clean-bin

.PHONY: clean-bin
clean-bin: ## Remove all generated binaries
	rm -rf bin
