# Copyright 2020 VMware, Inc.  All rights reserved. -- VMware Confidential

# Cayman GO component
# Change the branch and CLN to FIPS compliant `cayman_go` repository
# https://gitlab.eng.vmware.com/core-build/cayman_go
# Ensure any updates to this branch in future should use `boringcrypto`
# branch of GO
CAYMAN_GO_BRANCH = 'vmware-go1.15.0-boringcrypto'
CAYMAN_GO_CLN = 'd037af11cfdcae3e311de155c3097d1378f423e6'
CAYMAN_GO_BUILDTYPE = 'release'
CAYMAN_GO_HOSTTYPES = {
    'linux64': 'linux64',
    'linux-centos72-gc32': 'linux64',
    'linux-centos72-gc32-fw': 'linux64',
}
