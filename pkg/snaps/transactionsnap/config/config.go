/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	configFileName     = "config"
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
	devConfigPath      = "$GOPATH/src/github.com/securekey/fabric-snaps/cmd/config/sampleconfig"
)

var peerConfig = viper.New()

// Init configuration and logging for Snaps. By default, the we look for
// configuration files at a path described by the environment variable
// "FABRIC_CFG_PATH". This is where the configuration is expected to be set in
// a production Snaps image. For testing and development, a GOPATH, project
// relative path is used. Optionally, a path override parameter can be passed in
// @param {string} [OPTIONAL] configPathOverride
// @returns {error} error, if any
func Init(configPathOverride string) error {
	var configPath = os.Getenv("FABRIC_CFG_PATH")
	replacer := strings.NewReplacer(".", "_")

	if configPath != "" {
		peerConfig.AddConfigPath(configPath)
	} else {
		if configPathOverride == "" {
			configPathOverride = devConfigPath
		}
		peerConfig.AddConfigPath(configPathOverride)
	}

	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(replacer)

	err := peerConfig.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error reading config file: %s", err)
	}

	return nil
}

// GetLocalPeer returns address and ports for the peer running inside the
// snap container
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

//GetMspConfigPath get member service config path
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
	return GetConfigPath(viper.GetString("client.enrolment.cert.file"))
}

// GetEnrolmentKeyPath returns absolute path to the Enrolment key
func GetEnrolmentKeyPath() string {
	return GetConfigPath(viper.GetString("client.enrolment.key.file"))
}

//GetMembershipPollInterval returns pool interval
func GetMembershipPollInterval() time.Duration {
	return viper.GetDuration("client.membership.pollinterval")
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
