/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// ConfigKey contain org,peer,appname
type ConfigKey struct {
	MspID   string
	PeerID  string
	AppName string
}

//ConfigKV represents key value struct for managing configurations
type ConfigKV struct {
	Key   ConfigKey
	Value []byte
}

//AppConfig identifier has application name and config
type AppConfig struct {
	AppName string
	Config  json.RawMessage
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

//ConfigService configuration service interface
type ConfigService interface {
	Get(channelID string, configKey ConfigKey) ([]byte, error)
}

//IsValid validates config message
func (cm ConfigMessage) IsValid() error {
	if cm.MspID == "" {
		return errors.New("MSPID cannot be empty")
	}
	if len(cm.Peers) == 0 {
		return errors.New("Collection of peers is required")
	}

	for _, config := range cm.Peers {
		if err := config.IsValid(); err != nil {
			return err
		}
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
	if len(ac.Config) == 0 {
		return errors.New("AppConfig is not set (empty payload)")
	}
	return nil
}
