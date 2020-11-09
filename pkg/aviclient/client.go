// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// This packages provides a wrapper for AVI Controller's REST Client

package aviclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/avinetworks/sdk/go/clients"
	"github.com/avinetworks/sdk/go/session"
)

var SharedAviClient *Client

type Client struct {
	config *AviClientConfig
	Client *clients.AviClient
}

type AviClientConfig struct {
	ServerIP  string
	Username  string
	Password  string
	CA        string
	Insecure  bool
	Transport *http.Transport
}

// NewAVIClient creates an Client
func NewAVIClient(config *AviClientConfig) (*Client, error) {
	// Initialize transport
	var transport *http.Transport
	if config.CA != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(config.CA))
		transport =
			&http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: caCertPool,
				},
			}
	}
	// Passed in transport overwrites the one created above
	if config.Transport != nil {
		transport = config.Transport
	}

	options := []func(*session.AviSession) error{
		session.SetPassword(config.Password),
		session.SetNoControllerStatusCheck,
		session.SetTransport(transport),
	}
	if config.CA == "" {
		options = append(options, session.SetInsecure)
	}

	client, err := clients.NewAviClient(config.ServerIP, config.Username, options...)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client: client,
		config: config,
	}, nil
}
