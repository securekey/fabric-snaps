/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"time"

	"github.com/securekey/fabric-snaps/util/errors"
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
	GetClientCert() (string, errors.Error)
	GetNamedClientOverride() map[string]*ClientTLS
	GetSchemaConfig(contentType string) (*SchemaConfig, errors.Error)
	GetCaCerts() ([]string, errors.Error)
	GetPeerClientKey() (string, errors.Error)
	GetCryptoProvider() (string, errors.Error)
	IsSystemCertPoolEnabled() bool
	TimeoutOrDefault(timeoutType HTTPClientTimeoutType) time.Duration
	IsPeerTLSConfigEnabled() bool
	IsHeaderAllowed(name string) bool
	IsKeyCacheEnabled() bool
	KeyCacheRefreshInterval() time.Duration
}
