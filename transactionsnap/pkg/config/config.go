/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

const (
	configFileName              = "config"
	peerConfigFileName          = "core"
	cmdRootPrefix               = "core"
	defaultSelectionMaxAttempts = 1
	defaultSelectionInterval    = time.Second
	defaultHandlerTimeout       = 30 * time.Second
)

var logger = logging.NewLogger("txnsnap")
var defaultLogLevel = "info"

//Config implements Config interface
type Config struct {
	peerConfig         *viper.Viper
	txnSnapConfig      *viper.Viper
	txnSnapConfigBytes []byte
}

//NewConfig returns config struct
func NewConfig(peerConfigPath string, channelID string) (transactionsnapApi.Config, error) {
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
		return nil, errors.WithMessage(errors.GeneralError, err, "Fatal error reading peer config file")
	}
	//txSnapConfig
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"),
		PeerID: peerConfig.GetString("peer.id"), AppName: "txnsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, errors.New(errors.GeneralError, "Cannot create cache instance")
	}
	dataConfig, err := cacheInstance.Get(channelID, key)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed cacheInstance")
	}
	if dataConfig == nil {
		return nil, errors.New(errors.GeneralError, "config data is empty")
	}
	txnSnapConfig := viper.New()
	txnSnapConfig.SetConfigType("YAML")
	err = txnSnapConfig.ReadConfig(bytes.NewBuffer(dataConfig))
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "snap_config_init_error")
	}
	txnSnapConfig.SetEnvPrefix(cmdRootPrefix)
	txnSnapConfig.AutomaticEnv()
	txnSnapConfig.SetEnvKeyReplacer(replacer)

	c := &Config{peerConfig: peerConfig, txnSnapConfig: txnSnapConfig, txnSnapConfigBytes: dataConfig}
	err = c.initializeLogging()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error initializing logging")
	}
	return c, nil
}

//GetConfigBytes returns config bytes
func (c *Config) GetConfigBytes() []byte {
	return c.txnSnapConfigBytes
}

// GetLocalPeer returns address and ports for the peer running inside the
// txn snap container
func (c *Config) GetLocalPeer() (*transactionsnapApi.PeerConfig, error) {
	var peer = &transactionsnapApi.PeerConfig{}
	var err error

	peerAddress := c.peerConfig.GetString("peer.address")
	if peerAddress == "" {
		return nil, errors.New(errors.GeneralError, "Peer address not found in config")
	}
	eventAddress := c.peerConfig.GetString("peer.events.address")
	if eventAddress == "" {
		return nil, errors.New(errors.GeneralError, "Peer event address not found in config")
	}
	splitPeerAddress := strings.Split(peerAddress, ":")
	peer.Host = c.GetGRPCProtocol() + splitPeerAddress[0]
	peer.Port, err = strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed strconv.Atoi")
	}
	splitEventAddress := strings.Split(eventAddress, ":")
	// Event host should be set to the peer host as that is the advertised address
	peer.EventHost = c.GetGRPCProtocol() + splitPeerAddress[0]
	peer.EventPort, err = strconv.Atoi(splitEventAddress[1])
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed strconv.Atoi")
	}
	peer.MSPid = []byte(c.GetMspID())
	if peer.MSPid == nil || string(peer.MSPid) == "" {
		return nil, errors.New(errors.GeneralError, "Peer localMspId not found in config")
	}

	return peer, nil
}

// GetMspID returns the MSP ID for the local peer
func (c *Config) GetMspID() string {
	return c.peerConfig.GetString("peer.localMspId")
}

//GetMspConfigPath returns the MSP config path for peer
func (c *Config) GetMspConfigPath() string {
	return substGoPath(c.peerConfig.GetString("peer.mspConfigPath"))
}

// substGoPath replaces instances of '$GOPATH' with the GOPATH. If the system
// has multiple GOPATHs then the first is used.
func substGoPath(s string) string {
	gpDefault := build.Default.GOPATH
	gps := filepath.SplitList(gpDefault)

	return strings.Replace(s, "$GOPATH", gps[0], -1)
}

// GetTLSRootCertPath returns absolute path to the TLS root certificate
func (c *Config) GetTLSRootCertPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.rootcert.file"))
}

// GetTLSRootCert returns root TLS certificate
func (c *Config) GetTLSRootCert() *x509.Certificate {
	certPath := c.GetTLSRootCertPath()
	return getCertFromPath(certPath)
}

func getCertPemFromPath(certPath string) []byte {
	pemBuffer, err := ioutil.ReadFile(certPath)
	if err != nil {
		logger.Warnf("cert fixture missing at path '%s', err: %s", certPath, err)
		return nil
	}
	return pemBuffer
}

