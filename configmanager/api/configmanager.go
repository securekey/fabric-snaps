/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

// VERSION of config data
const VERSION = "1"

// ConfigKey contain mspID,peerID,appname,appversion,componentname,componentversion
type ConfigKey struct {
	MspID            string
	PeerID           string
	AppName          string
	AppVersion       string
	ComponentName    string
	ComponentVersion string
}

//String return string value for config key
func (configKey *ConfigKey) String() string {
	return strings.Join([]string{configKey.MspID, configKey.PeerID, configKey.AppName, configKey.AppVersion, configKey.ComponentName, configKey.ComponentVersion}, "!")
}

//ConfigKV represents key value struct for managing configurations
type ConfigKV struct {
	Key   ConfigKey
	Value []byte
}

//ComponentConfig represents app component
type ComponentConfig struct {
	Name    string
	Config  string
	Version string
	TxID    string
}

//AppConfig identifier has application name , config version
type AppConfig struct {
	AppName    string
	Version    string
	Config     string
	Components []ComponentConfig
}

//PeerConfig identifier has peer identifier and collection of application configurations
type PeerConfig struct {
	PeerID string
	App    []AppConfig
}

//ConfigMessage - has MSP identifier and collection of peers
type ConfigMessage struct {
	MspID string
	Peers []PeerConfig
	Apps  []AppConfig
}

// ConfigClient is used to publish messages
type ConfigClient interface {
	Get(stub shim.ChaincodeStubInterface, configKey *ConfigKey) (viper *viper.Viper, err error)
}

//ConfigManager is used to manage configuration in ledger(save,get,delete)
type ConfigManager interface {
	//Save configuration - The submited payload should be in form of ConfigMessage
	Save(config []byte) errors.Error
	//Get configuration - Gets configuration based on config key.
	//For the valid config key retuned array will have only one element.
	//For the config key containing only MspID all configurations for that MspID will be returned
	Get(configKey ConfigKey) ([]*ConfigKV, errors.Error)
	//Delete configuration -
	//For the valid config one config message will be deleted
	//For the config key containing only MspID all configurations for that MspID will be deleted
	Delete(configKey ConfigKey) errors.Error
}

// ConfigType indicates the type (format) of the configuration
type ConfigType string

const (
	// YAML indicates that the configuration is in YAML format
	YAML ConfigType = "YAML"

	// JSON indicates that the configuration is in JSON format
	JSON ConfigType = "JSON"
)

//ConfigService configuration service interface
type ConfigService interface {
	//Get returns the config bytes along with dirty flag for the given channel and config key.
	// dirty flag bool returns true only if config is updated since its last retrieval
	Get(channelID string, configKey ConfigKey) ([]byte, bool, errors.Error)
	//GetViper returns a Viper instance along with dirty fla that wraps the config for the given channel and config key.
	// If the config key doesn't exist then nil is returned.
	//dirty flag bool returns true only if config is updated since its last retrieval
	GetViper(channelID string, configKey ConfigKey, configType ConfigType) (*viper.Viper, bool, errors.Error)
}

//IsValid validates config message
func (cm ConfigMessage) IsValid() errors.Error {
	if cm.MspID == "" {
		return errors.New(errors.InvalidConfigMessage, "MSPID cannot be empty")
	}

	if len(cm.Peers) == 0 && len(cm.Apps) == 0 {
		return errors.New(errors.InvalidConfigMessage, "Either peers or apps should be set")
	}

	if len(cm.Peers) > 0 {
		for _, config := range cm.Peers {
			if err := config.IsValid(); err != nil {
				return err
			}
		}

	}

	return nil
}

//IsValid validates config messagegetIndexKey
func (pc PeerConfig) IsValid() errors.Error {
	if pc.PeerID == "" {
		return errors.New(errors.InvalidPeerConfig, "PeerID cannot be empty")
	}
	if len(pc.App) == 0 {
		return errors.New(errors.InvalidPeerConfig, "App cannot be empty")
	}
	//App is required
	for _, appConfig := range pc.App {
		if err := appConfig.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

//IsValid appconfig
func (ac AppConfig) IsValid() errors.Error {
	if ac.AppName == "" {
		return errors.New(errors.InvalidAppConfig, "AppName cannot be empty")
	}
	if len(ac.Config) == 0 && len(ac.Components) == 0 {
		return errors.New(errors.InvalidAppConfig, "Neither AppConfig or Components is set (empty payload)")
	}
	if len(ac.Version) == 0 {
		return errors.New(errors.InvalidAppConfig, "AppVersion is not set (empty version)")
	}
	return nil
}

//IsValid ComponentConfig
func (cc ComponentConfig) IsValid() errors.Error {
	if cc.Name == "" {
		return errors.New(errors.InvalidComponentConfig, "Component Name cannot be empty")
	}
	if cc.TxID != "" {
		return errors.New(errors.InvalidComponentConfig, "Tx id should be empty")
	}
	if cc.Config == "" {
		return errors.New(errors.InvalidComponentConfig, "Component config cannot be empty")
	}
	if len(cc.Version) == 0 {
		return errors.New(errors.InvalidComponentConfig, "Component Version is not set (empty version)")
	}
	return nil
}
