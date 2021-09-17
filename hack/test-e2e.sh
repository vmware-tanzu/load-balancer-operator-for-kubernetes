#!/bin/bash
# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

# Copyright (c) 2020 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

################################################################################
# usage: e2e
#  This program deploys a local test environment using AKOO and Kind.
################################################################################

set -o errexit  # Exits immediately on unexpected errors (does not bypass traps)
set -o nounset  # Errors if variables are used without first being defined
set -o pipefail # Non-zero exit codes in piped commands causes pipeline to fail
                # with that code

# Bring up testing environment
hack/e2e.sh -u

# Set aliases for accessing both clusters
alias kk='kubectl --kubeconfig=$PWD/tkg-lcp.kubeconfig'
alias kw='kubectl --kubeconfig=$PWD/workload-cls.kubeconfig'

# Set a bash-specific shell option to expand aliases in shell scripts
shopt -s expand_aliases

# Set the default kubeconfig to the management cluster
export KUBECONFIG=$PWD/tkg-lcp.kubeconfig

# Build manager docker image
make docker-build

# Load manager docker image into kind cluster
kind load docker-image --name tkg-lcp harbor-pks.vmware.com/tkgextensions/tkg-networking/tanzu-ako-operator:dev

# Deploy the AKO Operator in the management cluster
make deploy || true

# Make sure AKO Operator is up and running
akooip=""
n=1
while [[ -z "${akooip}" && $n -le 10 ]]; do
  sleep 3s
  akooip="$(kk get pods -n akoo-system -o wide | grep '^akoo-.*Running' | grep -e '[0-9]*\.[0-9]*\.[0-9]*\.[0-9]' -o)" || true 
  n=$(( n+1 ))
done
if [ "$n" == "11" ];then
  echo "AKO Operator can't get ready"
  exit 1
else
  echo "AKO Operator is running at ${akooip}"
fi

# Enable AVI in the workload cluster
kk label cluster workload-cls cluster-service.network.tkg.tanzu.vmware.com/avi=""

# Making sure AKO is deployed into the workload cluster
akoip=""
n=1
while [[ -z "${akoip}" && $n -le 10 ]]; do
  sleep 5s
  akoip="$(kw get pods -n tkg-system-networking -o wide | grep '^ako-.*Running' | grep -e '[0-9]*\.[0-9]*\.[0-9]*\.[0-9]' -o)" || true
  n=$(( n+1 ))
done
if [ "$n" == "11" ];then
  echo "AKO can't get ready"
  exit 1
else
  echo "AKO is running at ${akoip}"
fi

# Making sure the configmap exists
configmap=""
n=1
while [[ -z "${configmap}" && $n -le 10 ]]; do
  configmap="$(kw get configmap -n tkg-system-networking | grep '^avi-k8s-config' -o)" || true
  sleep 3s
  n=$(( n+1 ))
done
if [ "$n" == "11" ];then
  echo "Configmap doesn't exists"
  exit 1
else
  echo "${configmap} exists"
fi

# Making sure AKO Operator adds the finalizer on the cluster
finalizer=""
n=1
while [[ -z "${finalizer}" && $n -le 10 ]]; do
  finalizer="$(kk get cluster workload-cls -o yaml | grep 'ako-operator.network.tkg.tanzu.vmware.com' -o | head -1)" || true 
  sleep 3s
  n=$(( n+1 ))
done
if [ "$n" == "11" ];then
  echo "Finalizer doesn't exists"
  exit 1
else
  echo "${finalizer} exists"
fi

# Making sure the pre-terminate hook is added to the workload cluster Machines
preTerminateHook=""
n=1
while [[ -z "${preTerminateHook}" && $n -le 10 ]]; do
  preTerminateHook="$(kk get machine -o yaml | grep terminate | grep 'pre-terminate\.delete\.hook.*ako-operator' -o | head -1)" || true 
  sleep 3s
  n=$(( n+1 ))
done
if [ "$n" == "11" ];then
  echo "Pre-terminate hook doesn't exists"
  exit 1
else
  echo "${preTerminateHook} exists"
fi

# Deleting the workload cluster
kk delete cluster workload-cls

# Making sure the workload cluster is deleted
n=1
while kk get cluster "workload-cls"; do
  sleep 5s
  n=$(( n+1 ))
  if [ $n -ge 11 ];then
    echo "Workload cluster delete failed"
    exit 1
  fi
done
echo "Workload cluster is deleted"

# Delete testing environment
hack/e2e.sh -d
