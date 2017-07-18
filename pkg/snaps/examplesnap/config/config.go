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

var logger = logging.MustGetLogger("examplesnap-config")
var defaultLogFormat = `%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`
var defaultLogLevel = "info"

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
	logger.Debugf("Logging initialized. Log level: %s", logging.GetLevel(""))

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

// GetGreeting returns greeting
func GetGreeting() string {
	return viper.GetString("greeting")
}
