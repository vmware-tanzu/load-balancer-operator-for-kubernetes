// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"github.com/vmware/alb-sdk/go/models"
	"k8s.io/utils/pointer"
)

var AkoRolePermissionMap = map[string]string{
	"PERMISSION_VIRTUALSERVICE":                "WRITE_ACCESS",
	"PERMISSION_POOL":                          "WRITE_ACCESS",
	"PERMISSION_POOLGROUP":                     "WRITE_ACCESS",
	"PERMISSION_HTTPPOLICYSET":                 "WRITE_ACCESS",
	"PERMISSION_NETWORKSECURITYPOLICY":         "WRITE_ACCESS",
	"PERMISSION_AUTOSCALE":                     "WRITE_ACCESS",
	"PERMISSION_DNSPOLICY":                     "WRITE_ACCESS",
	"PERMISSION_NETWORKPROFILE":                "WRITE_ACCESS",
	"PERMISSION_APPLICATIONPROFILE":            "WRITE_ACCESS",
	"PERMISSION_APPLICATIONPERSISTENCEPROFILE": "WRITE_ACCESS",
	"PERMISSION_HEALTHMONITOR":                 "WRITE_ACCESS",
	"PERMISSION_ANALYTICSPROFILE":              "WRITE_ACCESS",
	"PERMISSION_IPAMDNSPROVIDERPROFILE":        "WRITE_ACCESS",
	"PERMISSION_CUSTOMIPAMDNSPROFILE":          "WRITE_ACCESS",
	"PERMISSION_TRAFFICCLONEPROFILE":           "WRITE_ACCESS",
	"PERMISSION_VSDATASCRIPTSET":               "WRITE_ACCESS",
	"PERMISSION_PKIPROFILE":                    "WRITE_ACCESS",
	"PERMISSION_SSLKEYANDCERTIFICATE":          "WRITE_ACCESS",
	"PERMISSION_SERVICEENGINEGROUP":            "WRITE_ACCESS",
	"PERMISSION_NETWORK":                       "WRITE_ACCESS",
	"PERMISSION_VRFCONTEXT":                    "WRITE_ACCESS",
	"PERMISSION_L4POLICYSET":                   "WRITE_ACCESS",

	"PERMISSION_IPADDRGROUP":                  "READ_ACCESS",
	"PERMISSION_STRINGGROUP":                  "READ_ACCESS",
	"PERMISSION_PROTOCOLPARSER":               "READ_ACCESS",
	"PERMISSION_SSLPROFILE":                   "READ_ACCESS",
	"PERMISSION_AUTHPROFILE":                  "READ_ACCESS",
	"PERMISSION_PINGACCESSAGENT":              "READ_ACCESS",
	"PERMISSION_CERTIFICATEMANAGEMENTPROFILE": "READ_ACCESS",
	"PERMISSION_HARDWARESECURITYMODULEGROUP":  "READ_ACCESS",
	"PERMISSION_SSOPOLICY":                    "READ_ACCESS",
	"PERMISSION_WAFPROFILE":                   "READ_ACCESS",
	"PERMISSION_WAFPOLICY":                    "READ_ACCESS",
	"PERMISSION_CLOUD":                        "READ_ACCESS",
	"PERMISSION_SYSTEMCONFIGURATION":          "READ_ACCESS",
	"PERMISSION_CONTROLLER":                   "READ_ACCESS",
	"PERMISSION_TENANT":                       "READ_ACCESS",

	"PERMISSION_NATPOLICY":         "NO_ACCESS",
	"PERMISSION_WAFPOLICYPSMGROUP": "NO_ACCESS",
	"PERMISSION_ERRORPAGEPROFILE":  "NO_ACCESS",
	"PERMISSION_ERRORPAGEBODY":     "NO_ACCESS",
	"PERMISSION_ALERTCONFIG":       "NO_ACCESS",
	"PERMISSION_ALERT":             "NO_ACCESS",
	"PERMISSION_ACTIONGROUPCONFIG": "NO_ACCESS",
	"PERMISSION_ALERTSYSLOGCONFIG": "NO_ACCESS",
	"PERMISSION_ALERTEMAILCONFIG":  "NO_ACCESS",
	"PERMISSION_SNMPTRAPPROFILE":   "NO_ACCESS",
	"PERMISSION_TRAFFIC_CAPTURE":   "NO_ACCESS",
	"PERMISSION_SERVICEENGINE":     "NO_ACCESS",
	"PERMISSION_USER_CREDENTIAL":   "NO_ACCESS",
	"PERMISSION_REBOOT":            "NO_ACCESS",
	"PERMISSION_UPGRADE":           "NO_ACCESS",
	"PERMISSION_TECHSUPPORT":       "NO_ACCESS",
	"PERMISSION_INTERNAL":          "NO_ACCESS",
	"PERMISSION_CONTROLLERSITE":    "NO_ACCESS",
	"PERMISSION_IMAGE":             "NO_ACCESS",
	"PERMISSION_USER":              "NO_ACCESS",
	"PERMISSION_ROLE":              "NO_ACCESS",
	"PERMISSION_GSLB":              "NO_ACCESS",
	"PERMISSION_GSLBSERVICE":       "NO_ACCESS",
	"PERMISSION_GSLBGEODBPROFILE":  "NO_ACCESS",
}

var AkoRolePermission = []*models.Permission{
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_VIRTUALSERVICE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_POOL"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_POOLGROUP"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_HTTPPOLICYSET"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_NETWORKSECURITYPOLICY"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_AUTOSCALE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_DNSPOLICY"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_NETWORKPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_APPLICATIONPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_APPLICATIONPERSISTENCEPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_HEALTHMONITOR"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ANALYTICSPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_IPAMDNSPROVIDERPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_CUSTOMIPAMDNSPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_TRAFFICCLONEPROFILE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_IPADDRGROUP"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_STRINGGROUP"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_VSDATASCRIPTSET"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_PROTOCOLPARSER"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SSLPROFILE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_AUTHPROFILE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_PINGACCESSAGENT"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_PKIPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SSLKEYANDCERTIFICATE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_CERTIFICATEMANAGEMENTPROFILE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_HARDWARESECURITYMODULEGROUP"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SSOPOLICY"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_NATPOLICY"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_WAFPROFILE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_WAFPOLICY"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_WAFPOLICYPSMGROUP"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ERRORPAGEPROFILE"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ERRORPAGEBODY"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ALERTCONFIG"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ALERT"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ACTIONGROUPCONFIG"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ALERTSYSLOGCONFIG"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ALERTEMAILCONFIG"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SNMPTRAPPROFILE"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_TRAFFIC_CAPTURE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_CLOUD"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SERVICEENGINE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SERVICEENGINEGROUP"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_NETWORK"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_VRFCONTEXT"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_USER_CREDENTIAL"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SYSTEMCONFIGURATION"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_CONTROLLER"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_REBOOT"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_UPGRADE"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_TECHSUPPORT"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_INTERNAL"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_CONTROLLERSITE"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_IMAGE"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_USER"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_ROLE"),
	},
	{
		Type:     pointer.StringPtr("READ_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_TENANT"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_GSLB"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_GSLBSERVICE"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_GSLBGEODBPROFILE"),
	},
	{
		Type:     pointer.StringPtr("WRITE_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_L4POLICYSET"),
	},
}
