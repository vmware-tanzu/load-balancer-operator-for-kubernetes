#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

DEPLOY_TKG_ENABLED="$1"

if [[ "${DEPLOY_TKG_ENABLED}" == "false" ]]; then
	echo "Skip deploying TKG"
	return
fi

export GOVC_URL="$(cat testbed/testbedInfo.json | jq -cr '.vc[0].ip')"
export ESXI_HOSTS="$(cat testbed/testbedInfo.json| jq -cr '.esx[].ip' | tr '\n' ' ' | sed 's/ $//')"

TESTBEDNAME=$(cat testbed/testbedInfo.json | jq '.name' | sed "s/['\"]//g")

mkdir -p ~/.nimbusctl/"${TESTBEDNAME}"

# Add a placeholder task to skip the provision
echo "gulel-nimbus-f2xd7" >> ~/.nimbusctl/"${TESTBEDNAME}"

./tkg_e2e -e "${TESTBEDNAME}" -u

cp ~/.nimbusctl/"${TESTBEDNAME}"/config.yaml ./config.yaml
cp ~/.kube-tkg/config ./kubeconfig
