/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package httpsnap

import (
	"fmt"

	"github.com/spf13/viper"

	config "github.com/securekey/fabric-snaps/cmd/config"
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

// helper function to return the list of ca certs
func getCaCerts() []string {

	caCerts := viper.GetStringSlice("httpsnap.tls.caCerts")
	absoluteCaCerts := make([]string, 0, len(caCerts))

	for _, v := range caCerts {
		absoluteCaCerts = append(absoluteCaCerts, config.GetConfigPath(v))
	}

	return absoluteCaCerts
}

// helper function to retrieve schema configuration
func getSchemaMap() (schemaMap map[string]*SchemaConfig, err error) {

	var schemaConfigs []SchemaConfig
	err = viper.UnmarshalKey("httpsnap.schemas", &schemaConfigs)
	if err != nil {
		return nil, err
	}

	schemaMap = make(map[string]*SchemaConfig)

	for _, sc := range schemaConfigs {
		sc.Request = config.GetConfigPath(sc.Request)
		sc.Response = config.GetConfigPath(sc.Response)
		schemaMap[sc.Type] = &sc
	}

	return schemaMap, nil
}

func getClientCert() string {
	return config.GetConfigPath(viper.GetString("httpsnap.tls.clientCert"))
}

func getClientKey() string {
	return config.GetConfigPath(viper.GetString("httpsnap.tls.clientKey"))
}

func getNamedClientOverridePath() string {
	return config.GetConfigPath(viper.GetString("httpsnap.tls.namedClientOverridePath"))
}

func getSchemaConfig(contentType string) (*SchemaConfig, error) {
	schemaMap, err := getSchemaMap()
	if err != nil {
		return nil, err
	}

	schemaConfig := schemaMap[contentType]
	if schemaConfig == nil {
		return nil, fmt.Errorf("Schema configuration for content-type: %s not found", contentType)
	}

	return schemaConfig, nil
}
