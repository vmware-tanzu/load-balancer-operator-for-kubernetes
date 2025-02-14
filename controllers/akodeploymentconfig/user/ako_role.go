// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"github.com/go-logr/logr"
	"github.com/vmware/alb-sdk/go/models"
	"golang.org/x/mod/semver"
	"k8s.io/utils/ptr"
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
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_VIRTUALSERVICE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_POOL"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_POOLGROUP"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_HTTPPOLICYSET"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_NETWORKSECURITYPOLICY"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_AUTOSCALE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_DNSPOLICY"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_NETWORKPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_APPLICATIONPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_APPLICATIONPERSISTENCEPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_HEALTHMONITOR"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_ANALYTICSPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_IPAMDNSPROVIDERPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_CUSTOMIPAMDNSPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_TRAFFICCLONEPROFILE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_IPADDRGROUP"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_STRINGGROUP"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_VSDATASCRIPTSET"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_PROTOCOLPARSER"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_SSLPROFILE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_AUTHPROFILE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_PINGACCESSAGENT"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_PKIPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_SSLKEYANDCERTIFICATE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_CERTIFICATEMANAGEMENTPROFILE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_HARDWARESECURITYMODULEGROUP"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_SSOPOLICY"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_NATPOLICY"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_WAFPROFILE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_WAFPOLICY"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_WAFPOLICYPSMGROUP"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ERRORPAGEPROFILE"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ERRORPAGEBODY"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ALERTCONFIG"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ALERT"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ACTIONGROUPCONFIG"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ALERTSYSLOGCONFIG"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ALERTEMAILCONFIG"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_SNMPTRAPPROFILE"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_TRAFFIC_CAPTURE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_CLOUD"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_SERVICEENGINE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_SERVICEENGINEGROUP"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_NETWORK"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_VRFCONTEXT"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_USER_CREDENTIAL"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_SYSTEMCONFIGURATION"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_CONTROLLER"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_REBOOT"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_UPGRADE"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_TECHSUPPORT"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_INTERNAL"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_CONTROLLERSITE"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_IMAGE"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_USER"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_ROLE"),
	},
	{
		Type:     ptr.To("READ_ACCESS"),
		Resource: ptr.To("PERMISSION_TENANT"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_GSLB"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_GSLBSERVICE"),
	},
	{
		Type:     ptr.To("NO_ACCESS"),
		Resource: ptr.To("PERMISSION_GSLBGEODBPROFILE"),
	},
	{
		Type:     ptr.To("WRITE_ACCESS"),
		Resource: ptr.To("PERMISSION_L4POLICYSET"),
	},
}

var deprecatePermissionMap = map[string]string{
	"PERMISSION_PINGACCESSAGENT": "v30.2.1",
}

func filterAkoRolePermissionByVersion(log logr.Logger, permissions []*models.Permission, version string) []*models.Permission {
	filtered := []*models.Permission{}
	for _, permission := range permissions {
		if v, ok := deprecatePermissionMap[*permission.Resource]; ok && semver.Compare(version, v) >= 0 {
			log.Info("Skip deprecated permission", "permission", *permission.Resource)
			// Skip deprecated permission
			continue
		}

		filtered = append(filtered, permission)

	}
	return filtered
}
