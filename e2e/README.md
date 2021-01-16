# How to run e2e test agains a testbed

1. Get a testbed from pipeline <https://kscom.svc.eng.vmware.com/job/tkg-cluster-with-avi>
1. After a successful build, download the artifacts
1. Unzip the package

```bash
# You should see something similar
➜ ls
archive     archive.zip

➜ cd archive
➜ ls
akodeploymentconfig.yaml      tkg-cluster-mc-113_config     tkgversion.txt log

➜ export ARCHIVE_PATH=$PWD
```

1. Copy the following files over

```bash
# Copy the management cluster kubeconfig
# Assume:
#  1. ARCHIVE_PATH environmental variable saves the absolute path to the
#     unzipped artifacts;
#  2. AKO_OPERATOR_PATH environmental variable saves the absolute path to the
#     ako operator repo;

cp ${ARCHIVE_PATH}/tkg-cluster-mc-[this matches your build number]_config ${AKO_OPERATOR_PATH}/e2e/static/mc.kubeconfig
cp ${ARCHIVE_PATH}/.tkg/config.yaml ${AKO_OPERATOR_PATH}/e2e/static/tkg-config.yaml
cp ${ARCHIVE_PATH}/akodeploymentconfig.yaml ${AKO_OPERATOR_PATH}/e2e/static/akodeploymentconfig.yaml
```

1. Update the tkg.regions.file in tkg config file

```bash
# 1. open ${AKO_OPERATOR_PATH}/e2e/static/tkg-config.yaml
# 2. find the `tkg` section similar to the following
tkg:
    regions:
      - name: tkg-cluster-mc-113
        context: tkg-cluster-mc-113-admin@tkg-cluster-mc-113
        file: [here!!!]
        status: Success
        isCurrentContext: false
    current-region-context: tkg-cluster-mc-113-admin@tkg-cluster-mc-113

# 3. change the value of file field to the absolute path, i.e: ${AKO_OPERATOR_PATH}/e2e/static/tkg-config.yaml
```

1. open ${AKO_OPERATOR_PATH}/e2e/env.json
    1. update env.mc-kubeconfig.context to the mc's context. Usually it's in the
     form of "tkg-cluster-mc-113-admin@tkg-cluster-mc-113"
    1. update env.worker to the testbed's worker ip, which can be found in
     ${ARCHIVE_PATH}/vc.txt. It's the IP used in STATIC_IP_SERVICE_ENDPOINT

1. run e2e test

```bash
hack/run-e2e.sh
```
