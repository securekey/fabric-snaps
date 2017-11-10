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
	//Save configuration
	Save(jsonConfig []byte) error
	//Get configuration
	Get(configKey ConfigKey) (appconfig []byte, err error)
	//Delete configuration
	Delete(configKey ConfigKey) error
	//Query for configs based on supplied critria.
	//Returned map's key is string representation og configKey and value is config for that key
	QueryForConfigs(criteria SearchCriteria) (*map[string]string, error)
}

//ConfigService configuration service interface
type ConfigService interface {
	Get(configKey ConfigKey) ([]byte, error)
}

//ConfigServiceAdmin admin interface for configuration service
type ConfigServiceAdmin interface {
	ConfigService
	//To refresh items in cache
	Refresh(stub shim.ChaincodeStubInterface, mspID string) error
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
