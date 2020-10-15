#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

NIMBUS_USER="$1"
ESX_BUILD="$2"
VC_BUILD="$3"
TESTBED="$4"
NUM_ESX="$5"
STATIC_IP_ENABLED="$6"
AVI_CONTROLLER_OVF_URL="$7"

# Use sc by default for Nimbus Pod choice
[[ -z "${NIMBUS_LOC:-}" ]] && NIMBUS_LOC=sc

# If NIMBUS_USER is not specified, fall back to fangyuanl
if [[ "${NIMBUS_USER}" = "null" || "${NIMBUS_USER}" = "timer" ]]; then
    NIMBUS_USER=fangyuanl
fi

testbeddir="testbed"
mkdir -p "${testbeddir}"
RAND=$(openssl rand -hex 6)

cat testbed_spec/"${TESTBED}.rb"
NIMBUS_CONTEXTS=nsx USER="${NIMBUS_USER}" NIMBUS_LOCATION=${NIMBUS_LOC} /mts/git/bin/nimbus-testbeddeploy \
  --testbedSpecRubyFile testbed_spec/"${TESTBED}.rb" \
  --runName "tkg-networking-avi-${RAND}" \
  --esxBuild "${ESX_BUILD}" \
  --vcenterBuild "${VC_BUILD}" \
  --esx-count "${NUM_ESX}" \
  --arg "static_ip_enabled:${STATIC_IP_ENABLED}" \
  --arg "avi_controller_ovf_url:${AVI_CONTROLLER_OVF_URL}" \
  --resultsDir "${testbeddir}"
