# Copyright (c) 2019 VMware, Inc. All rights reserved.
# -- VMware Confidential

"""
Support code for building products with gobuild.
"""
import helpers
import targets.ako_operator

# The targets dictionary maps target names to classes which
# implement the gobuild Target interface.  The dictionary is not
# strictly necessary, but aids greatly in the implementation of
# GetTarget and GetAllTargets below.  Be sure to remove the samples
# from this list when you add gobuild support to your product
# tree!

TARGETS = {
    'ako-operator': targets.ako_operator.AKOOperator
}

def GetTarget(log, name):
    """
    Return the Target class for the build identified by the target
    'name'.  The log parameter is a standard python logging object
    which can be used to write to the gobuild log if you like.
    """
    objtype = TARGETS.get(name)
    if not objtype:
        return None

    return objtype()


def GetAllTargets(log):
    """
    Return a list of all targets that can be built from this
    branch.
    """
    return TARGETS.keys()


def GetBranch(log):
    """
    Return the name of the branch.
    """
    return helpers.GetBranch()
