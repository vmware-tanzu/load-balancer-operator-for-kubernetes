#!/bin/bash
# Copyright 2021 VMware, Inc.
# SPDX-License-Identifier: Apache-2.0

# This script is used to prepare gobuild slave using
# template linux-centos72-gc32 to install docker.

# This is a stop gap measure as mentioned in
# this bug: http://go/docker-on-build

# A more enhanced and easy to use functionality is being
# worked on through docker project type in Gobuild
# YAML interface. Read more here: http://go/docker-uru-yaml
DOCKER_STORAGE_DIR="${BUILDDIR}/docker"
DOCKER_TMPDIR="${DOCKER_STORAGE_DIR}/tmp"

set -o errexit
set -o nounset
set -o pipefail

rm -rf /etc/yum.repos.d/*
ls -al /etc/yum.repos.d/
echo "Removed RDOPS Repo."
yum remove -y docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-selinux  docker-engine-selinux docker-engine
echo "Removed old docker"

function add_to_yum {
    local artifactory_base="https://build-artifactory.eng.vmware.com/artifactory/"
    local repo_path="${1}"
    yum-config-manager --add-repo $artifactory_base$repo_path;
}

rpm --import https://build-artifactory.eng.vmware.com/artifactory/download.docker.com/linux/centos/gpg
echo "Imported Artifactory GPG key"

add_to_yum "download.docker.com/linux/centos/7/x86_64/stable"

for repo_type in extras os updates;
do
    add_to_yum centos-remote/7/$repo_type/x86_64
done
echo "Added yum repos from artifactory."

echo "Installing docker-ce dependencies."
yum install -y yum-utils device-mapper-persistent-data lvm2

# Installing docker
DOCKER_VER="18.09.5"
yum install -y docker-ce-$DOCKER_VER docker-ce-cli-$DOCKER_VER
#groupadd docker
/usr/sbin/usermod -aG docker mts
/usr/sbin/service docker restart
chown mts:docker /var/run/docker.sock
chown -R mts:docker /var/lib/docker
echo "Installing Docker Version $DOCKER_VER complete."

# The recommended docker driver is overlay
# The default storage directory for docker on the build vm is /var/lib/docker
# which is on the root partition. Since the storage on this partition is limited
# we move the storage directory under ${BUILDROOT} which is 100G in total space.
mkdir -m777 -p ${DOCKER_TMPDIR}
# TODO: do not edit docker.service with sed. This is being tracked here: https://jira.eng.vmware.com/browse/DS-1230
sed -i '/ExecStart/iexport DOCKER_TMP='$DOCKER_TMPDIR'' /lib/systemd/system/docker.service
sed -i 's#ExecStart=/usr/bin/dockerd#ExecStart=/usr/bin/dockerd -g '"$DOCKER_STORAGE_DIR"'#g' /lib/systemd/system/docker.service

mkdir /etc/systemd/system/docker.service.d
cat>/etc/systemd/system/docker.service.d/docker.conf<<EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd
EOF

systemctl daemon-reload
/usr/sbin/service docker restart
