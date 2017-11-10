/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"path/filepath"
	"strings"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/client"
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"

	"github.com/spf13/viper"
)

const (
	configFileName     = "config"
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
)

var logger = logging.NewLogger("httpsnap-config")
var defaultLogFormat = `%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`
var defaultLogLevel = "info"

// config implements Config interface
type config struct {
	peerConfig     *viper.Viper
	httpSnapConfig *viper.Viper
}

// NewConfig return config struct
func NewConfig(configPathOverride string, stub shim.ChaincodeStubInterface) (httpsnapApi.Config, error) {

	replacer := strings.NewReplacer(".", "_")
	configPath := "/opt/extsysccs/config/httpsnap"
	peerConfigPath := "/etc/hyperledger/fabric"

	if configPathOverride != "" {
		configPath = configPathOverride
		peerConfigPath = configPathOverride
	}
	//httpSnapConfig Config
	httpSnapConfig := viper.New()
	httpSnapConfig.AddConfigPath(configPath)
	httpSnapConfig.SetConfigName(configFileName)
	httpSnapConfig.SetEnvPrefix(cmdRootPrefix)
	httpSnapConfig.AutomaticEnv()
	httpSnapConfig.SetEnvKeyReplacer(replacer)

	//peer Config
	peerConfig := viper.New()
	peerConfig.AddConfigPath(peerConfigPath)
	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(replacer)

	err := httpSnapConfig.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Fatal error reading config file: %s", err)
	}

	err = peerConfig.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Fatal error reading config file: %s", err)
	}

	httpSnapConfig, err = client.NewTempConfigClient(httpSnapConfig).Get(stub, &configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"), PeerID: peerConfig.GetString("peer.id"), AppName: "httpsnap"})
	if err != nil {
		return nil, fmt.Errorf("Fatal error from NewConfigClient: %s", err)
	}
	c := &config{peerConfig: peerConfig, httpSnapConfig: httpSnapConfig}
	err = c.initializeLogging()
	if err != nil {
		return nil, fmt.Errorf("Error initializing logging: %s", err)
	}
	return c, nil
}

// Helper function to initialize logging
func (c *config) initializeLogging() error {

	logLevel := c.httpSnapConfig.GetString("logging.level")

	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetLevel(level, "")
	logger.Debugf("Httpsnap logging initialized. Log level: %s", logging.GetLevel(""))

	return nil
}

// GetConfigPath returns the absolute value of the given path that is relative to the config file
// For example, if the config file is at /opt/snaps/example/config.yaml,
// calling GetConfigPath("tls/cert") will return /opt/snaps/example/tls/cert
func (c *config) GetConfigPath(path string) string {
	basePath := filepath.Dir(c.httpSnapConfig.ConfigFileUsed())

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(basePath, path)
}

// GetCaCerts returns the list of ca certs
func (c *config) GetCaCerts() []string {

	caCerts := c.httpSnapConfig.GetStringSlice("tls.caCerts")
	absoluteCaCerts := make([]string, 0, len(caCerts))

	for _, v := range caCerts {
		absoluteCaCerts = append(absoluteCaCerts, v)
	}

	return absoluteCaCerts
}

// Helper function to retieve schema configuration
func (c *config) getSchemaMap() (schemaMap map[string]*httpsnapApi.SchemaConfig, err error) {

	var schemaConfigs []httpsnapApi.SchemaConfig
	err = c.httpSnapConfig.UnmarshalKey("schemas", &schemaConfigs)
	if err != nil {
		return nil, err
	}

	schemaMap = make(map[string]*httpsnapApi.SchemaConfig, len(schemaConfigs))

	for _, sc := range schemaConfigs {
		schemaMap[sc.Type] = &sc
	}

	return schemaMap, nil
}

// GetClientCert returns client cert
func (c *config) GetClientCert() string {
	return c.httpSnapConfig.GetString("tls.clientCert")
}

// GetClientKey returns client key
func (c *config) GetClientKey() string {
	return c.httpSnapConfig.GetString("tls.clientKey")
}

// GetNamedClientOverridePath returns map of clientTLS
func (c *config) GetNamedClientOverride() (map[string]*httpsnapApi.ClientTLS, error) {
	var clientTLS map[string]*httpsnapApi.ClientTLS
	err := c.httpSnapConfig.UnmarshalKey("tls.namedClientOverride", &clientTLS)
	if err != nil {
		return nil, err
	}

	return clientTLS, nil
}

// GetSchemaConfig return schema configuration based on content type
func (c *config) GetSchemaConfig(contentType string) (*httpsnapApi.SchemaConfig, error) {
	schemaMap, err := c.getSchemaMap()
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
