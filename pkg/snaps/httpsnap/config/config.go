/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package config

import (
	"fmt"
	"os"
	"path/filepath"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

const (
	configFileName = "config"
)

var logger = logging.MustGetLogger("httpsnap-config")
var defaultLogFormat = `%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`
var defaultLogLevel = "info"

// SchemaConfig defines request and response schemas for content type
type SchemaConfig struct {
	// Content type
	Type string

	// Request schema
	Request string

	// Response schema
	Response string
}

// Init configuration and logging for snap. By default, we look for configuration files
// in working directory. Optionally, a path override parameter can be passed in.
// @param {string} [OPTIONAL] configPathOverride
// @returns {error} error, if any
func Init(configPathOverride string) error {

	// default config path is working directory
	configPath := "./"
	if configPathOverride != "" {
		configPath = configPathOverride
	}

	viper.AddConfigPath(configPath)
	viper.SetConfigName(configFileName)
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Error reading snap config file: %s", err)
	}

	err = initializeLogging()
	if err != nil {
		return fmt.Errorf("Error initializing logging: %s", err)
	}

	return nil
}

// Helper function to initialize logging
func initializeLogging() error {

	logFormat := viper.GetString("logging.format")
	if logFormat == "" {
		logFormat = defaultLogFormat
	}

	logLevel := viper.GetString("logging.level")
	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logging.MustStringFormatter(logFormat))
	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetBackend(backendFormatter).SetLevel(level, "")
	logger.Debugf("Httpsnap logging initialized. Log level: %s", logging.GetLevel(""))

	return nil
}

// GetConfigPath returns the absolute value of the given path that is relative to the config file
// For example, if the config file is at /opt/snaps/example/config.yaml,
// calling GetConfigPath("tls/cert") will return /opt/snaps/example/tls/cert
func GetConfigPath(path string) string {
	basePath := filepath.Dir(viper.ConfigFileUsed())

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(basePath, path)
}

// GetCaCerts returns the list of ca certs
func GetCaCerts() []string {

	caCerts := viper.GetStringSlice("tls.caCerts")
	absoluteCaCerts := make([]string, 0, len(caCerts))

	for _, v := range caCerts {
		absoluteCaCerts = append(absoluteCaCerts, GetConfigPath(v))
	}

	return absoluteCaCerts
}

// Helper function to retieve schema configuration
func getSchemaMap() (schemaMap map[string]*SchemaConfig, err error) {

	var schemaConfigs []SchemaConfig
	err = viper.UnmarshalKey("schemas", &schemaConfigs)
	if err != nil {
		return nil, err
	}

	schemaMap = make(map[string]*SchemaConfig, len(schemaConfigs))

	for _, sc := range schemaConfigs {
		sc.Request = GetConfigPath(sc.Request)
		sc.Response = GetConfigPath(sc.Response)
		schemaMap[sc.Type] = &sc
	}

	return schemaMap, nil
}

// GetClientCert returns client cert
func GetClientCert() string {
	return GetConfigPath(viper.GetString("tls.clientCert"))
}

// GetClientKey returns client key
func GetClientKey() string {
	return GetConfigPath(viper.GetString("tls.clientKey"))
}

// GetNamedClientOverridePath returns overide path
func GetNamedClientOverridePath() string {
	return GetConfigPath(viper.GetString("tls.namedClientOverridePath"))
}

// GetSchemaConfig return schema configuration based on content type
func GetSchemaConfig(contentType string) (*SchemaConfig, error) {
	schemaMap, err := getSchemaMap()
	if err != nil {
		return nil, err
	}

	schemaConfig := schemaMap[contentType]
	logger.Debugf("Schema config: %s", schemaConfig)
	if schemaConfig == nil {
		return nil, fmt.Errorf("Schema configuration for content-type: %s not found", contentType)
	}

	return schemaConfig, nil
}
