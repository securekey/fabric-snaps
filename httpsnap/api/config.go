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

// Config configuration interface
type Config interface {
	GetConfigPath(path string) string
	GetClientCert() string
	GetClientKey() string
	GetNamedClientOverridePath() string
	GetSchemaConfig(contentType string) (*SchemaConfig, error)
	GetCaCerts() []string
}
