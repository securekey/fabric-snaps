/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"encoding/json"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configmanager/api"
)

var logger = logging.MustGetLogger("config-manager")

const keyDivider = "_"

// ConfigManagerImpl implements configuration management functionality
type configManagerImpl struct {
	stub shim.ChaincodeStubInterface
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

//NewConfigManager returns config manager implementation
func NewConfigManager(stub shim.ChaincodeStubInterface) api.ConfigManager {
	return &configManagerImpl{stub: stub}
}

// Save saves configuration data in the ledger
func (cmngr *configManagerImpl) Save(configData []byte) error {

	if len(configData) == 0 {
		return errors.New("Configuration must be provided")
	}
	//parse configuration request
	configMessageMap, err := parseConfigMessage(configData)
	if err != nil {
		return err
	}
	for key, value := range configMessageMap {
		strkey := configKeyToString(key)
		if err = cmngr.stub.PutState(strkey, value); err != nil {
			return errors.Errorf("PutState failed, err %s", err)
		}
	}
	return nil
}

// Get gets configuration from the ledger using composite key
func (cmngr *configManagerImpl) Get(configKey api.ConfigKey) ([]byte, error) {
	err := validateConfigKey(configKey)
	if err != nil {
		return nil, err
	}
	key := configKeyToString(configKey)
	//get configuration for valid key
	config, err := cmngr.stub.GetState(key)
	if err != nil {
		return nil, err
	}
	return config, nil
}

//Delete delets configuration from the ledger using composite key
func (cmngr *configManagerImpl) Delete(configKey api.ConfigKey) error {
	if err := validateConfigKey(configKey); err != nil {
		return err
	}
	key := configKeyToString(configKey)
	//delete configuration for valid key
	return cmngr.stub.DelState(key)
}

//ParseConfigMessage unmarshals supplied config message and returns
//map[compositekey]configurationbytes to the caller
func parseConfigMessage(configData []byte) (map[api.ConfigKey][]byte, error) {

	configMap := make(map[api.ConfigKey][]byte)
	var parsedConfig ConfigMessage

	if err := json.Unmarshal(configData, &parsedConfig); err != nil {
		return nil, errors.Errorf("Cannot unmarshal config message %v", err)
	}
	//validate config
	if err := parsedConfig.IsValid(); err != nil {
		return nil, err
	}

	mspID := parsedConfig.MspID
	for _, config := range parsedConfig.Peers {
		for _, appConfig := range config.App {
			key, err := createConfigKey(mspID, config.PeerID, appConfig.AppName)
			if err != nil {
				return nil, err
			}
			configMap[key] = []byte(appConfig.Config)
		}
	}
	return configMap, nil
}

//createCompositeKey creates key using mspID, peerID and appName
func createConfigKey(mspID string, peerID string, appName string) (api.ConfigKey, error) {
	configKey := api.ConfigKey{MspID: mspID, PeerID: peerID, AppName: appName}
	if err := validateConfigKey(configKey); err != nil {
		return configKey, err
	}
	return configKey, nil
}

//validate component parts of ConfigKey
func validateConfigKey(configKey api.ConfigKey) error {
	if len(configKey.MspID) > 0 && len(configKey.PeerID) > 0 && len(configKey.AppName) > 0 {
		return nil
	}
	return errors.Errorf("Cannot create key using mspID: %s, peerID %s, appName %s", configKey.MspID, configKey.PeerID, configKey.AppName)
}

//converts configKey to string
func configKeyToString(configKey api.ConfigKey) string {
	return strings.Join([]string{configKey.MspID, configKey.PeerID, configKey.AppName}, "_")
}

//Format error messages about inproper configuration
func formatError(tag string) error {
	return errors.Errorf("Configuration message does not have proper %s", tag)
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

//IsValid validates config message
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
