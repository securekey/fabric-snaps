/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"

	"github.com/spf13/viper"
)

const (
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
	defaultTimeout     = time.Second * 5
)

var logger = logging.NewLogger("httpsnap-config")
var defaultLogFormat = `%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`
var defaultLogLevel = "info"

// FilePathSeparator separator defined by os.Separator.
const FilePathSeparator = string(filepath.Separator)

// config implements Config interface
type config struct {
	peerConfig     *viper.Viper
	httpSnapConfig *viper.Viper
	peerConfigPath string
}

// NewConfig return config struct
func NewConfig(peerConfigPath string, channelID string) (httpsnapApi.Config, error) {
	replacer := strings.NewReplacer(".", "_")
	if peerConfigPath == "" {
		peerConfigPath = "/etc/hyperledger/fabric"
	}
	//peer Config
	peerConfig := viper.New()
	peerConfig.AddConfigPath(peerConfigPath)
	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(replacer)
	err := peerConfig.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Fatal error reading peer config file: %s", err)
	}
	//httpSnapConfig Config
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"), PeerID: peerConfig.GetString("peer.id"), AppName: "httpsnap"}
	cacheInstance := configmgmtService.GetInstance()
	configData, err := cacheInstance.Get(channelID, key)
	if err != nil {
		return nil, err
	}
	httpSnapConfig := viper.New()
	httpSnapConfig.SetConfigType("YAML")
	httpSnapConfig.ReadConfig(bytes.NewBuffer(configData))
	httpSnapConfig.SetEnvPrefix(cmdRootPrefix)
	httpSnapConfig.AutomaticEnv()
	httpSnapConfig.SetEnvKeyReplacer(replacer)
	c := &config{peerConfig: peerConfig, httpSnapConfig: httpSnapConfig, peerConfigPath: peerConfigPath}
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

	logging.SetLevel("httpsnap", level)
	logger.Debugf("Httpsnap logging initialized. Log level: %s", logging.GetLevel("httpsnap"))

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

// IsSystemCertsPoolEnabled returns true if loading of the system cert pool is enabled
func (c *config) IsSystemCertPoolEnabled() bool {
	return c.httpSnapConfig.GetBool("tls.enableSystemCertPool")
}

// GetClientKey returns client key
func (c *config) GetClientKey() (string, error) {
	fileData, err := ioutil.ReadFile(substGoPath(c.httpSnapConfig.GetString("tls.clientKey")))
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

// IsPeerTLSConfigEnabled returns true if peer TLS config is enabled
func (c *config) IsPeerTLSConfigEnabled() bool {
	return c.httpSnapConfig.GetBool("tls.usePeerConfig")
}

// GetPeerClientCert returns client tls cert
func (c *config) GetPeerClientCert() (string, error) {
	fileData, err := ioutil.ReadFile(c.translatePeerPath(c.peerConfig.GetString("peer.tls.cert.file")))
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

// GetPeerClientKey returns client tls key
func (c *config) GetPeerClientKey() (string, error) {
	fileData, err := ioutil.ReadFile(c.translatePeerPath(c.peerConfig.GetString("peer.tls.key.file")))
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

// GetPeerClientKey returns client tls key
func (c *config) GetPeerTLSRootCert() (string, error) {
	rootCertLocation := c.peerConfig.GetString("peer.tls.rootcert.file")
	if rootCertLocation == "" {
		return "", nil
	}
	fileData, err := ioutil.ReadFile(c.translatePeerPath(rootCertLocation))
	if err != nil {
		return "", err
	}
	return string(fileData), nil
}

// translatePeerPath Translates a relative path into a fully qualified path, fully qualified path will be ignored
func (c *config) translatePeerPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.peerConfigPath, path)
}

// GetNamedClientOverridePath returns map of clientTLS
func (c *config) GetNamedClientOverride() (map[string]*httpsnapApi.ClientTLS, error) {
	var clientTLS map[string]*httpsnapApi.ClientTLS
	err := c.httpSnapConfig.UnmarshalKey("tls.namedClientOverride", &clientTLS)
	if err != nil {
		return nil, err
	}
	for k, v := range clientTLS {
		fileData, err := ioutil.ReadFile(substGoPath(v.Key))
		if err != nil {
			return nil, err
		}
		clientTLS[k].Key = string(fileData)
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

// TimeoutOrDefault reads connection timeouts for the given connection type
func (c *config) TimeoutOrDefault(tt httpsnapApi.HTTPClientTimeoutType) time.Duration {
	var timeout time.Duration
	switch tt {
	case httpsnapApi.Global:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.client.timeout")
	case httpsnapApi.TransportTLSHandshake:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.transport.tlsHandshake")
	case httpsnapApi.TransportResponseHeader:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.transport.responseHeader")
	case httpsnapApi.TransportExpectContinue:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.transport.expectContinue")
	case httpsnapApi.TransportIdleConn:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.transport.idleConn")
	case httpsnapApi.DialerTimeout:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.dialer.timeout")
	case httpsnapApi.DialerKeepAlive:
		timeout = c.httpSnapConfig.GetDuration("httpclient.timeout.dialer.keepAlive")
	}
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return timeout
}

// substGoPath replaces instances of '$GOPATH' with the GOPATH. If the system
// has multiple GOPATHs then the first is used.
func substGoPath(s string) string {
	gpDefault := build.Default.GOPATH
	gps := filepath.SplitList(gpDefault)

	return strings.Replace(s, "$GOPATH", gps[0], -1)
}
