// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package aviclient

import (
	"github.com/vmware/alb-sdk/go/models"
	"github.com/vmware/alb-sdk/go/session"
)

type Client interface {
	ServiceEngineGroupGetByName(name string, options ...session.ApiOptionsParams) (*models.ServiceEngineGroup, error)

	NetworkGetByName(name string, options ...session.ApiOptionsParams) (*models.Network, error)
	NetworkUpdate(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error)

	CloudGetByName(name string, options ...session.ApiOptionsParams) (*models.Cloud, error)

	UserGetByName(name string, options ...session.ApiOptionsParams) (*models.User, error)
	UserDeleteByName(name string, options ...session.ApiOptionsParams) error
	UserCreate(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error)
	UserUpdate(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error)

	TenantGet(uuid string, options ...session.ApiOptionsParams) (*models.Tenant, error)

	RoleGetByName(name string, options ...session.ApiOptionsParams) (*models.Role, error)
	RoleCreate(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error)

	IPAMDNSProviderProfileGet(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error)
	IPAMDNSProviderProfileUpdate(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error)

	VirtualServiceGetByName(name string, options ...session.ApiOptionsParams) (*models.VirtualService, error)

	PoolGetByName(name string, options ...session.ApiOptionsParams) (*models.Pool, error)

	AviCertificateConfig() (string, error)
}
