/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"
	"github.com/securekey/fabric-snaps/util/configcache"
	"github.com/securekey/fabric-snaps/util/errors"

	"github.com/spf13/viper"
)

const (
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
	defaultTimeout     = time.Second * 5
)

var logger = logging.NewLogger("httpsnap")
var defaultLogLevel = "info"
var peerConfigCache = configcache.New(peerConfigFileName, cmdRootPrefix, "/etc/hyperledger/fabric")

// FilePathSeparator separator defined by os.Separator.
const FilePathSeparator = string(filepath.Separator)

// config implements Config interface
type config struct {
	peerConfig     *viper.Viper
	httpSnapConfig *viper.Viper
	peerConfigPath string
	//preloaded entities
	clientTLS     map[string]*httpsnapApi.ClientTLS
	headers       map[string]bool
	schemaConfigs map[string]*httpsnapApi.SchemaConfig
}

// NewConfig return config struct
func NewConfig(peerConfigPath string, channelID string) (httpsnapApi.Config, bool, error) {
	peerConfig, err := peerConfigCache.Get(peerConfigPath)
	if err != nil {
		return nil, false, err
	}
	//httpSnapConfig Config
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"), PeerID: peerConfig.GetString("peer.id"), AppName: "httpsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, false, errors.New(errors.SystemError, "Cannot create cache instance")
	}
	configData, dirty, err := cacheInstance.Get(channelID, key)
	if err != nil {
		return nil, false, errors.WithMessage(errors.SystemError, err, "Failed cacheInstance")
	}
	if configData == nil {
		return nil, false, errors.New(errors.InitializeConfigError, "config data is empty")
	}
	httpSnapConfig := viper.New()
	httpSnapConfig.SetConfigType("YAML")
	err = httpSnapConfig.ReadConfig(bytes.NewBuffer(configData))
	if err != nil {
		return nil, false, errors.WithMessage(errors.InitializeConfigError, err, "snap_config_init_error")
	}
	httpSnapConfig.SetEnvPrefix(cmdRootPrefix)
	httpSnapConfig.AutomaticEnv()
	httpSnapConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c := &config{peerConfig: peerConfig, httpSnapConfig: httpSnapConfig, peerConfigPath: peerConfigPath}
	err = c.preloadEntities()
	if err != nil {
		return nil, false, err
	}
	if dirty {
		err = c.initializeLogging()
		if err != nil {
			return nil, false, err
		}
	}

	return c, dirty, nil
}

// Helper function to initialize logging
func (c *config) initializeLogging() errors.Error {
	logLevel := c.httpSnapConfig.GetString("logging.level")

	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return errors.WithMessage(errors.InitializeLoggingError, err, "Error initializing log level")
	}

	logging.SetLevel("httpsnap", level)
	logger.Debugf("Httpsnap logging initialized. Log level: %s", logLevel)

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

// IsHeaderAllowed returns true if specified http header type is enabled
func (c *config) IsHeaderAllowed(name string) bool {
	val, _ := c.headers[strings.ToLower(name)]
	return val
}

// GetCaCerts returns the list of ca certs
// if not found in config and use peer tls config enabled
// then returns peer config tls root cert
func (c *config) GetCaCerts() ([]string, errors.Error) {

	caCerts := c.httpSnapConfig.GetStringSlice("tls.caCerts")
	absoluteCaCerts := make([]string, 0, len(caCerts))

	for _, v := range caCerts {
		absoluteCaCerts = append(absoluteCaCerts, v)
	}

	if len(absoluteCaCerts) == 0 && c.IsPeerTLSConfigEnabled() {
		return c.getPeerTLSRootCert()
	}

	return absoluteCaCerts, nil
}

// GetClientCert returns client cert
// if not found in config and use peer tls config enabled
// then returns peer config client cert
func (c *config) GetClientCert() (string, errors.Error) {
	clientCert := c.httpSnapConfig.GetString("tls.clientCert")

	if clientCert == "" && c.IsPeerTLSConfigEnabled() {
		return c.getPeerClientCert()
	}
	return clientCert, nil
}

