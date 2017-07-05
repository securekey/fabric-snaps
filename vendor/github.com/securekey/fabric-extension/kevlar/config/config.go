/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package config

import (
	"fmt"
	"os"
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
	devConfigPath      = "$GOPATH/src/github.com/securekey/fabric-extension/kevlar/sampleconfig"
)

var peerConfig = viper.New()
var logger = logging.MustGetLogger("fmp-config")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

// Init configuration and logging for Kevlar. By default, the we look for
// configuration files at a path described by the environment variable
// "FABRIC_CFG_PATH". This is where the configuration is expected to be set in
// a production Kevlar image. For testing and development, a GOPATH, project
// relative path is used. Optionally, a path override parameter can be passed in
// @param {string} [OPTIONAL] configPathOverride
// @returns {error} error, if any
func Init(configPathOverride string) error {
	var configPath = os.Getenv("FABRIC_CFG_PATH")
	replacer := strings.NewReplacer(".", "_")

	if configPath != "" {
		viper.AddConfigPath(configPath)
		peerConfig.AddConfigPath(configPath)
	} else {
		if configPathOverride == "" {
			configPathOverride = devConfigPath
		}
		viper.AddConfigPath(configPathOverride)
		peerConfig.AddConfigPath(configPathOverride)
	}
	viper.SetConfigName(configFileName)
	viper.SetEnvPrefix(cmdRootPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(replacer)

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
// kevlar container
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
	peer.Host = splitPeerAddress[0]
	peer.Port, err = strconv.Atoi(splitPeerAddress[1])
	if err != nil {
		return nil, err
	}
	splitEventAddress := strings.Split(eventAddress, ":")
	peer.EventHost = splitEventAddress[0]
	peer.EventPort, err = strconv.Atoi(splitEventAddress[1])
	if err != nil {
		return nil, err
	}
	peer.MSPid = []byte(peerConfig.GetString("peer.localMspId"))
	if peer.MSPid == nil || string(peer.MSPid) == "" {
		return nil, fmt.Errorf("Peer localMspId not found in config")
	}

	return peer, nil
}

// IsTLSEnabled is TLS enabled?
func IsTLSEnabled() bool {
	return peerConfig.GetBool("peer.tls.enabled")
}

func GetMspConfigPath() string {
	return GetConfigPath(peerConfig.GetString("peer.mspConfigPath"))
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

// GetEnrolmentCertPath returns absolute path to the Enrolment cert
func GetEnrolmentCertPath() string {
	return GetConfigPath(viper.GetString("kevlar.enrolment.cert.file"))
}

// GetEnrolmentKeyPath returns absolute path to the Enrolment key
func GetEnrolmentKeyPath() string {
	return GetConfigPath(viper.GetString("kevlar.enrolment.key.file"))
}

// GetEDSClientCertPath returns absolute path to the cert for external data sources
func GetEDSClientCertPath() string {
	return GetConfigPath(viper.GetString("edsservice.tls.clientCert"))
}

// GetEDSClientKeyPath returns absolute path to the key for external data sources
func GetEDSClientKeyPath() string {
	return GetConfigPath(viper.GetString("edsservice.tls.clientKey"))
}

// GetEDSCaCertPath returns absolute path to the ca cert for external data sources
func GetEDSCaCertPath() string {
	return GetConfigPath(viper.GetString("edsservice.tls.caCert"))
}

// GetEDSOverridePath returns absolute path to client's credentials override folder
func GetEDSOverridePath() string {
	return GetConfigPath(viper.GetString("edsservice.tls.overridePath"))
}

// GetKevlarServerPort returns kevlar server port
func GetKevlarServerPort() string {
	return viper.GetString("kevlar.server.port")
}

// GetKevlarEventPort returns kevlar event port
func GetKevlarEventPort() string {
	return viper.GetString("kevlar.event.port")
}

// GetKevlarInvokeServerPort returns kevlar invoke server port
func GetKevlarInvokeServerPort() string {
	return viper.GetString("kevlar.invokeserver.port")
}

func GetMembershipPollInterval() time.Duration {
	return viper.GetDuration("kevlar.membership.pollinterval")
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

func initializeLogging() error {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	level, err := logging.LogLevel(viper.GetString("kevlar.loglevel"))
	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetBackend(backendFormatter).SetLevel(level, "")

	logger.Debugf("Kevlar Logger initialized. Log level: %s", logging.GetLevel(""))

	return nil
}
