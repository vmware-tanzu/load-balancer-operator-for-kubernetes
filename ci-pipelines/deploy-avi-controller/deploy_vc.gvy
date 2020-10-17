#!groovy
// Copyright (c) 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

pipeline {
    agent {
        node {
            label 'nimbus-cloud'
            customWorkspace "${BUILD_ID}"
        }
    }

    parameters {
       string(description: '[Optional] esx build, default to 6.7 u3', name: 'ESX_BUILD', defaultValue: 'ob-14320389')
       string(description: '[Optional] vc build, default to 6.7 u3', name: 'VC_BUILD', defaultValue: 'ob-14367737')
       string(description: '[Optional] the testbed type (iscsi/vsan)', name: 'TESTBED', defaultValue: 'vsan')
       string(description: '[Optional] number of ESX in testbed', name: 'NUMESX', defaultValue: '3')
       string(description: '[Optional] Static IP Service', name: 'STATIC_IP_ENABLED', defaultValue: 'true')
       string(description: '[Optional] AVI Controller OVF URL', name: 'AVI_CONTROLLER_OVF_URL', defaultValue: 'http://sc-dbc1105.eng.vmware.com/fangyuanl/images/controller-20.1.2-9171.ovf')
       choice(
           name: 'NIMBUS_LOC',
           choices: ['wdc', 'sc,wdc', 'sc'],
           description: '[Optional] Specify which Nimbus datacenter location for deployment',
       )
    }


    stages {
        stage('deploy vCenter'){
            steps {
                script {
                  dir('akoo/ci-pipelines/deploy-avi-controller'){
                    def userid
                    wrap([$class: 'BuildUser']) {
                      userid = env.BUILD_USER_ID
                    }
                    sh './install_jq.sh'

                    sh "./deploy_vc.sh ${userid} ${ESX_BUILD} ${VC_BUILD} ${TESTBED} ${NUMESX} ${STATIC_IP_ENABLED} ${AVI_CONTROLLER_OVF_URL}"

                    sh "./get_vc_ip.sh ${STATIC_IP_ENABLED}"

                    archiveArtifacts artifacts: 'vc.txt', fingerprint: true
                  }
                }
            }
        }
    }
}
