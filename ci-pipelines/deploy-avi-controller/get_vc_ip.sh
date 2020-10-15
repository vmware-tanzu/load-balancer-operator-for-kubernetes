#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail
set -o xtrace

STATIC_IP_ENABLED="$1"

IP=$(cat testbed/testbedInfo.json | jq '.vc[0] .ip' | sed "s/['\"]//g")
echo "vCenter ip address: ${IP}"

TESTBEDNAME=$(cat testbed/testbedInfo.json | jq '.name' | sed "s/['\"]//g")
echo "testbed name: ${TESTBEDNAME}"

if [[ ${STATIC_IP_ENABLED} == "false" ]]; then
  cat << EOF > ./vc.txt
vcip=${IP}
testbedname=${TESTBEDNAME}
EOF
else
  STATICIP_WORKER_URL=$(cat testbed/testbedInfo.json | jq '.worker[0] .nsips' | sed "s/['\"]//g")
  CONTROL_PLANE_ENDPOINT_IP=$(curl -s ${STATICIP_WORKER_URL} | jq '.ip' | sed "s/['\"]//g")
cat << EOF > ./vc.txt
vcip=${IP}
testbedname=${TESTBEDNAME}
control_plane_endpoint_ip=${CONTROL_PLANE_ENDPOINT_IP}
STATIC_IP_SERVICE_ENDPOINT=${STATICIP_WORKER_URL}
EOF
fi

cat << EOF > ./vc_internal.txt
${IP}
EOF

