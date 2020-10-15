#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

sudo rpm -qa | grep -qw jq || sudo yum -y install jq