// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// This packages provides a wrapper for AVI Controller's REST Client

package aviclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"regexp"
	"strings"

	"github.com/avinetworks/sdk/go/clients"
	"github.com/avinetworks/sdk/go/models"
	"github.com/avinetworks/sdk/go/session"
)

type realAviClient struct {
	config *AviClientConfig
	*clients.AviClient
}

type AviClientConfig struct {
	ServerIP  string
	Username  string
	Password  string
	CA        string
	Insecure  bool // Should only be used for tests
	Transport *http.Transport

	// ServerName is used to verify the hostname on the returned
	// certificates unless Insecure is true. It is also included
	// in the client's handshake to support virtual hosting unless it is
	// an IP address.
	ServerName string
}

// NewAviClient creates an Client
func NewAviClient(config *AviClientConfig) (*realAviClient, error) {
	// Initialize transport
	var transport *http.Transport
	if config.CA != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(config.CA))
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
		if config.ServerName != "" {
			transport.TLSClientConfig.ServerName = config.ServerName
		}
	}
	// Passed in transport overwrites the one created above
	if config.Transport != nil {
		transport = config.Transport
	}

	options := []func(*session.AviSession) error{
		session.SetPassword(config.Password),
		session.SetTransport(transport),
	}
	if config.CA == "" {
		options = append(options, session.SetInsecure)
	}

	client, err := clients.NewAviClient(config.ServerIP, config.Username, options...)
	if err != nil {
		return nil, err
	}
	return &realAviClient{
		AviClient: client,
		config:    config,
	}, nil
}

// GetUUIDFromRef takes a AVI Ref, parses it as a classic URL and returns the
// last part
func GetUUIDFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// IsAviUserAlreadyExistsError returns if an error is User Already Exists error
// by matching error message
func IsAviUserAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	matched, err := regexp.Match(`User with this Username already exist`, []byte(err.Error()))
	return err == nil && matched
}

// IsAviUserAlreadyExistsError returns if an error is User doesn't exist error
// by matching error message
func IsAviUserNonExistentError(err error) bool {
	if err == nil {
		return false
	}
	matched, err := regexp.Match(`No object of type user with name .*is found`, []byte(err.Error()))
	return err == nil && matched
}

// IsAviRoleNonExistentError returns if an error is User role doesn't exist error
// by matching error message
func IsAviRoleNonExistentError(err error) bool {
	if err == nil {
		return false
	}
	matched, err := regexp.Match(`No object of type role with name .*is found`, []byte(err.Error()))
	return err == nil && matched
}

func (r *realAviClient) NetworkGetByName(name string, options ...session.ApiOptionsParams) (*models.Network, error) {
	return r.Network.GetByName(name)
}

func (r *realAviClient) NetworkUpdate(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error) {
	return r.Network.Update(obj)
}

func (r *realAviClient) CloudGetByName(name string, options ...session.ApiOptionsParams) (*models.Cloud, error) {
	return r.Cloud.GetByName(name)
}

func (r *realAviClient) IPAMDNSProviderProfileGet(uuid string, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
	return r.IPAMDNSProviderProfile.Get(uuid)
}

func (r *realAviClient) IPAMDNSProviderProfileUpdate(obj *models.IPAMDNSProviderProfile, options ...session.ApiOptionsParams) (*models.IPAMDNSProviderProfile, error) {
	return r.IPAMDNSProviderProfile.Update(obj)
}

func (r *realAviClient) UserGetByName(name string, options ...session.ApiOptionsParams) (*models.User, error) {
	return r.User.GetByName(name)
}

func (r *realAviClient) UserDeleteByName(name string, options ...session.ApiOptionsParams) error {
	return r.User.DeleteByName(name)
}

func (r *realAviClient) UserCreate(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
	return r.User.Create(obj)
}

func (r *realAviClient) UserUpdate(obj *models.User, options ...session.ApiOptionsParams) (*models.User, error) {
	return r.User.Update(obj)
}

func (r *realAviClient) TenantGet(uuid string, options ...session.ApiOptionsParams) (*models.Tenant, error) {
	return r.Tenant.Get(uuid)
}

func (r *realAviClient) RoleGetByName(name string, options ...session.ApiOptionsParams) (*models.Role, error) {
	return r.Role.GetByName(name)
}

func (r *realAviClient) RoleCreate(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error) {
	return r.Role.Create(obj)
}

func (r *realAviClient) VirtualServiceGetByName(name string, options ...session.ApiOptionsParams) (*models.VirtualService, error) {
	return r.VirtualService.GetByName(name)
}

func (r *realAviClient) PoolGetByName(name string, options ...session.ApiOptionsParams) (*models.Pool, error) {
	return r.Pool.GetByName(name)
}
