/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"fmt"
	"go/build"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/spf13/viper"
)

const (
	configFileName                  = "config"
	peerConfigFileName              = "core"
	cmdRootPrefix                   = "core"
	defaultSelectionMaxAttempts     = 1
	defaultSelectionInterval        = time.Second
	defaultEndorsementMaxAttempts   = 3
	defaultEndorsementRetryInterval = 2 * time.Second
	defaultCommitTimeout            = 30 * time.Second
)

var logger = logging.NewLogger("txn-snap-config")
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
		return nil, errors.Errorf("Fatal error reading peer config file: %s", err)
	}
	//txSnapConfig
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"),
		PeerID: peerConfig.GetString("peer.id"), AppName: "txnsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, errors.New("Cannot create cache instance")
	}
	dataConfig, err := cacheInstance.Get(channelID, key)
	if err != nil {
		return nil, err
	}
	if dataConfig == nil {
		return nil, fmt.Errorf("config data is empty")
	}
	txnSnapConfig := viper.New()
	txnSnapConfig.SetConfigType("YAML")
	txnSnapConfig.ReadConfig(bytes.NewBuffer(dataConfig))
	txnSnapConfig.SetEnvPrefix(cmdRootPrefix)
	txnSnapConfig.AutomaticEnv()
	txnSnapConfig.SetEnvKeyReplacer(replacer)

	c := &Config{peerConfig: peerConfig, txnSnapConfig: txnSnapConfig, txnSnapConfigBytes: dataConfig}
	err = c.initializeLogging()
	if err != nil {
		return nil, errors.Errorf("Error initializing logging: %s", err)
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
		return nil, errors.Errorf("Peer address not found in config")
	}
	eventAddress := c.peerConfig.GetString("peer.events.address")
	if eventAddress == "" {
		return nil, errors.Errorf("Peer event address not found in config")
	}
	splitPeerAddress := strings.Split(peerAddress, ":")
	peer.Host = c.GetGRPCProtocol() + splitPeerAddress[0]
	peer.Port, err = strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, err
	}
	splitEventAddress := strings.Split(eventAddress, ":")
	// Event host should be set to the peer host as that is the advertised address
	peer.EventHost = c.GetGRPCProtocol() + splitPeerAddress[0]
	peer.EventPort, err = strconv.Atoi(splitEventAddress[1])
	if err != nil {
		return nil, err
	}
	peer.MSPid = []byte(c.GetMspID())
	if peer.MSPid == nil || string(peer.MSPid) == "" {
		return nil, errors.Errorf("Peer localMspId not found in config")
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

// GetTLSCertPath returns absolute path to the TLS certificate
func (c *Config) GetTLSCertPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.cert.file"))
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

// GetEndorsementMaxAttempts returns the maximum number of attempts
// at successful endorsements.
func (c *Config) GetEndorsementMaxAttempts() int {
	maxAttempts := c.txnSnapConfig.GetInt("txnsnap.endorsement.maxattempts")
	if maxAttempts == 0 {
		return defaultEndorsementMaxAttempts
	}
	return maxAttempts
}

// GetEndorsementRetryInterval is the amount of time to wait between
// attempts at endorsements.
func (c *Config) GetEndorsementRetryInterval() time.Duration {
	interval := c.txnSnapConfig.GetDuration("txnsnap.endorsement.interval")
	if interval == 0 {
		return defaultEndorsementRetryInterval
	}
	return interval
}

// GetCommitTimeout is the amount of time to wait for a commit.
func (c *Config) GetCommitTimeout() time.Duration {
	interval := c.txnSnapConfig.GetDuration("txnsnap.commit.timeout")
	if interval == 0 {
		return defaultCommitTimeout
	}
	return interval
}

// initializeLogging initializes the logger
func (c *Config) initializeLogging() error {
	logLevel := c.txnSnapConfig.GetString("txnsnap.loglevel")

	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return errors.Errorf("Error initializing log level: %s", err)
	}

	logging.SetLevel("", level)
	logger.Debugf("Txnsnap logging initialized. Log level: %s", logging.GetLevel(""))

	return nil
}