func getCertFromPath(certPath string) *x509.Certificate {
	pemBuffer, err := ioutil.ReadFile(certPath)
	if err != nil {
		logger.Warnf("cert fixture missing at path '%s', err: %s", certPath, err)
		return nil
	}

	certBlock, _ := pem.Decode(pemBuffer)
	if certBlock == nil {
		logger.Warnf("failed to decode certificate bytes [%v]", pemBuffer)
		return nil
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		logger.Warnf("failed to parse certificate: %s", err)
		return nil
	}

	return cert
}

// GetTLSCertPath returns absolute path to the TLS certificate
func (c *Config) GetTLSCertPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.cert.file"))
}

// GetTLSCert returns client TLS certificate
func (c *Config) GetTLSCert() *x509.Certificate {
	certPath := c.GetTLSCertPath()
	return getCertFromPath(certPath)
}

// GetTLSCertPem returns client TLS certificate pem
func (c *Config) GetTLSCertPem() []byte {
	certPath := c.GetTLSCertPath()
	return getCertPemFromPath(certPath)
}

// GetTLSKeyPath returns absolute path to the TLS key
func (c *Config) GetTLSKeyPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.key.file"))
}

// GetGRPCProtocol to get grpc protocol
func (c *Config) GetGRPCProtocol() string {
	if c.peerConfig.GetBool("peer.tls.enabled") {
		return "grpcs://"
	}
	return "grpc://"
}

// GetConfigPath returns the absolute value of the given path that is
// relative to the config file
// For example, if the config file is at /etc/hyperledger/config.yaml,
// calling GetConfigPath("tls/cert") will return /etc/hyperledger/tls/cert
func (c *Config) GetConfigPath(path string) string {
	basePath := filepath.Dir(c.txnSnapConfig.ConfigFileUsed())

	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(basePath, path)
}

//GetPeerConfig to get peers
func (c *Config) GetPeerConfig() *viper.Viper {
	return c.peerConfig
}

//GetTxnSnapConfig returns txnSnapConfig
func (c *Config) GetTxnSnapConfig() *viper.Viper {
	return c.txnSnapConfig
}

//GetCryptoProvider returns crypto provider name from peer config
func (c *Config) GetCryptoProvider() (string, error) {
	cryptoProvider := c.peerConfig.GetString("peer.BCCSP.Default")
	if cryptoProvider == "" {
		return "", errors.New(errors.GeneralError, "BCCSP Default provider not found")
	}
	return cryptoProvider, nil
}

// GetEndorserSelectionMaxAttempts returns the maximum number of attempts
// at retrieving at least one endorsing peer group, while waiting the
// specified interval between attempts.
func (c *Config) GetEndorserSelectionMaxAttempts() int {
	maxAttempts := c.txnSnapConfig.GetInt("txnsnap.selection.maxattempts")
	if maxAttempts == 0 {
		return defaultSelectionMaxAttempts
	}
	return maxAttempts
}

// GetEndorserSelectionInterval is the amount of time to wait between
// attempts at retrieving at least one endorsing peer group.
func (c *Config) GetEndorserSelectionInterval() time.Duration {
	interval := c.txnSnapConfig.GetDuration("txnsnap.selection.interval")
	if interval == 0 {
		return defaultSelectionInterval
	}
	return interval
}

// GetHandlerTimeout is the amount of time to wait for sdk handler
func (c *Config) GetHandlerTimeout() time.Duration {
	interval := c.txnSnapConfig.GetDuration("txnsnap.handler.timeout")
	if interval == 0 {
		return defaultHandlerTimeout
	}
	return interval
}

// RetryOpts transaction snap retry options
func (c *Config) RetryOpts() retry.Opts {
	attempts := c.txnSnapConfig.GetInt("txnsnap.retry.attempts")
	initialBackoff := c.txnSnapConfig.GetDuration("txnsnap.retry.initialbackoff")
	maxBackoff := c.txnSnapConfig.GetDuration("txnsnap.retry.maxbackoff")
	factor := c.txnSnapConfig.GetFloat64("txnsnap.retry.backofffactor")

	if attempts == 0 {
		attempts = retry.DefaultAttempts
	}
	if initialBackoff == 0 {
		initialBackoff = retry.DefaultInitialBackoff
	}
	if maxBackoff == 0 {
		maxBackoff = retry.DefaultMaxBackoff
	}
	if factor == 0 {
		factor = retry.DefaultBackoffFactor
	}

	return retry.Opts{
		Attempts:       attempts,
		InitialBackoff: initialBackoff,
		MaxBackoff:     maxBackoff,
		BackoffFactor:  factor,
	}
}

// initializeLogging initializes the logger
func (c *Config) initializeLogging() error {
	logLevel := c.txnSnapConfig.GetString("txnsnap.loglevel")

	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Error initializing log level")
	}

	logging.SetLevel("txnsnap", level)
	logger.Debugf("Txnsnap logging initialized. Log level: %s", logLevel)

	return nil
}
