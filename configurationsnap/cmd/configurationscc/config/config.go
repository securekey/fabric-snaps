/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("configurationscc/config")
var defaultRefreshInterval time.Duration = 10

const (
	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
	//	defaultRefreshInterval = 10
	customConfigName = "config"
)

// Config contains the configuration for the config snap
type Config struct {
	// PeerID is the local ID of the peer
	PeerID string
	// PeerMspID is the MSP ID of the local peer
	PeerMspID string
	//cache refresh interval
	RefreshInterval time.Duration
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
	//configuration snap config
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"),
		PeerID: peerConfig.GetString("peer.id"), AppName: "configurationsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, errors.New("Cannot create cache instance")
	}
	var refreshInterval = defaultRefreshInterval
	if channelID != "" {
		log.Debug("Getting config for channel: %s", channelID)

		dataConfig, err := cacheInstance.Get(channelID, key)
		if err != nil {
			return nil, err
		}
		if dataConfig == nil {
			return nil, fmt.Errorf("config data is empty")
		}
		replacer := strings.NewReplacer(".", "_")
		customConfig := viper.New()
		customConfig.SetConfigType("YAML")
		customConfig.ReadConfig(bytes.NewBuffer(dataConfig))
		customConfig.SetEnvPrefix(envPrefix)
		customConfig.AutomaticEnv()
		customConfig.SetEnvKeyReplacer(replacer)
		refreshInterval = customConfig.GetDuration("cache.refreshInterval")
		if err != nil {
			log.Warning("Cannot convert refresh interval to int")
			//use default value
			refreshInterval = defaultRefreshInterval
		}
	}
	log.Debug("Refresh Intrval: %d", refreshInterval)

	// Initialize from peer config
	config := &Config{
		PeerID:          peerID,
		PeerMspID:       mspID,
		RefreshInterval: refreshInterval,
	}

	return config, nil
}

//GetPeerMSPID returns peerMspID
func GetPeerMSPID(peerConfigPathOverride string) (string, error) {
	var peerConfigPath string
	if peerConfigPathOverride == "" {
		peerConfigPath = defaultPeerConfigPath
	} else {
		peerConfigPath = peerConfigPathOverride
	}
	peerConfig, err := newPeerViper(peerConfigPath)
	if err != nil {
		return "", errors.Wrapf(err, "error reading peer config")
	}

	mspID := peerConfig.GetString("peer.localMspId")
	fmt.Printf("returning local mspId %s", mspID)
	return mspID, nil

}

//GetDefaultRefreshInterval get fdefault interval
func GetDefaultRefreshInterval() time.Duration {
	return defaultRefreshInterval
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
