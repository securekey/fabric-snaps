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

	logging "github.com/op/go-logging"
	"github.com/spf13/viper"
)

const (
	configFileName     = "config"
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
)

var peerConfig = viper.New()
var logger = logging.MustGetLogger("txn-snap-config")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// Init configuration and logging for txn snap. By default, the we look for
// configuration files at a path described by the environment variable
// "FABRIC_CFG_PATH". This is where the configuration is expected to be set in
// a production image. For testing and development, a GOPATH, project
// relative path is used. Optionally, a path override parameter can be passed in
// @param {string} [OPTIONAL] configPathOverride
// @returns {error} error, if any
func Init(configPathOverride string) error {

	replacer := strings.NewReplacer(".", "_")
	configPath := "./"
	peerConfigPath := "/etc/hyperledger/fabric"

	if configPathOverride != "" {
		configPath = configPathOverride
		peerConfigPath = configPathOverride
	}
	//txnSnap Config
	viper.AddConfigPath(configPath)
	viper.SetConfigName(configFileName)
	viper.SetEnvPrefix(cmdRootPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(replacer)

	//peer Config
	peerConfig.AddConfigPath(peerConfigPath)
	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(replacer)

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error reading config file: %s", err)
	}

	err = peerConfig.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error reading config file: %s", err)
	}

	err = initializeLogging()
	if err != nil {
		return fmt.Errorf("Error initializing logging: %s", err)
	}

	return nil
}

// GetLocalPeer returns address and ports for the peer running inside the
// txn snap container
func GetLocalPeer() (*PeerConfig, error) {
	var peer = &PeerConfig{}
	var err error

	peerAddress := peerConfig.GetString("peer.address")
	if peerAddress == "" {
		return nil, fmt.Errorf("Peer address not found in config")
	}
	eventAddress := peerConfig.GetString("peer.events.address")
	if eventAddress == "" {
		return nil, fmt.Errorf("Peer event address not found in config")
	}
	splitPeerAddress := strings.Split(peerAddress, ":")
	peer.Host = GetGRPCProtocol() + splitPeerAddress[0]
	peer.Port, err = strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, err
	}
	splitEventAddress := strings.Split(eventAddress, ":")
	peer.EventHost = GetGRPCProtocol() + splitEventAddress[0]
	peer.EventPort, err = strconv.Atoi(splitEventAddress[1])
	if err != nil {
		return nil, err
	}
	peer.MSPid = []byte(GetMspID())
	if peer.MSPid == nil || string(peer.MSPid) == "" {
		return nil, fmt.Errorf("Peer localMspId not found in config")
	}

	return peer, nil
}

// GetMspID returns the MSP ID for the local peer
func GetMspID() string {
	return peerConfig.GetString("peer.localMspId")
}

// GetTLSRootCertPath returns absolute path to the TLS root certificate
func GetTLSRootCertPath() string {
	return GetConfigPath(peerConfig.GetString("peer.tls.rootcert.file"))
}

// GetTLSCertPath returns absolute path to the TLS certificate
func GetTLSCertPath() string {
	return GetConfigPath(peerConfig.GetString("peer.tls.cert.file"))
}

// GetTLSKeyPath returns absolute path to the TLS key
func GetTLSKeyPath() string {
	return GetConfigPath(peerConfig.GetString("peer.tls.key.file"))
}

// GetMembershipPollInterval get membership pollinterval
func GetMembershipPollInterval() time.Duration {
	return viper.GetDuration("txnsnap.membership.pollinterval")
}

// GetGRPCProtocol returns 'grpcs://' or "grpc://" based on if tls is enabled or disabled
func GetGRPCProtocol() string {
	if viper.GetBool("peer.tls.enabled") {
		return "grpcs://"
	}
	return "grpc://"
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

// initializeLogging initializes the logger
func initializeLogging() error {
	level, err := logging.LogLevel(viper.GetString("txnsnap.loglevel"))
	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetLevel(level, "")                // default module
	logging.SetLevel(level, "txn-snap-config") // this current file's module
	logger.Debugf("txnsnap Logger initialized. Default Log level: %s, txn-snap-config Log level: %s", logging.GetLevel(""), logging.GetLevel("txn-snap-config"))

	return nil
}
