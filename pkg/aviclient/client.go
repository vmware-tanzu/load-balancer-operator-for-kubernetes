// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

// This packages provides a wrapper for AVI Controller's REST Client

package aviclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/vmware/alb-sdk/go/clients"
	"github.com/vmware/alb-sdk/go/models"
	"github.com/vmware/alb-sdk/go/session"
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

var ErrEmptyInput = errors.New("input is empty")

// NewAviClientFromSecrets creates a Client from two secrets, adminCredential and CA
func NewAviClientFromSecrets(c client.Client, ctx context.Context, log logr.Logger,
	controllerIP, credName, credNamespace, caName, caNamespace, version string) (*realAviClient, error) {
	if controllerIP == "" {
		log.Error(ErrEmptyInput, "controllerIP is empty", "controllerIP", controllerIP)
		return nil, ErrEmptyInput
	}

	if credName == "" || credNamespace == "" || caName == "" || caNamespace == "" {
		log.Error(ErrEmptyInput, "empty secret", "credName",
			credName, "credNamespace", credNamespace,
			"caName", caName, "caNamespace", caNamespace)
		return nil, ErrEmptyInput
	}

	adminCredential := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{
		Name:      credName,
		Namespace: credNamespace,
	}, adminCredential); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cannot find referenced AdminCredential Secret, requeue the request")
		} else {
			log.Error(err, "Failed to find referenced AdminCredential Secret")
		}
		return nil, err
	}

	aviControllerCA := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{
		Name:      caName,
		Namespace: caNamespace,
	}, aviControllerCA); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Cannot find referenced CertificateAuthorityRef Secret, requeue the request")
		} else {
			log.Error(err, "Failed to find referenced CertificateAuthorityRef Secret")
		}
		return nil, err
	}
	aviClient, err := NewAviClient(&AviClientConfig{
		ServerIP: controllerIP,
		Username: string(adminCredential.Data["username"][:]),
		Password: string(adminCredential.Data["password"][:]),
		CA:       string(aviControllerCA.Data["certificateAuthorityData"][:]),
	}, version)
	if err != nil {
		log.Error(err, "Failed to initialize AVI Controller Client, requeue the request")
		return nil, err
	}

	return aviClient, nil
}

// NewAviClient creates an Client
func NewAviClient(config *AviClientConfig, version string) (*realAviClient, error) {
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

	if version != "" {
		options = append(options, session.SetVersion(version))
	}

	if config.CA == "" {
		options = append(options, session.SetInsecure)
	}

	c, err := clients.NewAviClient(config.ServerIP, config.Username, options...)
	if err != nil {
		return nil, err
	}
	return &realAviClient{
		AviClient: c,
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

func (r *realAviClient) GetControllerVersion() (string, error) {
	return r.AviSession.GetControllerVersion()
}

func (r *realAviClient) GetObjectByName(obj string, name string, cloudName string, result interface{}, options ...session.ApiOptionsParams) error {
	uri := "/api/" + obj + "/?include_name&name=" + name + "&cloud_ref.name=" + cloudName
	res, err := r.AviSession.GetCollectionRaw(uri, options...)
	if err != nil {
		return err
	}
	if res.Count == 0 {
		return errors.New("No object of type " + obj + " with name " + name + " is found")
	} else if res.Count > 1 {
		return errors.New("More than one object of type " + obj + " with name " + name + " is found")
	}
	elems := make([]json.RawMessage, 1)
	err = json.Unmarshal(res.Results, &elems)
	if err != nil {
		return err
	}
	return json.Unmarshal(elems[0], &result)
}

func (r *realAviClient) ServiceEngineGroupGetByName(name, cloudName string, options ...session.ApiOptionsParams) (*models.ServiceEngineGroup, error) {
	var obj *models.ServiceEngineGroup
	err := r.GetObjectByName("serviceenginegroup", name, cloudName, &obj, options...)
	return obj, err
}

func (r *realAviClient) ServiceEngineGroupCreate(obj *models.ServiceEngineGroup, options ...session.ApiOptionsParams) (*models.ServiceEngineGroup, error) {
	return r.ServiceEngineGroup.Create(obj)
}

func (r *realAviClient) NetworkGetByName(name, cloudName string, options ...session.ApiOptionsParams) (*models.Network, error) {
	var obj *models.Network
	err := r.GetObjectByName("network", name, cloudName, &obj, options...)
	return obj, err
}

func (r *realAviClient) NetworkCreate(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error) {
	return r.Network.Create(obj)
}

func (r *realAviClient) NetworkUpdate(obj *models.Network, options ...session.ApiOptionsParams) (*models.Network, error) {
	return r.Network.Update(obj)
}

func (r *realAviClient) CloudGetByName(name string, options ...session.ApiOptionsParams) (*models.Cloud, error) {
	return r.Cloud.GetByName(name)
}

func (r *realAviClient) CloudCreate(obj *models.Cloud, options ...session.ApiOptionsParams) (*models.Cloud, error) {
	return r.Cloud.Create(obj)
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

func (r *realAviClient) RoleUpdate(obj *models.Role, options ...session.ApiOptionsParams) (*models.Role, error) {
	return r.Role.Update(obj)
}

func (r *realAviClient) VirtualServiceGetByName(name string, options ...session.ApiOptionsParams) (*models.VirtualService, error) {
	return r.VirtualService.GetByName(name)
}

func (r *realAviClient) PoolGetByName(name string, options ...session.ApiOptionsParams) (*models.Pool, error) {
	return r.Pool.GetByName(name)
}

func (r *realAviClient) AviCertificateConfig() (string, error) {
	return r.config.CA, nil
}
