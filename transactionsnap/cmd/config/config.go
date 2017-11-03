/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/client"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/spf13/viper"
)

const (
	configFileName     = "config"
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
)

var logger = logging.MustGetLogger("txn-snap-config")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// config implements Config interface
type config struct {
	peerConfig    *viper.Viper
	txnSnapConfig *viper.Viper
}

// NewConfig return config struct
func NewConfig(configPathOverride string, stub shim.ChaincodeStubInterface) (transactionsnapApi.Config, error) {

	replacer := strings.NewReplacer(".", "_")
	configPath := "./"
	peerConfigPath := "/etc/hyperledger/fabric"

	if configPathOverride != "" {
		configPath = configPathOverride
		peerConfigPath = configPathOverride
	}
	//txnSnap Config
	txnSnapConfig := viper.New()
	txnSnapConfig.AddConfigPath(configPath)
	txnSnapConfig.SetConfigName(configFileName)
	txnSnapConfig.SetEnvPrefix(cmdRootPrefix)
	txnSnapConfig.AutomaticEnv()
	txnSnapConfig.SetEnvKeyReplacer(replacer)

	//peer Config
	peerConfig := viper.New()
	peerConfig.AddConfigPath(peerConfigPath)
	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(replacer)

	err := txnSnapConfig.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Fatal error reading config file: %s", err)
	}

	err = peerConfig.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Fatal error reading config file: %s", err)
	}

	txnSnapConfig, err = client.NewTempConfigClient(txnSnapConfig).Get(stub, &configmanagerApi.ConfigKey{Org: peerConfig.GetString("peer.localMspId"), Peer: peerConfig.GetString("peer.id"), Appname: "txnsnap"})
	if err != nil {
		return nil, fmt.Errorf("Fatal error from NewConfigClient: %s", err)
	}
	c := &config{peerConfig: peerConfig, txnSnapConfig: txnSnapConfig}
	err = c.initializeLogging()
	if err != nil {
		return nil, fmt.Errorf("Error initializing logging: %s", err)
	}
	return c, nil
}

// GetLocalPeer returns address and ports for the peer running inside the
// txn snap container
func (c *config) GetLocalPeer() (*transactionsnapApi.PeerConfig, error) {
	var peer = &transactionsnapApi.PeerConfig{}
	var err error

	peerAddress := c.peerConfig.GetString("peer.address")
	if peerAddress == "" {
		return nil, fmt.Errorf("Peer address not found in config")
	}
	eventAddress := c.peerConfig.GetString("peer.events.address")
	if eventAddress == "" {
		return nil, fmt.Errorf("Peer event address not found in config")
	}
	splitPeerAddress := strings.Split(peerAddress, ":")
	peer.Host = c.GetGRPCProtocol() + splitPeerAddress[0]
	peer.Port, err = strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, err
	}
	splitEventAddress := strings.Split(eventAddress, ":")
	peer.EventHost = c.GetGRPCProtocol() + splitEventAddress[0]
	peer.EventPort, err = strconv.Atoi(splitEventAddress[1])
	if err != nil {
		return nil, err
	}
	peer.MSPid = []byte(c.GetMspID())
	if peer.MSPid == nil || string(peer.MSPid) == "" {
		return nil, fmt.Errorf("Peer localMspId not found in config")
	}

	return peer, nil
}

// GetMspID returns the MSP ID for the local peer
func (c *config) GetMspID() string {
	return c.peerConfig.GetString("peer.localMspId")
}

// GetTLSRootCertPath returns absolute path to the TLS root certificate
func (c *config) GetTLSRootCertPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.rootcert.file"))
}

// GetTLSCertPath returns absolute path to the TLS certificate
func (c *config) GetTLSCertPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.cert.file"))
}

// GetTLSKeyPath returns absolute path to the TLS key
func (c *config) GetTLSKeyPath() string {
	return c.GetConfigPath(c.peerConfig.GetString("peer.tls.key.file"))
}

// GetMembershipPollInterval get membership pollinterval
func (c *config) GetMembershipPollInterval() time.Duration {
	return c.txnSnapConfig.GetDuration("txnsnap.membership.pollinterval")
}

func (c *config) GetGRPCProtocol() string {
	if viper.GetBool("peer.tls.enabled") {
		return "grpcs://"
	}
	return "grpc://"
}

// GetConfigPath returns the absolute value of the given path that is
// relative to the config file
// For example, if the config file is at /etc/hyperledger/config.yaml,
// calling GetConfigPath("tls/cert") will return /etc/hyperledger/tls/cert
func (c *config) GetConfigPath(path string) string {
	basePath := filepath.Dir(c.txnSnapConfig.ConfigFileUsed())

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(basePath, path)
}

func (c *config) GetPeerConfig() *viper.Viper {
	return c.peerConfig
}

// initializeLogging initializes the logger
func (c *config) initializeLogging() error {
	level, err := logging.LogLevel(c.txnSnapConfig.GetString("txnsnap.loglevel"))
	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetLevel(level, "")                // default module
	logging.SetLevel(level, "txn-snap-config") // this current file's module
	logger.Debugf("txnsnap Logger initialized. Default Log level: %s, txn-snap-config Log level: %s", logging.GetLevel(""), logging.GetLevel("txn-snap-config"))

	return nil
}
