/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configmanager/api"
)

var logger = logging.MustGetLogger("config-manager")

const (
	// indexOrg is the name of the index to retrieve configurations per org
	indexMspID = "cfgmgmt-mspid"
	// configData is the name of the data collection for config maanger
	configData = "config-mngmt"
)

// indexes contains a list of indexes that should be added for configurations
var indexes = [...]string{indexMspID}

// ConfigManagerImpl implements configuration management functionality
type configManagerImpl struct {
	stub shim.ChaincodeStubInterface
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
	return cmngr.saveConfigs(configMessageMap)
}

//saveConfigs saves key&configs to the repository.
//also it adds indexes for saved records
func (cmngr *configManagerImpl) saveConfigs(configMessageMap map[api.ConfigKey][]byte) error {
	for key, value := range configMessageMap {
		logger.Debugf("Saving configs %v,%s", key, string(value[:]))
		strkey, err := ConfigKeyToString(key)
		if err != nil {
			return errors.Errorf("Cannot put state. Invalid key %s", err)
		}
		if err = cmngr.stub.PutState(strkey, value); err != nil {
			return errors.Errorf("PutState failed, err %s", err)
		}
		//add index for saved state
		if err := cmngr.addIndexes(key); err != nil {
			return errors.Errorf("Got error while adding index for %v", key)
		}
	}
	return nil
}

// Get gets configuration from the ledger using config key
func (cmngr *configManagerImpl) Get(configKey api.ConfigKey) ([]*api.ConfigKV, error) {
	err := ValidateConfigKey(configKey)
	if err != nil {
		//search for all configs by mspID
		return cmngr.getConfigs(configKey)
	}
	//search for one config by valid key
	return cmngr.getConfig(configKey)
}

//getConfig to get config for valid key
func (cmngr *configManagerImpl) getConfig(configKey api.ConfigKey) ([]*api.ConfigKV, error) {
	logger.Debugf("Getting config for %v", configKey)

	key, err := ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}
	//get configuration for valid key
	config, err := cmngr.stub.GetState(key)
	if err != nil {
		return nil, err
	}
	if config == nil && err == nil {
		logger.Debugf("Nothing there for key %s", key)
	}
	configKeys := []*api.ConfigKV{&api.ConfigKV{Key: configKey, Value: config}}
	return configKeys, nil
}

