#!/bin/bash

# Copyright (c) 2020 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script was ported over from the TKG/grid-api repo

################################################################################
# usage: e2e
#  This program deploys a local test environment using Kind and the Cluster API
#  provider for Docker (CAPD).
################################################################################

set -o errexit  # Exits immediately on unexpected errors (does not bypass traps)
set -o nounset  # Errors if variables are used without first being defined
set -o pipefail # Non-zero exit codes in piped commands causes pipeline to fail
                # with that code

# Change directories to the parent directory of the one in which this script is
# located.
cd "$(dirname "${BASH_SOURCE[0]}")/.."

export PATH=$PATH:$PWD/hack/tools/bin
export KIND_EXPERIMENTAL_DOCKER_NETWORK=bridge # required for CAPD

################################################################################
##                                  usage
################################################################################

usage="$(
  cat <<EOF
usage: ${0} [FLAGS]
  Deploys a local test environment using Kind and the Cluster API providre for
  Docker (CAPD).

FLAGS
  -h    show this help and exit
  -u    deploy one kind cluster as the CAPI management cluster, then two CAPD
        clusters ontop.
  -d    destroy CAPD and kind clusters.

Globals
  KIND_MANAGEMENT_CLUSTER
        name of the kind management cluster. default: tkg-lcp.
  SKIP_CAPD_IMAGE_LOAD
        skip loading CAPD manager docker image.
  SKIP_CLEANUP_CAPD_CLUSTERS
        skip cleaning up CAPD clusters.
  SKIP_CLEANUP_MGMT_CLUSTER
        skip cleaning up CAPI kind management cluster.

Examples
  Create e2e environment from existing kind cluster with name "my-kind"
        KIND_MANAGEMENT_CLUSTER="my-kind" bash hack/e2e.sh -u
  Destroys all CAPD clusters but not the kind management cluster
        SKIP_CLEANUP_MGMT_CLUSTER="1" bash hack/e2e.sh -d
EOF
)"

################################################################################
##                                   args
################################################################################

# If KIND_MINIMUM_VERSION is empty or unset, use "tkg-lcp"
KIND_MANAGEMENT_CLUSTER="${KIND_MANAGEMENT_CLUSTER:-tkg-lcp}"
CAPI_VERSION="v0.3.10"
# Runtime setup
# By default do not skip anything.
SKIP_CAPD_IMAGE_LOAD="${SKIP_CAPD_IMAGE_LOAD:-}"
SKIP_CLEANUP_CAPD_CLUSTERS="${SKIP_CLEANUP_CAPD_CLUSTERS:-}"
SKIP_CLEANUP_MGMT_CLUSTER="${SKIP_CLEANUP_MGMT_CLUSTER:-}"
E2E_UP=""
E2E_DOWN=""

# The capd manager docker image name and tag. These have to be consistent with
# what're used in capd default kustomize manifest.
CAPD_DEFAULT_IMAGE="gcr.io/k8s-staging-cluster-api/capd-manager:dev"
CAPD_IMAGE="gcr.io/k8s-staging-cluster-api/capd-manager:${CAPI_VERSION}"

ROOT_DIR="$PWD"
SCRIPTS_DIR="${ROOT_DIR}/hack"

IMAGE_REGISTRY="${IMAGE_REGISTRY:-gcr.io/kubernetes-development-244305}"
IMAGE_TAG="${IMAGE_TAG:-dev}"

WORKLOAD_CLUSTERS=${WORKLOAD_CLUSTERS:-"workload-cls"}

################################################################################
##                                  require
################################################################################

