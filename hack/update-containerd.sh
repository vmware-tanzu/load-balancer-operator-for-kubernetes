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

usage="$(
  cat <<EOF
usage: bash hack/update-container.sh [docker image id]
EOF
)"

function error() {
  local exit_code="${?}"
  echo "${@}" 1>&2
  return "${exit_code}"
}
function fatal() { error "${@}" || exit "${?}"; }

[[ ! "${#}" -eq "1" ]] && fatal "$(
  cat <<EOF
${usage}
EOF
)" 

CONTAINERD_CONFIG="$(< hack/containerd/config.toml)"

# Disable shellcheck SC2086 as ${1} is explicitly used. Adding quotes would make
# the command fail
# shellcheck disable=SC2086
docker exec ${1} sh -c "cat > /etc/containerd/config.toml <<EOF
${CONTAINERD_CONFIG}
EOF
"

# Disable shellcheck SC2086 as ${1} is explicitly used. Adding quotes would make
# the command fail
# shellcheck disable=SC2086
docker exec ${1} sh -c "cat /etc/containerd/config.toml"

# Disable shellcheck SC2086 as ${1} is explicitly used. Adding quotes would make
# the command fail
# shellcheck disable=SC2086
docker exec ${1} sh -c "systemctl restart containerd"

exit 0
