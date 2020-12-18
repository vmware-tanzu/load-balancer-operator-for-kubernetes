#!/bin/bash

# Copyright (c) 2020 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -o errexit  # Exits immediately on unexpected errors (does not bypass traps)
set -o nounset  # Errors if variables are used without first being defined
set -o pipefail # Non-zero exit codes in piped commands causes pipeline to fail
                # with that code

function sha256 {
  local cmd

  if command -v sha256sum &> /dev/null; then
    cmd=(sha256sum)
  elif command -v shasum &> /dev/null; then
    cmd=(shasum -a 256)
  else
    echo "ERROR: could not find shasum or sha256sum."
    return 1
  fi

  "${cmd[@]}" "$@"
}

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
ROOT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." >/dev/null 2>&1 && pwd )"

echo "***** Publishing manifests deliverable to '${MANIFESTS_PUBLISH_DIR}'..."
mkdir -p ${MANIFESTS_PUBLISH_DIR}

MANIFESTS_TAR_FILE_NAME="ako-operator-manifests-${VERSION}.tar.gz"
MANIFESTS_TAR_FILE="${MANIFESTS_PUBLISH_DIR}/${MANIFESTS_TAR_FILE_NAME}"
CHECKSUM_FILENAME="${MANIFESTS_TAR_FILE_NAME}-checksums.txt"
GPG_FILENAME="${MANIFESTS_TAR_FILE_NAME}-checksums.txt.asc"

tar -czf "${MANIFESTS_TAR_FILE}" "manifests"
  
cd "${MANIFESTS_PUBLISH_DIR}"
sha256 "${MANIFESTS_TAR_FILE}" > "${CHECKSUM_FILENAME}"

gpgsignc textsign -i "${CHECKSUM_FILENAME}" \
  -o "${GPG_FILENAME}" \
  --hash=sha256 --keyid="001E5CC9"