function check_dependencies() {
  # Ensure Kind 0.7.0+ is available.
  command -v kind >/dev/null 2>&1 || fatal "kind 0.7.0+ is required"
  if [[ 10#"$(kind --version 2>&1 | awk '{print $3}' | tr -d '.')" -lt 10#070 ]]; then
    echo "kind 0.7.0+ is required" && exit 1
  fi

  # Ensure jq 1.3+ is available.
  command -v jq >/dev/null 2>&1 || fatal "jq 1.3+ is required"
  if [[ "$(jq --version 2>&1 | awk -F- '{print $2}' | tr -d '.')" -lt 13 ]]; then
    echo "jq 1.3+ is required" && exit 1
  fi

  # Require kustomize and kubectl as well.
  command -v kubectl >/dev/null 2>&1 || fatal "kubectl is required"
  command -v kustomize >/dev/null 2>&1 || fatal "kustomize is required"
  command -v docker >/dev/null 2>&1 || fatal "docker is required"
  command -v ytt >/dev/null 2>&1 || fatal "ytt is required"
}

################################################################################
##                                   funcs
################################################################################

# error stores exit code, writes arguments to STDERR, and returns stored exit code
# fatal is like error except it will exit program if exit code >0
function error() {
  local exit_code="${?}"
  echo "${@}" 1>&2
  return "${exit_code}"
}
function fatal() { error "${@}" || exit "${?}"; }

# base64d decodes STDIN from base64
# safe for linux and darwin
function base64d() { base64 -D 2>/dev/null || base64 -d; }

# safe for linux and darwin
function base64e() { base64 -w0 2>/dev/null || base64; }

function md5safe() { md5sum 2>/dev/null || md5; }

# base64e encodes STDIN to base64

# kubectl_mgc executes kubectl against management cluster.
function kubectl_mgc() { kubectl --context "kind-${KIND_MANAGEMENT_CLUSTER}" "${@}"; }

function e2e_up() {
  check_dependencies

  # Create the management cluster.
  if ! kind get clusters 2>/dev/null | grep -q "${KIND_MANAGEMENT_CLUSTER}"; then
    echo "creating kind management cluster ${KIND_MANAGEMENT_CLUSTER}"
    kind create cluster --config "${SCRIPTS_DIR}/kind/kind-cluster-with-extramounts.yaml" --name "${KIND_MANAGEMENT_CLUSTER}"
  fi

  # Install and wait for the cert manager webhook service to become available
  kubectl_mgc apply -f https://github.com/jetstack/cert-manager/releases/download/v0.11.0/cert-manager.yaml
  kubectl_mgc wait --for=condition=Available --timeout=300s apiservice v1beta1.webhook.cert-manager.io

  # Install Cluster API
  kubectl_mgc apply -f https://github.com/kubernetes-sigs/cluster-api/releases/download/${CAPI_VERSION}/cluster-api-components.yaml
  kubectl_mgc apply -f https://github.com/kubernetes-sigs/cluster-api/releases/download/${CAPI_VERSION}/bootstrap-components.yaml

  # Enable the ClusterResourceSet feature in cluster API
  kubectl_mgc -n capi-webhook-system patch deployment capi-controller-manager \
    --type=strategic --patch="$(
      cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-addr=127.0.0.1:8080
        - --webhook-port=9443
        - --feature-gates=ClusterResourceSet=true,MachinePool=false
EOF
    )"

  kubectl_mgc -n capi-system patch deployment capi-controller-manager \
    --type=strategic --patch="$(
      cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-addr=127.0.0.1:8080
        - --enable-leader-election
        - --feature-gates=ClusterResourceSet=true,MachinePool=false
EOF
    )"

  kubectl_mgc -n capi-webhook-system patch deployment capi-kubeadm-bootstrap-controller-manager \
    --type=strategic --patch="$(
      cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-addr=127.0.0.1:8080
        - --webhook-port=9443
        - --feature-gates=ClusterResourceSet=true,MachinePool=false
EOF
    )"

  kubectl_mgc -n capi-kubeadm-bootstrap-system patch deployment capi-kubeadm-bootstrap-controller-manager \
    --type=strategic --patch="$(
      cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - --metrics-addr=127.0.0.1:8080
        - --enable-leader-election
        - --feature-gates=ClusterResourceSet=true,MachinePool=false
EOF
    )"

  kubectl_mgc wait --for=condition=Available --timeout=300s deployment/capi-controller-manager -n capi-system
  kubectl_mgc wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-bootstrap-controller-manager -n capi-kubeadm-bootstrap-system
  kubectl_mgc wait --for=condition=Available --timeout=300s deployment/capi-kubeadm-control-plane-controller-manager -n capi-kubeadm-control-plane-system

  # replace the default image name to our desired image name.
  kustomize build "https://github.com/kubernetes-sigs/cluster-api/test/infrastructure/docker/config/?ref=${CAPI_VERSION}" |
    sed 's~'"${CAPD_DEFAULT_IMAGE}"'~'"${CAPD_IMAGE}"'~g' |
    kubectl_mgc apply -f -

  kubectl_mgc -n capd-system patch deployment capd-controller-manager \
    --type=strategic --patch="$(
      cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - -v=4
        - --metrics-addr=127.0.0.1:8080
        - --feature-gates=ClusterResourceSet=true,MachinePool=false
EOF
    )"

  kubectl_mgc wait --for=condition=Available --timeout=300s deployment/capd-controller-manager -n capd-system

  local clusters
  IFS=' ' read -r -a clusters <<<"${WORKLOAD_CLUSTERS}"

  for cluster in "${clusters[@]}"; do
    create_cluster "${cluster}"
    kubectl_mgc label cluster "${cluster}" cluster-role.tkg.tanzu.vmware.com/workload= --overwrite
  done

  kind get kubeconfig --name "${KIND_MANAGEMENT_CLUSTER}" > "${ROOT_DIR}/${KIND_MANAGEMENT_CLUSTER}.kubeconfig"
  cat <<EOF
################################################################################
cluster artifacts:
  name: ${KIND_MANAGEMENT_CLUSTER}
  kubeconfig: ${KIND_MANAGEMENT_CLUSTER}.kubeconfig
EOF

  for cluster in "${clusters[@]}"; do
    cat <<EOF
################################################################################
cluster artifacts:
  name: ${cluster}
  manifest: ${ROOT_DIR}/${cluster}.yaml
  kubeconfig: ${ROOT_DIR}/${cluster}.kubeconfig
EOF
  done

  return 0
}

