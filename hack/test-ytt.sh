#!/bin/bash

# Copyright (c) 2020 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -o errexit  # Exits immediately on unexpected errors (does not bypass traps)
set -o nounset  # Errors if variables are used without first being defined
set -o pipefail # Non-zero exit codes in piped commands causes pipeline to fail
                # with that code

# Change directories to the parent directory of the one in which this script is
# located.
cd "$(dirname "${BASH_SOURCE[0]}")/.."

export PATH=$PATH:$PWD/hack/tools/bin

export TEMPLATE_DIR="config/ytt/akodeploymentconfig"

if command -v tput &>/dev/null && tty -s; then
  RED=$(tput setaf 1)
  NORMAL=$(tput sgr0)
else
  RED=$(echo -en "\e[31m")
  NORMAL=$(echo -en "\e[00m")
fi

log_failure() {
  printf "${RED}✖ %s${NORMAL}\n" "$@" >&2
}

assert_eq() {
  local expected="$1"
  local actual="$2"
  local msg="${3-}"

  if [ "$expected" == "$actual" ]; then
    return 0
  else
    if [ "${#msg}" -gt 0 ]; then
      log_failure "$expected == $actual :: $msg" || true
    fi
    return 1
  fi
}

case1() {
  # Test the default AKODeploymentConfig template generation
  ytt -f config/ytt/akodeploymentconfig/values.yaml -f config/ytt/akodeploymentconfig/akodeploymentconfig.yaml >/dev/null 2>&1
}

case2() {
  # Test the ip pools section 
  res="$(ytt -f config/ytt/akodeploymentconfig/values.yaml -f config/ytt/akodeploymentconfig/akodeploymentconfig.yaml -v AVI_DATA_NETWORK_IP_POOL_START=10.0.0.2 -v AVI_DATA_NETWORK_IP_POOL_END=10.0.0.3 -o json 2>&1)"
  assert_eq "$(echo "${res}" | jq -cr 'select( .spec).spec.dataNetwork.ipPools[].start')" "10.0.0.2" "failed ipPools"
  assert_eq "$(echo "${res}" | jq -cr 'select( .spec).spec.dataNetwork.ipPools[].end')" "10.0.0.3" "failed ipPools"
}

case1
case2