// GetPeerClientKey returns peer tls client key
func (c *config) GetPeerClientKey() (string, errors.Error) {
	clientKeyLocation := c.peerConfig.GetString("peer.tls.clientKey.file")
	if clientKeyLocation == "" {
		clientKeyLocation = c.peerConfig.GetString("peer.tls.key.file")
	}

	fileData, err := ioutil.ReadFile(c.translatePeerPath(clientKeyLocation))
	if err != nil {
		return "", errors.WithMessage(errors.SystemError, err, "Failed to read peer's tls client key file")
	}
	return string(fileData), nil
}

// IsSystemCertsPoolEnabled returns true if loading of the system cert pool is enabled
func (c *config) IsSystemCertPoolEnabled() bool {
	return c.httpSnapConfig.GetBool("tls.enableSystemCertPool")
}

// IsPeerTLSConfigEnabled returns true if peer TLS config is enabled
func (c *config) IsPeerTLSConfigEnabled() bool {
	return c.httpSnapConfig.GetBool("tls.allowPeerConfig")
}

// getPeerClientCert returns client tls cert
func (c *config) getPeerClientCert() (string, errors.Error) {

	clientCertLocation := c.peerConfig.GetString("peer.tls.clientCert.file")
	if clientCertLocation == "" {
		clientCertLocation = c.peerConfig.GetString("peer.tls.cert.file")
	}

	fileData, err := ioutil.ReadFile(c.translatePeerPath(clientCertLocation))
	if err != nil {
		return "", errors.WithMessage(errors.SystemError, err, "Failed to read peer's tls client cert file")
	}
	return string(fileData), nil
}

// getPeerTLSRootCert returns tls root certs from peer config
func (c *config) getPeerTLSRootCert() ([]string, errors.Error) {

	rootCertLocation := c.peerConfig.GetString("peer.tls.rootcert.file")
	if rootCertLocation == "" {
		return make([]string, 0), nil
	}

	fileData, err := ioutil.ReadFile(c.translatePeerPath(rootCertLocation))
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to read peer's tls root cert file")
	}

	return []string{string(fileData)}, nil
}

// translatePeerPath Translates a relative path into a fully qualified path, fully qualified path will be ignored
func (c *config) translatePeerPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.peerConfigPath, path)
}

// GetNamedClientOverridePath returns map of clientTLS
func (c *config) GetNamedClientOverride() map[string]*httpsnapApi.ClientTLS {
	return c.clientTLS
}

// GetSchemaConfig return schema configuration based on content type
func (c *config) GetSchemaConfig(contentType string) (*httpsnapApi.SchemaConfig, errors.Error) {

	schemaConfig := c.schemaConfigs[contentType]
	logger.Debugf("Schema config: %s", schemaConfig)
	if schemaConfig == nil {
		return nil, errors.Errorf(errors.MissingConfigDataError, "Schema configuration for content-type: %s not found", contentType)
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

func (c *config) GetCryptoProvider() (string, errors.Error) {
	cryptoProvider := c.peerConfig.GetString("peer.BCCSP.Default")
	if cryptoProvider == "" {
		return "", errors.New(errors.CryptoConfigError, "BCCSP Default provider not found")
	}
	return cryptoProvider, nil
}

func (c *config) preloadEntities() errors.Error {

	//client TLS configs
	err := c.httpSnapConfig.UnmarshalKey("tls.namedClientOverride", &c.clientTLS)
	if err != nil {
		return errors.WithMessage(errors.InitializeConfigError, err, "Failed to unmarshal tls.namedClientOverride")
	}

	// header configs
	var allHeaders []string
	err = c.httpSnapConfig.UnmarshalKey("headers", &allHeaders)
	if err != nil {
		return errors.WithMessage(errors.InitializeConfigError, err, "Failed to unmarshal headers")

	}

	if allHeaders == nil || len(allHeaders) == 0 {
		return errors.New(errors.InitializeConfigError, "Missing http headers configuration")
	}

	c.headers = make(map[string]bool, len(allHeaders))
	for _, h := range allHeaders {
		c.headers[strings.ToLower(h)] = true
	}

	//schema configs
	var allSchemas []httpsnapApi.SchemaConfig
	err = c.httpSnapConfig.UnmarshalKey("schemas", &allSchemas)
	if err != nil {
		return errors.WithMessage(errors.InitializeConfigError, err, "Failed to unmarshal schemas")
	}

	c.schemaConfigs = make(map[string]*httpsnapApi.SchemaConfig, len(allSchemas))

	for _, sc := range allSchemas {
		c.schemaConfigs[sc.Type] = &sc
	}

	return nil
}
