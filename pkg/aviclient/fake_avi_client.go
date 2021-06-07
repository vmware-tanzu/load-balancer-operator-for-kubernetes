// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package aviclient

import (
	"github.com/avinetworks/sdk/go/models"
	"github.com/avinetworks/sdk/go/session"
)

// FakeAviClient -- an API Client for Avi Controller for intg test of AkoDeploymentConfig
type FakeAviClient struct {
	Network                *NetworkClient
	Cloud                  *CloudClient
	IPAMDNSProviderProfile *IPAMDNSProviderProfileClient
	User                   *UserClient
	Tenant                 *TenantClient
	Role                   *RoleClient
	VirtualService         *VirtualServiceClient
	Pool                   *PoolClient
}

func NewFakeAviClient() *FakeAviClient {
	return &FakeAviClient{
		Network:                &NetworkClient{},
		Cloud:                  &CloudClient{},
		IPAMDNSProviderProfile: &IPAMDNSProviderProfileClient{},
		User:                   &UserClient{},
		Tenant:                 &TenantClient{},
		Role:                   &RoleClient{},
	}
}

func (r *FakeAviClient) NetworkGetByName(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
	return r.Network.GetByName(name)
}

func (r *FakeAviClient) NetworkUpdate(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error) {
	return r.Network.Update(obj)
}

func (r *FakeAviClient) CloudGetByName(name string, options ...session.ApiOptionsParams) (*models.Cloud, error) {
	return r.Cloud.GetByName(name)
}

func (r *FakeAviClient) IPAMDNSProviderProfileGet(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
	return r.IPAMDNSProviderProfile.Get(uuid)
}

func (r *FakeAviClient) IPAMDNSProviderProfileUpdate(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
	return r.IPAMDNSProviderProfile.Update(obj)
}

func (r *FakeAviClient) UserGetByName(name string, options ...session.ApiOptionsParams) (*models.User, error) {
	return r.User.GetByName(name)
}
func (r *FakeAviClient) UserDeleteByName(name string, options ...session.ApiOptionsParams) error {
	return r.User.DeleteByName(name)
}

func (r *FakeAviClient) UserCreate(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
	return r.User.Create(obj)
}

func (r *FakeAviClient) UserUpdate(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
	return r.User.Update(obj)
}

func (r *FakeAviClient) TenantGet(uuid string, options ...session.ApiOptionsParams) (*models.Tenant, error) {
	return r.Tenant.Get(uuid)
}

func (r *FakeAviClient) RoleGetByName(name string, options ...session.ApiOptionsParams) (*models.Role, error) {
	return r.Role.GetByName(name)
}

func (r *FakeAviClient) RoleCreate(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error) {
	return r.Role.Create(obj)
}

func (r *FakeAviClient) VirtualServiceGetByName(name string, options ...session.ApiOptionsParams) (*models.VirtualService, error) {
	return r.VirtualService.GetByName(name)
}

func (r *FakeAviClient) PoolGetByName(name string, options ...session.ApiOptionsParams) (*models.Pool, error) {
	return r.Pool.GetByName(name)
}

// Network Client
type NetworkClient struct {
	getByNameFn GetByNameFunc
	updateFn    UpdateFn
}

type GetByNameFunc func(name string, options ...session.ApiOptionsParams) (*models.Network, error)
type UpdateFn func(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error)

func (client *NetworkClient) SetGetByNameFn(fn GetByNameFunc) {
	client.getByNameFn = fn
}

func (client *NetworkClient) SetUpdateFn(fn UpdateFn) {
	client.updateFn = fn
}

func (client *NetworkClient) GetByName(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
	return client.getByNameFn(name)
}

func (client *NetworkClient) Update(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error) {
	return client.updateFn(obj)
}

// Cloud Client
type CloudClient struct {
	getByNameCloudFn GetByNameCloudFunc
}

type GetByNameCloudFunc func(name string, options ...session.ApiOptionsParams) (*models.Cloud, error)

func (client *CloudClient) SetGetByNameCloudFunc(fn GetByNameCloudFunc) {
	client.getByNameCloudFn = fn
}

func (client *CloudClient) GetByName(name string, options ...session.ApiOptionsParams) (*models.Cloud, error) {
	return client.getByNameCloudFn(name)
}

//IPAMDNSProviderProfile
type IPAMDNSProviderProfileClient struct {
	getIPAMFn    GetIPAMFunc
	updateIPAMFn UpdateIPAMFn
}

