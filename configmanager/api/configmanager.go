/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// VERSION of config data
const VERSION = "1"

// ConfigKey contain org,peer,appname,version
type ConfigKey struct {
	MspID         string
	PeerID        string
	AppName       string
	ComponentName string
	Version       string
}

//ConfigKV represents key value struct for managing configurations
type ConfigKV struct {
	Key   ConfigKey
	Value []byte
}

//Components represents app component
type Component struct {
	Name   string
	Config string
}

//AppConfig identifier has application name , config version
type AppConfig struct {
	AppName    string
	Version    string
	Config     string
	Components []Component
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
	Save(config []byte) error
	//Get configuration - Gets configuration based on config key.
	//For the valid config key retuned array will have only one element.
	//For the config key containing only MspID all configurations for that MspID will be returned
	Get(configKey ConfigKey) ([]*ConfigKV, error)
	//Delete configuration -
	//For the valid config one config message will be deleted
	//For the config key containing only MspID all configurations for that MspID will be deleted
	Delete(configKey ConfigKey) error
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
	//Get returns the config bytes for the given channel and config key
	Get(channelID string, configKey ConfigKey) ([]byte, error)
	//GetViper returns a Viper instance that wraps the config for the given channel and config key.
	// If the config key doesn't exist then nil is returned.
	GetViper(channelID string, configKey ConfigKey, configType ConfigType) (*viper.Viper, error)
}

//IsValid validates config message
func (cm ConfigMessage) IsValid() error {
	if cm.MspID == "" {
		return errors.New("MSPID cannot be empty")
	}

	if len(cm.Peers) == 0 && len(cm.Apps) == 0 {
		return errors.New("Either peers or apps should be set")
	}

	if len(cm.Peers) > 0 {
		for _, config := range cm.Peers {
			if err := config.IsValid(); err != nil {
				return err
			}
		}

		//	return errors.New("Collection of peers is required")
	}

	return nil
}

//IsValid validates config messagegetIndexKey
func (pc PeerConfig) IsValid() error {
	if pc.PeerID == "" {
		return errors.New("PeerID cannot be empty")
	}
	if len(pc.App) == 0 {
		return errors.New("App cannot be empty")
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
func (ac AppConfig) IsValid() error {
	if ac.AppName == "" {
		return errors.New("AppName cannot be empty")
	}
	if len(ac.Config) == 0 && len(ac.Components) == 0 {
		return errors.New("Neither AppConfig or Components is set (empty payload)")
	}
	if len(ac.Version) == 0 {
		return errors.New("AppVersion is not set (empty version)")
	}
	return nil
}
