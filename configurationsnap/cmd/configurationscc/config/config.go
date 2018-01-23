/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/log"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("configurationscc/config")
var defaultRefreshInterval = 10 * time.Second
var minimumRefreshInterval = 5 * time.Second

const (
	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
	customConfigName      = "config"
)

//Config contains the configuration for the config snap
type Config struct {
	// PeerID is the local ID of the peer
	PeerID string
	// PeerMspID is the MSP ID of the local peer
	PeerMspID string
	//cache refresh interval
	RefreshInterval time.Duration
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
			if refreshInterval < minimumRefreshInterval {
				refreshInterval = minimumRefreshInterval
			}
		}

	}
	log.Debug("Refresh Intrval: %.0f", refreshInterval)

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
	return mspID, nil

}

//GetPeerID returns peerID
func GetPeerID(peerConfigPathOverride string) (string, error) {
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
	peerID := peerConfig.GetString("peer.id")
	return peerID, nil
}

//GetBCCSPOpts to get bccsp options from configurationcc config
func GetBCCSPOpts(channelID string, peerConfigPath string) (*factory.FactoryOpts, error) {
	var cfgopts *factory.FactoryOpts
	peerMspID, err := GetPeerMSPID(peerConfigPath)
	if err != nil {
		return cfgopts, err
	}
	peerID, err := GetPeerID(peerConfigPath)
	if err != nil {
		return cfgopts, err
	}
	configKey := configmanagerApi.ConfigKey{MspID: peerMspID, PeerID: peerID, AppName: "configurationsnap"}
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)

	csconfig, err := instance.GetViper(channelID, configKey, configmanagerApi.YAML)
	if err != nil {
		return cfgopts, err
	}
	logger.Debugf("Config from HL for %v %v", configKey, csconfig)

	switch GetProvider(csconfig) {
	case "PKCS11":
		return getPKCSOptions(csconfig)
	case "PLUGIN":
		return getPluginOptions(csconfig)
	default:
		return nil, errors.Errorf("Provider '%s' is not supported", GetProvider(csconfig))
	}
}

func getPluginOptions(csconfig *viper.Viper) (*factory.FactoryOpts, error) {

	cfglib := GetLib(csconfig)
	cfg := csconfig.GetStringMap("BCCSP.Security.Config")
	logger.Debugf("BCCSP Plugin option config map %v", cfg)
	pluginOpt := factory.PluginOpts{
		Library: cfglib,
		Config:  cfg,
	}
	opts := &factory.FactoryOpts{
		ProviderName: "PLUGIN",
		PluginOpts:   &pluginOpt,
	}
	logger.Debugf("BCCSP Plugin option config map %v", cfg)
	return opts, nil
}

func getPKCSOptions(csconfig *viper.Viper) (*factory.FactoryOpts, error) {
	//from config file
	cfglib := GetLib(csconfig)
	logger.Debugf("Security library from config %s", cfglib)

	lib := FindPKCS11Lib(cfglib)
	if lib != "" {
		errors.Errorf("PKCS Lib path was not set")
	}
	pin := GetPin(csconfig)
	if pin != "" {
		errors.Errorf("PKCS PIN  was not set")
	}
	label := GetLabel(csconfig)
	if label != "" {
		errors.Errorf("PKCS Label  was not set")
	}
	ksopts := &pkcs11.FileKeystoreOpts{
		KeyStorePath: GetKeystorePath(csconfig),
	}
	pkcsOpt := pkcs11.PKCS11Opts{
		SecLevel:     GetLevel(csconfig),
		HashFamily:   GetHashAlg(csconfig),
		Ephemeral:    GetEphemeral(csconfig),
		Library:      lib,
		Pin:          pin,
		Label:        label,
		FileKeystore: ksopts,
	}
	logger.Debugf("Creating PKCS11 provider with options %v", pkcsOpt)
	opts := &factory.FactoryOpts{
		ProviderName: "PKCS11",
		Pkcs11Opts:   &pkcsOpt,
	}

	return opts, nil

}

//GetCSRConfigOptions to pass CSR config opts
func GetCSRConfigOptions(channelID string, peerConfigPath string) (*CSRConfig, error) {
	csrConfig := CSRConfig{}
	peerMspID, err := GetPeerMSPID(peerConfigPath)
	if err != nil {
		return &csrConfig, err
	}
	peerID, err := GetPeerID(peerConfigPath)
	if err != nil {
		return &csrConfig, err
	}
	configKey := configmanagerApi.ConfigKey{MspID: peerMspID, PeerID: peerID, AppName: "configurationsnap"}
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)

	csconfig, err := instance.GetViper(channelID, configKey, configmanagerApi.YAML)
	if err != nil {
		return &csrConfig, err
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

//GetProvider returns provider
func GetProvider(csconfig *viper.Viper) string {
	return csconfig.GetString("BCCSP.Security.Provider")
}

//GetHashAlg returns hash alg
func GetHashAlg(csconfig *viper.Viper) string {
	return csconfig.GetString("BCCSP.Security.HashAlgorithm")
}

//GetEphemeral returns ephemeral
func GetEphemeral(csconfig *viper.Viper) bool {
	return csconfig.GetBool("BCCSP.Security.Ephemeral")
}

//GetLevel returns level
func GetLevel(csconfig *viper.Viper) int {
	return csconfig.GetInt("BCCSP.Security.Level")
}

//GetPin returns pin
func GetPin(csconfig *viper.Viper) string {
	return csconfig.GetString("BCCSP.Security.Pin")
}

//GetLib returns lib
func GetLib(csconfig *viper.Viper) string {
	return csconfig.GetString("BCCSP.Security.Library")
}

//GetLabel returns label
func GetLabel(csconfig *viper.Viper) string {
	return csconfig.GetString("BCCSP.Security.Label")
}

//GetKeystorePath returns keystorePath
func GetKeystorePath(csconfig *viper.Viper) string {
	return csconfig.GetString("BCCSP.Security.KeystorePath")
}

//FindPKCS11Lib to check which one of configured libs exist for current ARCH
func FindPKCS11Lib(configuredLib string) string {
	logger.Debugf("PKCS library configurations paths  %s ", configuredLib)
	var lib string
	if configuredLib != "" {
		possibilities := strings.Split(configuredLib, ",")
		for _, path := range possibilities {
			trimpath := strings.TrimSpace(path)
			logger.Debugf("paths*** [%v] [%v]", path, trimpath)
			if _, err := os.Stat(trimpath); !os.IsNotExist(err) {
				lib = path
				break
			}
		}
	}
	logger.Debugf("Found pkcs library '%s'", lib)
	return lib
}

//GetDefaultRefreshInterval get default interval
func GetDefaultRefreshInterval() time.Duration {
	return defaultRefreshInterval
}

//GetMinimumRefreshInterval get minimum refresh interval
func GetMinimumRefreshInterval() time.Duration {
	return minimumRefreshInterval
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
