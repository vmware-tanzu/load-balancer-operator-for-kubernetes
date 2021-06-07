#!/bin/bash
# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

# Copyright (c) 2020 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# This script was ported over from the core-build/tkg-connectivity repo

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

# Change directories to the parent directory of the one in which this script is
# located.
cd "$(dirname "${BASH_SOURCE[0]}")/../.."

echo "***** Publishing build deliverable to '${IMAGE_PUBLISH_DIR}'..."
mkdir -p ${IMAGE_PUBLISH_DIR}

VERSION=${VERSION//+/_}

DIGEST_FILENAME="ako-operator-${VERSION}-image-digests.txt"
CHECKSUM_FILENAME="ako-operator-${VERSION}-image-checksums.txt"
GPG_FILENAME="ako-operator-${VERSION}-image-checksums.txt.asc"

export IMG=${IMAGE_REGISTRY}/tkg-networking/tanzu-ako-operator:${VERSION}

sudo docker save ${IMG} | gzip > ${IMAGE_PUBLISH_DIR}/ako-operator-${VERSION}.tar.gz

IMG_ID=$(sudo docker inspect -f '{{.ID}}' "${IMG}")
echo "${IMG}@${IMG_ID}" >> "${IMAGE_PUBLISH_DIR}/${DIGEST_FILENAME}"

echo ${VERSION} > ${PUBLISH_DIR}/VERSION

cd "${IMAGE_PUBLISH_DIR}"
rm -f "${CHECKSUM_FILENAME}"
sha256 -- * > "${CHECKSUM_FILENAME}"

gpgsignc textsign -i "${CHECKSUM_FILENAME}" \
  -o "${GPG_FILENAME}" \
  --hash=sha256 --keyid="001E5CC9"
