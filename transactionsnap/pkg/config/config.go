/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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
	"github.com/securekey/fabric-snaps/util/configcache"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

const (
	peerConfigFileName                = "core"
	cmdRootPrefix                     = "core"
	defaultSelectionMaxAttempts       = 1
	defaultSelectionInterval          = time.Second
	defaultClientCacheRefreshInterval = 60 * time.Second
)

var logger = logging.NewLogger("txnsnap")
var peerConfigCache = configcache.New(peerConfigFileName, cmdRootPrefix, "/etc/hyperledger/fabric")

//Config implements Config interface
type Config struct {
	peerConfig         *viper.Viper
	txnSnapConfig      *viper.Viper
	txnSnapConfigBytes []byte
}

//NewConfig returns config struct
func NewConfig(peerConfigPath string, channelID string) (transactionsnapApi.Config, errors.Error) {

	peerConfig, err := peerConfigCache.Get(peerConfigPath)
	if err != nil {
		return nil, errors.WithMessage(errors.InitializeConfigError, err, "Failed to get peer config from cache")
	}
	//txSnapConfig
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"),
		PeerID: peerConfig.GetString("peer.id"), AppName: "txnsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, errors.New(errors.SystemError, "Cannot create cache instance")
	}
	//txn snap has its own cache and config hash checks, no need of dirty flag from config cache
	dataConfig, _, err := cacheInstance.Get(channelID, key)
	if err != nil {
		return nil, errors.WithMessage(errors.InitializeConfigError, err, fmt.Sprintf("Failed to get config cache for channel %s and key %s", channelID, key))
	}
	if dataConfig == nil {
		return nil, errors.New(errors.MissingConfigDataError, fmt.Sprintf("config data is empty for channel %s and key %s", channelID, key))
	}
	txnSnapConfig := viper.New()
	txnSnapConfig.SetConfigType("YAML")
	err = txnSnapConfig.ReadConfig(bytes.NewBuffer(dataConfig))
	if err != nil {
		return nil, errors.WithMessage(errors.InitializeConfigError, err, "snap_config_init_error")
	}
	txnSnapConfig.SetEnvPrefix(cmdRootPrefix)
	txnSnapConfig.AutomaticEnv()
	txnSnapConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	c := &Config{peerConfig: peerConfig, txnSnapConfig: txnSnapConfig, txnSnapConfigBytes: dataConfig}

	return c, nil
}

//GetConfigBytes returns config bytes
func (c *Config) GetConfigBytes() []byte {
	return c.txnSnapConfigBytes
}

// GetLocalPeer returns address and ports for the peer running inside the
// txn snap container
func (c *Config) GetLocalPeer() (*transactionsnapApi.PeerConfig, errors.Error) {
	var peer = &transactionsnapApi.PeerConfig{}
	var err error

	peerAddress := c.peerConfig.GetString("peer.address")
	if peerAddress == "" {
		return nil, errors.New(errors.PeerConfigError, "Peer address not found in config")
	}
	eventAddress := c.peerConfig.GetString("peer.events.address")
	if eventAddress == "" {
		return nil, errors.New(errors.PeerConfigError, "Peer event address not found in config")
	}
	splitPeerAddress := strings.Split(peerAddress, ":")
	peer.Host = splitPeerAddress[0]
	peer.Port, err = strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, errors.WithMessage(errors.PeerConfigError, err, "Failed strconv.Atoi")
	}
	splitEventAddress := strings.Split(eventAddress, ":")
	// Event host should be set to the peer host as that is the advertised address
	peer.EventHost = splitPeerAddress[0]
	peer.EventPort, err = strconv.Atoi(splitEventAddress[1])
	if err != nil {
		return nil, errors.WithMessage(errors.PeerConfigError, err, "Failed strconv.Atoi")
	}
	peer.MSPid = []byte(c.GetMspID())
	if peer.MSPid == nil || string(peer.MSPid) == "" {
		return nil, errors.New(errors.PeerConfigError, "Peer localMspId not found in config")
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
	pemBuffer, err := ioutil.ReadFile(certPath) // nolint: gas
	if err != nil {
		logger.Warnf("cert fixture missing at path '%s', err: %s", certPath, err)
		return nil
	}
	return pemBuffer
}

func getCertFromPath(certPath string) *x509.Certificate {
	pemBuffer, err := ioutil.ReadFile(certPath) // nolint: gas
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
func (c *Config) GetCryptoProvider() (string, errors.Error) {
	cryptoProvider := c.peerConfig.GetString("peer.BCCSP.Default")
	if cryptoProvider == "" {
		return "", errors.New(errors.CryptoConfigError, "BCCSP Default provider not found")
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

// GetClientCacheRefreshInterval the client cache refresh interval
func (c *Config) GetClientCacheRefreshInterval() time.Duration {
	interval := c.txnSnapConfig.GetDuration("txnsnap.cache.refreshInterval")
	if interval == 0 {
		return defaultClientCacheRefreshInterval
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

// CCErrorRetryableCodes configuration for chaincode errors to retry
func (c *Config) CCErrorRetryableCodes() ([]int32, errors.Error) {
	var codes []int32

	codeStrings := c.txnSnapConfig.GetStringSlice("txnsnap.retry.ccErrorCodes")
	for _, codeString := range codeStrings {
		code, err := strconv.Atoi(codeString)
		if err != nil {
			return nil, errors.WithMessage(errors.InvalidConfigDataError, err, fmt.Sprintf("could not parse cc error retry codes %s", codeStrings))
		}
		codes = append(codes, int32(code))
	}

	return codes, nil
}