//getConfigs to get configs for MspId
func (cmngr *configManagerImpl) getConfigs(configKey api.ConfigKey) ([]*api.ConfigKV, error) {
	if configKey.MspID == "" {
		return nil, errors.Errorf("Invalid config key %v. MspID is required. ", configKey)
	}
	logger.Debugf("Getting configs for %v", configKey)

	configs, err := cmngr.search(configKey)
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (cmngr *configManagerImpl) deleteConfigs(configKey api.ConfigKey) error {
	if configKey.MspID == "" {
		return errors.Errorf("Invalid config key %v. MspID is required. ", configKey)
	}
	configs, err := cmngr.getConfigs(configKey)
	if err != nil {
		return err
	}
	for _, value := range configs {
		logger.Debugf("Deleting state for key: %v", value.Key)
		keyStr, err := ConfigKeyToString(value.Key)
		if err != nil {
			return err
		}
		if err := cmngr.stub.DelState(keyStr); err != nil {
			return err
		}
	}
	return nil
}

//Delete delets configuration from the ledger using config key
func (cmngr *configManagerImpl) Delete(configKey api.ConfigKey) error {
	err := ValidateConfigKey(configKey)
	if err != nil {
		//search for all configs by mspID
		return cmngr.deleteConfigs(configKey)
	}
	key, err := ConfigKeyToString(configKey)
	if err != nil {
		return err
	}
	//delete configuration for valid key
	return cmngr.stub.DelState(key)
}

//ParseConfigMessage unmarshals supplied config message and returns
//map[compositekey]configurationbytes to the caller
func parseConfigMessage(configData []byte) (map[api.ConfigKey][]byte, error) {

	configMap := make(map[api.ConfigKey][]byte)
	var parsedConfig api.ConfigMessage

	if err := json.Unmarshal(configData, &parsedConfig); err != nil {
		return nil, errors.Errorf("Cannot unmarshal config message %v %v", string(configData[:]), err)
	}
	//validate config
	if err := parsedConfig.IsValid(); err != nil {
		return nil, err
	}

	mspID := parsedConfig.MspID
	for _, config := range parsedConfig.Peers {
		for _, appConfig := range config.App {
			key, err := CreateConfigKey(mspID, config.PeerID, appConfig.AppName)
			if err != nil {
				return nil, err
			}
			configMap[key] = []byte(appConfig.Config)
		}
	}
	return configMap, nil
}

//addIndexes for configKey
func (cmngr *configManagerImpl) addIndexes(key api.ConfigKey) error {
	if err := ValidateConfigKey(key); err != nil {
		return err
	}
	for _, index := range indexes {
		if err := cmngr.addIndex(index, key); err != nil {
			return errors.Errorf("error adding index [%s]: %v", index, err)
		}
	}
	return nil
}

//addIndex for configKey
func (cmngr *configManagerImpl) addIndex(index string, configKey api.ConfigKey) error {
	if index == "" {
		return errors.Errorf("Index is empty")
	}
	if err := ValidateConfigKey(configKey); err != nil {
		return err
	}
	fields, err := getFieldsForIndex(index, configKey)
	if err != nil {
		return err
	}
	strKey, err := ConfigKeyToString(configKey)
	if err != nil {
		return err
	}
	indexKey, err := cmngr.getIndexKey(index, strKey, fields)
	if err != nil {
		return err
	}
	logger.Debugf("Adding index [%s]\n", indexKey)
	return cmngr.stub.PutState(indexKey, []byte{0x00})
}

//getIndexKey uses CreateCompositeKey to create key using index, key and fields
func (cmngr *configManagerImpl) getIndexKey(index string, key string, fields []string) (string, error) {
	if index == "" {
		return "", errors.New("Index is empty")
	}
	if key == "" {
		return "", errors.New("Key is empty")
	}
	if len(fields) == 0 {
		return "", errors.New("Field list is empty")
	}
	attributes := append(fields, key)
	indexKey, err := cmngr.stub.CreateCompositeKey(index, attributes)
	if err != nil {
		return "", errors.Errorf("Error creating comnposite key: %v", err)
	}
	return indexKey, nil
}

//getFieldsForIndex returns collection of fields to be indexed
func getFieldsForIndex(index string, key api.ConfigKey) ([]string, error) {
	if err := ValidateConfigKey(key); err != nil {
		return nil, err
	}
	switch index {
	case indexMspID:
		return []string{key.MspID}, nil
	default:
		return nil, errors.Errorf("unknown index [%s]", index)
	}
}

func (cmngr *configManagerImpl) search(key api.ConfigKey) ([]*api.ConfigKV, error) {
	//verify if key has MspID
	if key.MspID == "" {
		return nil, errors.Errorf("Invalid config key %v", key)
	}
	index, fields, err := getIndexAndFields(key)
	configsMap, err := cmngr.getConfigurations(index, fields)
	if err != nil {
		return nil, err
	}

	return configsMap, nil
}

//getConfigurations for given index and indexed fields
func (cmngr *configManagerImpl) getConfigurations(index string, fields []string) ([]*api.ConfigKV, error) {
	it, err := cmngr.stub.GetStateByPartialCompositeKey(index, fields)
	if err != nil {
		return nil, errors.Errorf("Unexpected error retrieving message statuses with index [%s]: %v", index, err)
	}
	defer it.Close()
	configKeys := []*api.ConfigKV{}
	for it.HasNext() {
		compositeKey, err := it.Next()
		if err != nil {
			return nil, err
		}
		_, compositeKeyParts, err := cmngr.stub.SplitCompositeKey(compositeKey.Key)
		if err != nil {
			return nil, errors.Errorf("Unexpected error splitting composite key. Key: [%s], Error: %v", compositeKey, err)
		}
		configID := compositeKeyParts[len(compositeKeyParts)-1]
		ck, err := StringToConfigKey(configID)
		if err != nil {
			return nil, err
		}
		//get config for key
		configForMspID, err := cmngr.getConfig(ck)
		if err != nil {
			return nil, err
		}
		configKeys = append(configKeys, configForMspID[0])
	}
	return configKeys, nil
}

//unmarshalConfig unmarshals messages
func unmarshalConfig(configBytes []byte) (string, error) {
	var appConfig string
	if len(configBytes) == 0 {
		return "", errors.Errorf("No configuration passed to unmarshaller")
	}
	err := json.Unmarshal(configBytes, &appConfig)
	return appConfig, err
}

//getIndexAndFields index and fields for search
func getIndexAndFields(key api.ConfigKey) (string, []string, error) {
	fields, err := getIndexedFields(key)
	if err != nil {
		return "", nil, err
	}
	return indexMspID, fields, nil

}

//getIndexedFields returns fields defined for search criteria
func getIndexedFields(key api.ConfigKey) ([]string, error) {
	if key.MspID == "" {
		return nil, errors.Errorf("Invalid key %v", key)
	}
	var fields []string
	fields = append(fields, key.MspID)
	return fields, nil
}
