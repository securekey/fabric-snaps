/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

// SchemaConfig defines request and response schemas for content type
type SchemaConfig struct {
	// Content type
	Type string

	// Request schema
	Request string

	// Response schema
	Response string
}

// ClientCrt defines client crt
type ClientCrt struct {
	// CA
	Ca string

	// Public Crt
	Crt string

	// Private Key
	Key string
}

// Config configuration interface
type Config interface {
	GetConfigPath(path string) string
	GetClientCert() string
	GetClientKey() string
	GetNamedClientOverride() (map[string]*ClientCrt, error)
	GetSchemaConfig(contentType string) (*SchemaConfig, error)
	GetCaCerts() []string
}
