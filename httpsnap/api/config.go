/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"time"
)

// SchemaConfig defines request and response schemas for content type
type SchemaConfig struct {
	// Content type
	Type string

	// Request schema
	Request string

	// Response schema
	Response string
}

// ClientTLS defines client crt and key
type ClientTLS struct {
	// CA
	Ca string

	// Public Crt
	Crt string
}

// HTTPClientTimeoutType enumerates the different types of timeouts used by http client
type HTTPClientTimeoutType int

// Timeouts used by HTTP client
const (
	Global HTTPClientTimeoutType = iota
	TransportTLSHandshake
	TransportResponseHeader
	TransportExpectContinue
	TransportIdleConn
	DialerTimeout
	DialerKeepAlive
)

// Config configuration interface
type Config interface {
	GetConfigPath(path string) string
	GetClientCert() (string, error)
	GetNamedClientOverride() (map[string]*ClientTLS, error)
	GetSchemaConfig(contentType string) (*SchemaConfig, error)
	GetCaCerts() ([]string, error)
	GetPeerClientKey() (string, error)
	GetCryptoProvider() (string, error)
	IsSystemCertPoolEnabled() bool
	TimeoutOrDefault(timeoutType HTTPClientTimeoutType) time.Duration
	IsPeerTLSConfigEnabled() bool
	IsHeaderAllowed(name string) (bool, error)
}
