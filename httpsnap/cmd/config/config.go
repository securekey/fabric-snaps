/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"go/build"
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
}

// NewConfig return config struct
func NewConfig(peerConfigPath string, channelID string) (httpsnapApi.Config, error) {
	peerConfig, err := peerConfigCache.Get(peerConfigPath)
	if err != nil {
		return nil, err
	}
	//httpSnapConfig Config
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"), PeerID: peerConfig.GetString("peer.id"), AppName: "httpsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, errors.New(errors.GeneralError, "Cannot create cache instance")
	}
	configData, err := cacheInstance.Get(channelID, key)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed cacheInstance")
	}
	if configData == nil {
		return nil, errors.New(errors.GeneralError, "config data is empty")
	}
	httpSnapConfig := viper.New()
	httpSnapConfig.SetConfigType("YAML")
	err = httpSnapConfig.ReadConfig(bytes.NewBuffer(configData))
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "snap_config_init_error")
	}
	httpSnapConfig.SetEnvPrefix(cmdRootPrefix)
	httpSnapConfig.AutomaticEnv()
	httpSnapConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c := &config{peerConfig: peerConfig, httpSnapConfig: httpSnapConfig, peerConfigPath: peerConfigPath}
	err = c.initializeLogging()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error initializing logging")
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
		return errors.WithMessage(errors.GeneralError, err, "Error initializing log level")
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

// Helper function to retrieve schema configuration
func (c *config) getSchemaMap() (schemaMap map[string]*httpsnapApi.SchemaConfig, err error) {

	var schemaConfigs []httpsnapApi.SchemaConfig
	err = c.httpSnapConfig.UnmarshalKey("schemas", &schemaConfigs)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed UnmarshalKey")
	}

	schemaMap = make(map[string]*httpsnapApi.SchemaConfig, len(schemaConfigs))

	for _, sc := range schemaConfigs {
		schemaMap[sc.Type] = &sc
	}

	return schemaMap, nil
}

// Helper function to retrieve allowed http request headers
func (c *config) getHeaderMap() (headerMap map[string]bool, err error) {

	var headers []string
	err = c.httpSnapConfig.UnmarshalKey("headers", &headers)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error getting allowed headers")
	}

	if headers == nil || len(headers) == 0 {
		return nil, errors.New(errors.GeneralError, "Missing http headers configuration")
	}

	headerMap = make(map[string]bool, len(headers))
	for _, h := range headers {
		headerMap[strings.ToLower(h)] = true
	}

	return headerMap, nil
}

// IsHeaderAllowed returns true if specified http header type is enabled
func (c *config) IsHeaderAllowed(name string) (bool, error) {
	headerMap, err := c.getHeaderMap()
	if err != nil {
		return false, err
	}

	val, ok := headerMap[strings.ToLower(name)]
	if !ok {
		return false, nil
	}

	return val, nil
}

// GetCaCerts returns the list of ca certs
// if not found in config and use peer tls config enabled
// then returns peer config tls root cert
func (c *config) GetCaCerts() ([]string, error) {

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
func (c *config) GetClientCert() (string, error) {
	clientCert := c.httpSnapConfig.GetString("tls.clientCert")

	if clientCert == "" && c.IsPeerTLSConfigEnabled() {
		return c.getPeerClientCert()
	}
	return clientCert, nil
}

// GetPeerClientKey returns peer tls client key
func (c *config) GetPeerClientKey() (string, error) {
	clientKeyLocation := c.peerConfig.GetString("peer.tls.clientKey.file")
	if clientKeyLocation == "" {
		clientKeyLocation = c.peerConfig.GetString("peer.tls.key.file")
	}

	fileData, err := ioutil.ReadFile(c.translatePeerPath(clientKeyLocation))
	if err != nil {
		return "", errors.WithMessage(errors.GeneralError, err, "Failed ReadFile")
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

// GetPeerClientCert returns client tls cert
func (c *config) getPeerClientCert() (string, error) {

	clientCertLocation := c.peerConfig.GetString("peer.tls.clientCert.file")
	if clientCertLocation == "" {
		clientCertLocation = c.peerConfig.GetString("peer.tls.cert.file")
	}

	fileData, err := ioutil.ReadFile(c.translatePeerPath(clientCertLocation))
	if err != nil {
		return "", errors.WithMessage(errors.GeneralError, err, "Failed ReadFile")
	}
	return string(fileData), nil
}

// GetPeerTLSRootCert returns tls root certs from peer config
func (c *config) getPeerTLSRootCert() ([]string, error) {

	rootCertLocation := c.peerConfig.GetString("peer.tls.rootcert.file")
	if rootCertLocation == "" {
		return make([]string, 0), nil
	}

	fileData, err := ioutil.ReadFile(c.translatePeerPath(rootCertLocation))
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed ReadFile")
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
func (c *config) GetNamedClientOverride() (map[string]*httpsnapApi.ClientTLS, error) {
	var clientTLS map[string]*httpsnapApi.ClientTLS
	err := c.httpSnapConfig.UnmarshalKey("tls.namedClientOverride", &clientTLS)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed UnmarshalKey")
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
		return nil, errors.Errorf(errors.GeneralError, "Schema configuration for content-type: %s not found", contentType)
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

func (c *config) GetCryptoProvider() (string, error) {
	cryptoProvider := c.peerConfig.GetString("peer.BCCSP.Default")
	if cryptoProvider == "" {
		return "", errors.New(errors.GeneralError, "BCCSP Default provider not found")
	}
	return cryptoProvider, nil
}

// substGoPath replaces instances of '$GOPATH' with the GOPATH. If the system
// has multiple GOPATHs then the first is used.
func substGoPath(s string) string {
	gpDefault := build.Default.GOPATH
	gps := filepath.SplitList(gpDefault)

	return strings.Replace(s, "$GOPATH", gps[0], -1)
}
