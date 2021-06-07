// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"github.com/avinetworks/sdk/go/models"
	"k8s.io/utils/pointer"
)

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
		Type:     pointer.StringPtr("NO_ACCESS"),
		Resource: pointer.StringPtr("PERMISSION_SYSTEMCONFIGURATION"),
	},
	{
		Type:     pointer.StringPtr("NO_ACCESS"),
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
