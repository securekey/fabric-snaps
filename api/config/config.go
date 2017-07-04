/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
)

const (
	configFileName = "config"
	cmdRootPrefix  = "core"
	devConfigPath  = "$GOPATH/src/github.com/securekey/fabric-snaps/cmd/config/sampleconfig"
)

var logger = logging.MustGetLogger("snap-config")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// SnapConfig defines the metadata needed to initialize the code
// when the fabric comes up. SnapConfigs are installed by adding an
// entry in config.yaml and creating the new SnapConfig implementation
type SnapConfig struct {
	// Enabled a convenient switch to enable/disable SnapConfig without
	// having to remove entry from Snaps array below
	Enabled bool

	//Unique name of the snap code, it should match the SnapConfig implementation class name
	Name string

	//String representation for InitArgs read by yaml
	InitArgsStr []string

	//InitArgs initialization arguments to startup the snap chaincode
	InitArgs [][]byte

	// SnapConfig is the actual SnapConfig object
	Snap shim.Chaincode

	// SnapURL to locate remote Snaps
	SnapURL string

	// TLSEnabled indicates whether TLS is used when invoking remote snaps
	TLSEnabled bool

	// TLSRootCertFile is the root certificate fil (only applicable if TLSEnabled=true)
	TLSRootCertFile string
}

// SnapConfigArray represents the list of snaps configurations from YAML
type SnapConfigArray struct {
	SnapConfigs []SnapConfig
}

// Init configuration and logging for SnapConfigs. By default, we look for
// configuration files at a path described by the environment variable
// "FABRIC_CFG_PATH". This is where the configuration is expected to be set in
// a production SnapConfigs image. For testing and development, a GOPATH, project
// relative path is used. Optionally, a path override parameter can be passed in
// @param {string} [OPTIONAL] configPathOverride
// @returns {error} error, if any
func Init(configPathOverride string) error {
	var configPath = os.Getenv("FABRIC_CFG_PATH")
	replacer := strings.NewReplacer(".", "_")

	if configPath != "" {
		viper.AddConfigPath(configPath)
	} else {
		if configPathOverride == "" {
			configPathOverride = devConfigPath
		}
		viper.AddConfigPath(configPathOverride)
	}
	viper.SetConfigName(configFileName)
	viper.SetEnvPrefix(cmdRootPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(replacer)

	err := viper.ReadInConfig()
	if err != nil {
		logger.Criticalf("Fatal error reading snap config file: %s", err)
		return fmt.Errorf("Fatal error reading snap config file: %s", err)
	}

	err = initializeLogging()
	if err != nil {
		logger.Criticalf("Error initializing logging: %s", err)
		return fmt.Errorf("Error initializing logging: %s", err)
	}

	return nil
}

func initializeLogging() error {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	level, err := logging.LogLevel(viper.GetString("snap.snapsd.loglevel"))

	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetBackend(backendFormatter).SetLevel(level, "")

	logger.Debugf("SnapConfigs Logger initialized. Log level: %s", logging.GetLevel(""))

	return nil
}

// IsTLSEnabled is TLS enabled?
func IsTLSEnabled() bool {
	return viper.GetBool("snap.server.tls.enabled")
}

// GetTLSRootCertPath returns absolute path to the TLS root certificate
func GetTLSRootCertPath() string {
	return GetConfigPath(viper.GetString("snap.server.tls.rootcert.file"))
}

// GetTLSCertPath returns absolute path to the TLS certificate
func GetTLSCertPath() string {
	return GetConfigPath(viper.GetString("snap.server.tls.cert.file"))
}

// GetTLSKeyPath returns absolute path to the TLS key
func GetTLSKeyPath() string {
	return GetConfigPath(viper.GetString("snap.server.tls.key.file"))
}

// GetSnapServerPort returns snap server port
func GetSnapServerPort() string {
	return viper.GetString("snap.server.port")
}

// GetConfigPath returns the absolute value of the given path that is
// relative to the config file
// For example, if the config file is at /etc/hyperledger/config.yaml,
// calling GetConfigPath("tls/cert") will return /etc/hyperledger/tls/cert
func GetConfigPath(path string) string {
	basePath := filepath.Dir(viper.ConfigFileUsed())

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(basePath, path)
}