type GetIPAMFunc func(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error)
type UpdateIPAMFn func(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error)

func (client *IPAMDNSProviderProfileClient) SetGetIPAMFunc(fn GetIPAMFunc) {
	client.getIPAMFn = fn
}

func (client *IPAMDNSProviderProfileClient) Get(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
	return client.getIPAMFn(uuid)
}

func (client *IPAMDNSProviderProfileClient) SetUpdateIPAMFn(fn UpdateIPAMFn) {
	client.updateIPAMFn = fn
}

func (client *IPAMDNSProviderProfileClient) Update(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
	return client.updateIPAMFn(obj)
}

// User Client
type UserClient struct {
	getByNameUserFn    GetByNameUserFunc
	deleteByNameUserFn DeleteByNameUserFunc
	createUserFunc     CreateUserFunc
	updateUserFunc     UpdateUserFunc
}

type GetByNameUserFunc func(name string, options ...session.ApiOptionsParams) (*models.User, error)
type DeleteByNameUserFunc func(name string, options ...session.ApiOptionsParams) error
type CreateUserFunc func(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error)
type UpdateUserFunc func(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error)

func (client *UserClient) SetGetByNameUserFunc(fn GetByNameUserFunc) {
	client.getByNameUserFn = fn
}

func (client *UserClient) SetDeleteByNameUserFunc(fn DeleteByNameUserFunc) {
	client.deleteByNameUserFn = fn
}

func (client *UserClient) SetCreateUserFunc(fn CreateUserFunc) {
	client.createUserFunc = fn
}

func (client *UserClient) SetUpdateUserFunc(fn UpdateUserFunc) {
	client.updateUserFunc = fn
}

func (client *UserClient) GetByName(name string, options ...session.ApiOptionsParams) (*models.User, error) {
	return client.getByNameUserFn(name)
}

func (client *UserClient) DeleteByName(name string, options ...session.ApiOptionsParams) error {
	return client.deleteByNameUserFn(name)
}

func (client *UserClient) Create(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
	return client.createUserFunc(obj)
}

func (client *UserClient) Update(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
	return client.updateUserFunc(obj)
}

//Tenant Client
type TenantClient struct {
	getTenantFn GetTenantFunc
}

type GetTenantFunc func(uuid string, options ...session.ApiOptionsParams) (*models.Tenant, error)

func (client *TenantClient) SetGetTenantFunc(fn GetTenantFunc) {
	client.getTenantFn = fn
}

func (client *TenantClient) Get(uuid string, options ...session.ApiOptionsParams) (*models.Tenant, error) {
	return client.getTenantFn(uuid)
}

// Role Client
type RoleClient struct {
	getByNameRoleFn GetByNameRoleFunc
	createRoleFunc  CreateRoleFunc
}

type GetByNameRoleFunc func(name string, options ...session.ApiOptionsParams) (*models.Role, error)
type CreateRoleFunc func(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error)

func (client *RoleClient) SetGetByNameRoleFunc(fn GetByNameRoleFunc) {
	client.getByNameRoleFn = fn
}

func (client *RoleClient) SetCreateRoleFunc(fn CreateRoleFunc) {
	client.createRoleFunc = fn
}

func (client *RoleClient) GetByName(name string, options ...session.ApiOptionsParams) (*models.Role, error) {
	return client.getByNameRoleFn(name)
}

func (client *RoleClient) Create(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error) {
	return client.createRoleFunc(obj)
}

// Pool Client
type PoolClient struct {
	getByNameFn GetByNamePoolFunc
}

type GetByNamePoolFunc func(name string, options ...session.ApiOptionsParams) (*models.Pool, error)

func (client *PoolClient) SetGetByNameFn(fn GetByNamePoolFunc) {
	client.getByNameFn = fn
}

func (client *PoolClient) GetByName(name string, options ...session.ApiOptionsParams) (*models.Pool, error) {
	return client.getByNameFn(name)
}

// VirtualService Client
type VirtualServiceClient struct {
	getByNameFn GetByNameVSFunc
}

type GetByNameVSFunc func(name string, options ...session.ApiOptionsParams) (*models.VirtualService, error)

func (client *VirtualServiceClient) SetGetByNameFn(fn GetByNameVSFunc) {
	client.getByNameFn = fn
}

func (client *VirtualServiceClient) GetByName(name string, options ...session.ApiOptionsParams) (*models.VirtualService, error) {
	return client.getByNameFn(name)
}
