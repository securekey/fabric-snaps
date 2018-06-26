/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

var logger = logging.NewLogger("confgurationsnap")
var defaultLogLevel = "info"
var defaultRefreshInterval = 10 * time.Second
var minimumRefreshInterval = 5 * time.Second

const (
	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
)

//Config contains the configuration for the config snap
type Config struct {
	// PeerID is the local ID of the peer
	PeerID string
	// PeerMspID is the MSP ID of the local peer
	PeerMspID string
	//cache refresh interval
	RefreshInterval  time.Duration
	ConfigSnapConfig *viper.Viper
}

//CSRConfig used to pass CSR configuration parameters
type CSRConfig struct {
	CommonName     string
	Country        string
	StateProvince  string
	Locality       string
	Org            string
	OrgUnit        string
	DNSNames       []string
	EmailAddresses []string
	IPAddresses    []net.IP
}

// New returns a new config snap configuration for the given channel
func New(channelID, peerConfigPathOverride string) (*Config, error) {

	peerConfig, err := newPeerViper(peerConfigPathOverride)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error reading peer config")
	}

	peerID := peerConfig.GetString("peer.id")
	mspID := peerConfig.GetString("peer.localMspId")
	//configuration snap config
	key := configmanagerApi.ConfigKey{MspID: peerConfig.GetString("peer.localMspId"),
		PeerID: peerConfig.GetString("peer.id"), AppName: "configurationsnap"}
	cacheInstance := configmgmtService.GetInstance()
	if cacheInstance == nil {
		return nil, errors.New(errors.GeneralError, "Cannot create cache instance")

	}
	var refreshInterval = defaultRefreshInterval
	var customConfig *viper.Viper
	var dirty = true
	var dataConfig []byte
	if channelID != "" {
		logger.Debugf("Getting config for channel: %s", channelID)

		dataConfig, dirty, err = cacheInstance.Get(channelID, key)
		if err != nil {
			return nil, err
		}
		if dataConfig == nil {
			return nil, errors.New(errors.GeneralError, "config data is empty")
		}
		replacer := strings.NewReplacer(".", "_")
		customConfig = viper.New()
		customConfig.SetConfigType("YAML")
		err = customConfig.ReadConfig(bytes.NewBuffer(dataConfig))
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "snap_config_init_error")
		}
		customConfig.SetEnvPrefix(envPrefix)
		customConfig.AutomaticEnv()
		customConfig.SetEnvKeyReplacer(replacer)
		refreshInterval = customConfig.GetDuration("cache.refreshInterval")
		if err != nil {
			logger.Debugf("Cannot convert refresh interval to int")
			//use default value
			if refreshInterval < minimumRefreshInterval {
				refreshInterval = minimumRefreshInterval
			}
		}

	}
	logger.Debugf("Refresh Intrval: %.0f", refreshInterval)

	// Initialize from peer config
	config := &Config{
		PeerID:           peerID,
		PeerMspID:        mspID,
		RefreshInterval:  refreshInterval,
		ConfigSnapConfig: customConfig,
	}
	if dirty {
		err = config.initializeLogging()
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "Error initializing logging")
		}
	}
	return config, nil
}

// initializeLogging initializes the loggerconfig
func (c *Config) initializeLogging() error {
	if c.ConfigSnapConfig == nil {
		return nil
	}
	logLevel := c.ConfigSnapConfig.GetString("configsnap.loglevel")

	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Error initializing log level")
	}

	logging.SetLevel("configsnap", level)
	logger.Debugf("Confignap logging initialized. Log level: %s", logLevel)

	return nil
}

//GetPeerMSPID returns peerMspID
func GetPeerMSPID(peerConfigPathOverride string) (string, error) {

	peerConfig, err := newPeerViper(peerConfigPathOverride)
	if err != nil {
		return "", errors.New(errors.GeneralError, "Error reading peer config")
	}

	mspID := peerConfig.GetString("peer.localMspId")
	return mspID, nil

}

