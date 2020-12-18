# Copyright (c) 2020 VMware, Inc.  All rights reserved. -- VMware Confidential

"""
Helpers for make-based targets.
"""

import os

import helpers.target


class MakeHelper:
    """
    Helper class for targets that build with make.

    Meant to be used as a mixin with helpers.target.Target. Provides a
    private method _Command() that helps to create a make command, as
    well as two private methods _GetStoreSourceRule() and
    _GetStoreBuildRule() that help to create storage rules.

       >>> import helpers.target
       >>> import helpers.make
       >>>
       >>> class HelloWorld(helpers.target.Target, helpers.make.MakeHelper):
       ...    def GetCommands(self, hosttype):
       ...       flags = {'PRODUCT': 'hello'}
       ...       return [ { 'desc'    : 'Running hello world sample',
       ...                  'root'    : '%(buildroot)',
       ...                  'log'     : 'helloworld.log',
       ...                  'command' : self._Command(hosttype,
                                                      helloworld,
                                                      **flags),
       ...                } ]
       ...    def GetStorageInfo(self, hosttype):
       ...       storinfo = []
       ...       if hosttype.startswith('linux'):
       ...          storinfo += self._GetStoreSourceRule('bora')
       ...       storinfo += self._GetStoreBuildRule(hosttype, 'bora')
       ...       return storinfo
       >>>
    """

    def _Command(self, hosttype, target, makeversion='3.81', **flags):
        """
        Return a dictionary representing a command to invoke make with
        standard makeflags.
        """

        def q(s):
            return '"%s"' % s

        defaults = {
            'GOBUILD_OFFICIAL_BUILD': '1',
            'GOBUILD_AUTO_COMPONENTS': '',
            'OBJDIR': q('%(buildtype)'),
            'RELTYPE': q('%(releasetype)'),
            'BUILD_NUMBER': q('%(buildnumber)'),
            'PRODUCT_BUILD_NUMBER': q('%(productbuildnumber)'),
            'CHANGE_NUMBER': q('%(changenumber)'),
            'BRANCH_NAME': q('%(branch)'),
            'PUBLISH_DIR': q('%(buildroot)/publish'),
            'REMOTE_COPY_SCRIPT': q('%(gobuildc) %(buildid)'),
        }
        # Handle verbosity
        if self.options.get('verbose'):
            defaults['VERBOSE'] = '3'

        # If the user has overridden the number of cpus to use for this
        # build, stick that on the command line, too.
        numcpus = self.options.get('numcpus')
        if numcpus:
            self.log.debug('Overriding num cpus (%s).' % numcpus)
            defaults['NUM_CPU'] = numcpus

        # Handle officialkey
        if 'officialkey' in self.options:
            if self.options.get('officialkey'):
                self.log.debug("Build will use official key")
                defaults['OFFICIALKEY'] = '1'
            else:
                self.log.debug("Build will not use official key")
                defaults['OFFICIALKEY'] = ''

        # Should we can for viruses?
        if hosttype.startswith('windows'):
            if self.options.get('virusscanner'):
                self.log.debug('Using virus scanner: %s' %
                               self.options['virusscanner'])
                defaults['VIRUS_SCAN'] = q(self.options['virusscanner'])

        # Add a GOBUILD_*_ROOT flag for every component we depend on.
        if hasattr(self, 'GetComponentDependencyAliases'):
            for d in self.GetComponentDependencyAliases():
                d = d.replace('-', '_')
                defaults['GOBUILD_%s_ROOT' % d.upper()] = \
                    '%%(gobuild_component_%s_root)' % d

        # Override the defaults above with the options passed in by
        # the client.
        defaults.update(flags)

        # Choose make
        if hosttype.startswith('linux'):
            makecmd = '/build/toolchain/lin32/make-%s/bin/make' % makeversion
        elif hosttype.startswith('windows'):
            tcroot = os.environ.get('TCROOT', 'C:/TCROOT-not-set')
            makecmd = '%s/win32/make-%s/make.exe' % (tcroot, makeversion)
        elif hosttype.startswith('mac'):
            makecmd = '/build/toolchain/mac32/make-%s/bin/make' % makeversion
        else:
            raise helpers.target.TargetException('unsupported hosttype: %s'
                                                 % hosttype)

        # Create the command line to invoke make
        cmd = '%s %s ' % (makecmd, target)
        for k in sorted(defaults.keys()):
            v = defaults[k]
            cmd += ' ' + str(k)
            if v is not None:
                cmd += '=' + str(v)

        return cmd

    def _DevCommand(self, hosttype, target, makeversion='3.81', **flags):
        """
        Return a dictionary representing a command to invoke make with
        standard makeflags.
        """

        def q(s):
            return '"%s"' % s

        defaults = {
            'OBJDIR':       q('%(buildtype)'),
            'RELTYPE':       q('%(releasetype)'),
            'BUILD_NUMBER':       q('%(buildnumber)'),
            'PRODUCT_BUILD_NUMBER':       q('%(productbuildnumber)'),
            'CHANGE_NUMBER':       q('%(changenumber)'),
            'BRANCH_NAME':       q('%(branch)'),
            'GOBUILD_AUTO_COMPONENTS_REQUEST': '1',
            'SHARED_BUILD_MACHINE': '1',
        }
        # Handle verbosity
        if self.options.get('verbose'):
            defaults['VERBOSE'] = '3'

        # If the user has overridden the number of cpus to use for this
        # build, stick that on the command line, too.
        numcpus = self.options.get('numcpus')
        if numcpus:
            self.log.debug('Overriding num cpus (%s).' % numcpus)
            defaults['NUM_CPU'] = numcpus

        # Override the defaults above with the options passed in by
        # the client.
        defaults.update(flags)

        # Choose make
        if hosttype.startswith('linux'):
            makecmd = '/build/toolchain/lin32/make-%s/bin/make' % makeversion
        elif hosttype.startswith('windows'):
            tcroot = os.environ.get('TCROOT', 'C:/TCROOT-not-set')
            makecmd = '%s/win32/make-%s/make.exe' % (tcroot, makeversion)
        elif hosttype.startswith('mac'):
            makecmd = '/build/toolchain/mac32/make-%s/bin/make' % makeversion
        else:
            raise helpers.target.TargetException('unsupported hosttype: %s'
                                                 % hosttype)

        # Create the command line to invoke make
        cmd = '%s %s ' % (makecmd, target)
        for k in sorted(defaults.keys()):
            v = defaults[k]
            cmd += ' ' + str(k)
            if v is not None:
                cmd += '=' + str(v)

        return cmd

    def _GetStoreSourceRule(self, tree):
        """
        Return the standard storage rules for a make based build.  The
        Linux side is responsible for copying the source files to storage.
        """
        return [{'type': 'source',
                 'src': '%s/' % tree
                 }]

    def _GetStoreBuildRule(self, hosttype, tree):
        """
        Return the standard storage rules for a make based build.  The
        Linux side is responsible for copying the source files to storage.
        """
        return [{'type': 'build',
                 'src': '%s/build' % (tree)
                 }]