function e2e_down() {
  # clean up CAPD clusters
  if [[ -z "${SKIP_CLEANUP_CAPD_CLUSTERS}" ]]; then
    # our management cluster has to be available to cleanup CAPD
    # clusters.
    local clusters
    IFS=' ' read -r -a clusters <<<"${WORKLOAD_CLUSTERS}"

    for cluster in "${clusters[@]}"; do
      # ignore status
      kind delete cluster --name "${cluster}" ||
        echo "cluster ${cluster} deleted."
      docker rm -f "${cluster}-lb" ||
        echo "cluster ${cluster} lb deleted."
      rm -fv "${ROOT_DIR}/${cluster}".kubeconfig 2>&1 ||
        echo "${ROOT_DIR}/${cluster}.kubeconfig deleted"
    done
  fi
  # clean up kind cluster
  if [[ -z "${SKIP_CLEANUP_MGMT_CLUSTER}" ]]; then
    # ignore status
    kind delete cluster --name "${KIND_MANAGEMENT_CLUSTER}" ||
      echo "kind cluster ${KIND_MANAGEMENT_CLUSTER} deleted."
  fi
  return 0
}

function create_cluster() {
  local simple_cluster_yaml="./hack/kind/simple-cluster.yaml"
  local clustername=$1

  local kubeconfig_path="${ROOT_DIR}/${clustername}.kubeconfig"

  local ip_addr="127.0.0.1"

  # Create per cluster deployment manifest, replace the original resource
  # names with our desired names, parameterized by clustername.
  if ! kind get clusters 2>/dev/null | grep -q "${clustername}"; then
    cat ${simple_cluster_yaml} |
      sed -e 's~my-cluster~'"${clustername}"'~g' \
       -e 's~controlplane-0~'"${clustername}"'-controlplane-0~g' \
       -e 's~worker-0~'"${clustername}"'-worker-0~g' |
      kubectl_mgc apply -f -
  fi
  while ! kubectl_mgc get secret "${clustername}"-kubeconfig; do
	  sleep 5s
  done

  kubectl_mgc get secret "${clustername}"-kubeconfig -o json | jq -cr '.data.value' | base64d >"${kubeconfig_path}"

  # Do not quote clusterkubectl when using it to allow for the correct
  # expansion.
  clusterkubectl="kubectl --kubeconfig=${kubeconfig_path}"

  # Get the API server port for the cluster.
  local api_server_port
  api_server_port="$(docker port "${clustername}"-lb 6443/tcp | cut -d ':' -f 2)"

  # We need to patch the kubeconfig fetched from CAPD:
  #   1. replace the cluster IP with host IP, 6443 with LB POD port;
  #   2. disable SSL by removing the CA and setting insecure to true;
  # Note: we're assuming the lb pod is ready at this moment. If it's not,
  # we'll error out because of the script global settings.
  ${clusterkubectl} config set clusters."${clustername}".server "https://${ip_addr}:${api_server_port}"
  ${clusterkubectl} config unset clusters."${clustername}".certificate-authority-data
  ${clusterkubectl} config set clusters."${clustername}".insecure-skip-tls-verify true

  # Ensure CAPD cluster lb and control plane being ready by querying nodes.
  while ! ${clusterkubectl} get nodes; do
    sleep 5s
  done

  # Deploy Calico cni into CAPD cluster.
  ${clusterkubectl} apply -f https://docs.projectcalico.org/v3.8/manifests/calico.yaml
  # Patch Calico in CAPD cluster to work around a kind issue.
  ${clusterkubectl} -n kube-system patch daemonset calico-node \
    --type=strategic --patch="$(
      cat <<EOF
spec:
  template:
    spec:
      containers:
      - name: calico-node
        env:
        - name: FELIX_IGNORELOOSERPF
          value: "true"
EOF
    )"

  # Wait until every node is in Ready condition.
  for node in $(${clusterkubectl} get nodes -o json | jq -cr '.items[].metadata.name'); do
    ${clusterkubectl} wait --for=condition=Ready --timeout=300s node/"${node}"
  done

  # Wait until every machine has ExternalIP in status
  for machine in $(kubectl_mgc get machine -l "cluster.x-k8s.io/cluster-name=${clustername}" -o json | jq -cr '.items[].metadata.name'); do
	  while [[ -z $(kubectl_mgc get machine "${machine}" -o json -o=jsonpath='{.status.addresses[?(@.type=="ExternalIP")].address}') ]]; do
		  sleep 5s;
	  done
  done
}

################################################################################
##                                   main
################################################################################

# Parse the command-line arguments.
while getopts ":hud" opt; do
  case ${opt} in
    h)
      error "${usage}" && exit 1
      ;;
    u)
      E2E_UP="1"
      ;;
    d)
      E2E_DOWN="1"
      ;;
    \?)
      error "invalid option: -${OPTARG} ${usage}" && exit 1
      ;;
    :)
      error "option -${OPTARG} requires an argument" && exit 1
      ;;
  esac
done
shift $((OPTIND - 1))

[[ ! "${#}" -eq "0" ]] && fatal "$(
  cat <<EOF
invalid option: $@
${usage}
EOF
)"

[[ -n "${E2E_UP}" ]] && e2e_up
[[ -n "${E2E_DOWN}" ]] && e2e_down

exit 0

