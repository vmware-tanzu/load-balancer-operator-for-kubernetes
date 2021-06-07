#!/bin/bash
# Copyright 2020 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

# This script was ported over from the core-build/tkg-connectivity repo

set -o errexit  # Exits immediately on unexpected errors (does not bypass traps)
set -o nounset  # Errors if variables are used without first being defined
set -o pipefail # Non-zero exit codes in piped commands causes pipeline to fail
                # with that code

# Change directories to the parent directory of the one in which this script is
# located.
cd "$(dirname "${BASH_SOURCE[0]}")/../.."

VERSION=${VERSION//+/_}

export IMG=${IMAGE_REGISTRY}/tkg-networking/tanzu-ako-operator:${VERSION}

git config --global url.git@gitlab.eng.vmware.com:.insteadof https://gitlab.eng.vmware.com/

# The docker installed on the build machine is older.
# In case of docker runtime, doing `docker pull` of image
# from registry and `docker push`, automatically converts the image to
# use latest image manifest specification. But with containerd runtime,
# `ctr push` to the docker registry fails because the image manifest
# version for images pushed using older version of docker is v2, schema
# 1 which is deprecated. Therefore `docker push` operation to push the
# container images on docker registry during build need to be performed
# with newer version of docker.
# Ref.: https://docs.docker.com/registry/spec/deprecated-schema-v1/

sudo sh ./hack/gobuild/install_docker.sh

sudo docker version

# Stop the docker service in case it's running.
sudo service docker stop

# Configure the docker daemon to use /build/docker-gobuild instead.
# This path does not exist on the build VM by default.
echo '{"graph":"/build/docker-gobuild"}' | sudo tee /etc/docker/daemon.json

# Now restart the docker service so we can build.
sudo service docker start

sudo docker build -f Dockerfile -t ${IMG} .

echo "***** successfully build ako operator docker images *****"