//GetPeerID returns peerID
func GetPeerID(peerConfigPathOverride string) (string, error) {
	peerConfig, err := newPeerViper(peerConfigPathOverride)
	if err != nil {
		return "", errors.New(errors.GeneralError, "Error reading peer config:PeerID")
	}
	peerID := peerConfig.GetString("peer.id")
	return peerID, nil
}

//GetBCCSPProvider get default BCCSP provider from the peer config
func GetBCCSPProvider(peerConfigPathOverride string) (string, error) {

	peerConfig, err := newPeerViper(peerConfigPathOverride)
	if err != nil {
		return "", errors.New(errors.GeneralError, "Error reading peer config:PeerID")
	}
	bccspProvider := peerConfig.GetString("peer.BCCSP.Default")
	logger.Debugf("Configured BCCSP provider: [%s]", bccspProvider)
	return bccspProvider, nil
}

func getMyConfig(channelID string, peerConfigPath string) (*viper.Viper, error) {
	peerMspID, err := GetPeerMSPID(peerConfigPath)
	if err != nil {
		return nil, err
	}
	peerID, err := GetPeerID(peerConfigPath)
	if err != nil {
		return nil, err
	}
	configKey := configmanagerApi.ConfigKey{MspID: peerMspID, PeerID: peerID, AppName: "configurationsnap"}
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)

	csconfig, _, err := instance.GetViper(channelID, configKey, configmanagerApi.YAML)
	if err != nil {
		return nil, err
	}
	if csconfig == nil {
		errMsg := fmt.Sprintf("Trying to get config for channel [%s], msp [%s], peer [%s] and app [configurationsnap]", channelID, peerMspID, peerID)
		logger.Debugf(errMsg)
		return nil, errors.New(errors.GeneralError, errMsg)
	}
	return csconfig, nil
}

//GetCSRConfigOptions to pass CSR config opts
func GetCSRConfigOptions(channelID string, peerConfigPath string) (*CSRConfig, error) {
	csrConfig := CSRConfig{}

	csconfig, err := getMyConfig(channelID, peerConfigPath)
	if err != nil {
		return nil, err
	}

	csrConfig.CommonName = csconfig.GetString("csr.cn")
	csrConfig.Country = csconfig.GetString("csr.names.country")
	csrConfig.Locality = csconfig.GetString("csr.names.locality")
	csrConfig.Org = csconfig.GetString("csr.names.org")
	csrConfig.OrgUnit = csconfig.GetString("csr.names.orgunit")
	csrConfig.StateProvince = csconfig.GetString("csr.names.stateprovince")
	csrConfig.DNSNames = csconfig.GetStringSlice("csr.alternativenames.DNSNames")
	csrConfig.EmailAddresses = csconfig.GetStringSlice("csr.alternativenames.EmailAddresses")
	ipaddresses := csconfig.GetStringSlice("csr.alternativenames.IPAddresses")
	var netAddrs []net.IP
	for _, v := range ipaddresses {
		if ip := net.ParseIP(v); ip != nil {

			netAddrs = append(netAddrs, ip)
		}
	}
	csrConfig.IPAddresses = netAddrs

	return &csrConfig, nil

}

//GetDefaultRefreshInterval get default interval
func GetDefaultRefreshInterval() time.Duration {
	return defaultRefreshInterval
}

//GetMinimumRefreshInterval get minimum refresh interval
func GetMinimumRefreshInterval() time.Duration {
	return minimumRefreshInterval
}

func newPeerViper(peerConfigPathOverride string) (*viper.Viper, error) {
	var peerConfigPath string
	if peerConfigPathOverride == "" {
		peerConfigPath = defaultPeerConfigPath
	} else {
		peerConfigPath = peerConfigPathOverride
	}
	peerViper := viper.New()
	peerViper.AddConfigPath(peerConfigPath)
	peerViper.SetConfigName(peerConfigName)
	peerViper.SetEnvPrefix(envPrefix)
	peerViper.AutomaticEnv()
	peerViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := peerViper.ReadInConfig(); err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "snap_config_init_error")
	}
	return peerViper, nil

}
