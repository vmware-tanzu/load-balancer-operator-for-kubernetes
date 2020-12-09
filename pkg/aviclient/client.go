// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// This packages provides a wrapper for AVI Controller's REST Client

package aviclient

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"

	"github.com/avinetworks/sdk/go/clients"
	"github.com/avinetworks/sdk/go/session"
)

var SharedAviClient *Client

type Client struct {
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
func NewAviClient(config *AviClientConfig) (*Client, error) {
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
	return &Client{
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
