/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"strings"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("configurationscc/config")

const (
	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
)

// Config contains the configuration for the config snap
type Config struct {
	// PeerID is the local ID of the peer
	PeerID string

	// PeerMspID is the MSP ID of the local peer
	PeerMspID string
}

// New returns a new config snap configuration for the given channel
func New(channelID, peerConfigPathOverride string) (*Config, error) {
	var peerConfigPath string
	if peerConfigPathOverride == "" {
		peerConfigPath = defaultPeerConfigPath
	} else {
		peerConfigPath = peerConfigPathOverride
	}

	peerConfig, err := newPeerViper(peerConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading peer config")
	}

	peerID := peerConfig.GetString("peer.id")
	mspID := peerConfig.GetString("peer.localMspId")

	// Initialize from peer config
	config := &Config{
		PeerID:    peerID,
		PeerMspID: mspID,
	}

	// TODO: Initalize channel-specific config for the configuration snap. (e.g. refresh interval, etc.)

	return config, nil
}

func newPeerViper(configPath string) (*viper.Viper, error) {
	peerViper := viper.New()
	peerViper.AddConfigPath(configPath)
	peerViper.SetConfigName(peerConfigName)
	peerViper.SetEnvPrefix(envPrefix)
	peerViper.AutomaticEnv()
	peerViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := peerViper.ReadInConfig(); err != nil {
		return nil, err
	}
	return peerViper, nil
}
