# Copyright (c) 2020 VMware, Inc.  All rights reserved.
# -- VMware Confidential

"""
ako-operator gobuild product module.
"""

import os
import helpers.env
import helpers.make
import helpers.target
import specs.ako_operator

AKOOPERATOR_REL_PATH = 'core-build/ako-operator'
AKOOPERATOR_SRC_PATH = 'gitlab.eng.vmware.com/' + AKOOPERATOR_REL_PATH

class AKOOperator(helpers.target.Target, helpers.make.MakeHelper):
    """
    AKO Operator Components
    """

    def GetBuildProductNames(self):
        return {'name':     'ako-operator',
                'longname': 'AKO Operator'}

    def GetClusterRequirements(self):
        # See http://machineweb.eng.vmware.com/label/all/
        return [
            'linux-centos72-gc32' # This has docker and isn't firewalled
        ]

    def GetRepositories(self, hosttype):
        return [{
            'rcs': 'git',
            'src': AKOOPERATOR_REL_PATH + ';%(branch);',
            'dst': 'home/mts/go/src/' + AKOOPERATOR_SRC_PATH
        }]

    def _GetEnvironment(self, hosttype):
        env = helpers.env.SafeEnvironment(hosttype)

        tcroot = os.environ.get('TCROOT', '/build/toolchain')
        paths = [os.path.join(tcroot, 'lin64', path)
                 for path in ['coreutils-5.97/bin',
                              'findutils-4.2.27/bin',
                              'grep-2.5.1a/bin',
                              'make-3.81/bin',
                              'cmake-2.8.10.2/bin',
                              'bash-4.1/bin',
                              'gawk-3.1.5/bin',
                              'sed-4.1.5/bin',
                              'tar-1.23/bin',
                              'gzip-1.5/bin',
                              'git-2.6.2/bin']]
        paths.append(env['PATH'])
        paths.append('/build/toolchain/noarch/vmware/gpgsign/')
        paths.append('/command')
        paths.append('/usr/local/bin')
        paths.append('/usr/local/sbin')
        paths.append('/bin')
        paths.append('/sbin')
        paths.append('/usr/bin')
        paths.append('/usr/sbin')
        paths.append('/usr/X11R6/bin')
        env['PATH'] = os.pathsep.join(paths)
        return env

    def GetCommands(self, hosttype):

        root = '%(buildroot)/home/mts/go/src/' + AKOOPERATOR_SRC_PATH
        makeflags = {}
        return [
		{
            'desc': 'Fetching unshallow git repo',
            'root': root,
            'log': 'fetch-unshallow.log',
            'env': self._GetEnvironment(hosttype),
            'command': self._Command(hosttype, 'gobuild-fetch-unshallow', **makeflags)
        },
        {
            'desc': 'Building ako operator container image',
            'root': root,
            'log': 'ako-operator.log',
            'env': self._GetEnvironment(hosttype),
            'command': self._Command(hosttype, 'gobuild', **makeflags)
        }]

    def GetStorageInfo(self, hosttype):
        return []

    def GetBuildProductVersion(self, hosttype):
		versionFile = os.path.join(self.options.get('buildroot'), 'publish', 'VERSION')
		with open(versionFile, 'r') as fp:
			version = fp.read().strip()
		return version

    def GetComponentPath(self):
        return '%(buildroot)/publish'

    def GetComponentDependencies(self):
        comps = {}
        comps['cayman_go'] = {
            'branch': specs.ako_operator.CAYMAN_GO_BRANCH,
            'change': specs.ako_operator.CAYMAN_GO_CLN,
            'buildtype': specs.ako_operator.CAYMAN_GO_BUILDTYPE,
            'hosttypes': specs.ako_operator.CAYMAN_GO_HOSTTYPES,
        }
        return comps
