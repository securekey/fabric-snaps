/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"encoding/json"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("configsnap")

const (
	// indexOrg is the name of the index to retrieve configurations per org
	indexMspID = "cfgmgmt-mspid"
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
		return errors.New(errors.GeneralError, "Configuration must be provided")
	}
	//parse configuration request
	configMessageMap, err := ParseConfigMessage(configData, cmngr.stub.GetTxID())
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
			return errors.Errorf(errors.GeneralError, "Cannot put state. Invalid key %s", err)
		}
		if err = cmngr.stub.PutState(strkey, value); err != nil {
			return errors.Wrap(errors.GeneralError, err, "PutState has failed")
		}
		//add index for saved state
		if err := cmngr.addIndexes(key); err != nil {
			return errors.Wrapf(errors.GeneralError, err, "Got error while adding index for %v", key)
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

	if len(configKey.ComponentName) > 0 && len(configKey.ComponentVersion) == 0 {
		values, err := cmngr.getConfigs(configKey)
		if err != nil {
			return nil, err
		}
		filterComp := make([]*api.ConfigKV, 0)
		for _, v := range values {
			if v.Key.ComponentName == configKey.ComponentName && v.Key.AppName == configKey.AppName {
				filterComp = append(filterComp, v)
			}
		}
		return filterComp, nil
	}

	//search for one config by valid key
	config, err := cmngr.getConfig(configKey)
	if err != nil {
		return nil, err
	}
	configKeys := []*api.ConfigKV{&api.ConfigKV{Key: configKey, Value: config}}
	return configKeys, nil
}

//getConfig to get config for valid key
func (cmngr *configManagerImpl) getConfig(configKey api.ConfigKey) ([]byte, error) {
	logger.Debugf("Getting config for %v", configKey)

	key, err := ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}
	//get configuration for valid key
	config, err := cmngr.stub.GetState(key)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "GetState failed")
	}
	if config == nil && len(config) == 0 {
		logger.Debugf("Nothing there for key %s", key)
	}
	return config, nil
}

//getConfigs to get configs for MspId
func (cmngr *configManagerImpl) getConfigs(configKey api.ConfigKey) ([]*api.ConfigKV, error) {
	if configKey.MspID == "" {
		return nil, errors.Errorf(errors.GeneralError, "Invalid config key %v. MspID is required. ", configKey)
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
		return errors.Errorf(errors.GeneralError, "Invalid config key %+v. MspID is required.", configKey)
	}
	configs, err := cmngr.getConfigs(configKey)
	if err != nil {
		return err
	}
	for _, value := range configs {
		logger.Debugf("Deleting state for key: %+v", value.Key)
		keyStr, err := ConfigKeyToString(value.Key)
		if err != nil {
			return err
		}
		if err := cmngr.stub.DelState(keyStr); err != nil {
			return errors.Wrap(errors.GeneralError, err, "DeleteState failed")
		}
	}
	return nil
}

//Delete deletes configuration from the ledger using config key
func (cmngr *configManagerImpl) Delete(configKey api.ConfigKey) error {
	err := ValidateConfigKey(configKey)
	if err != nil {
		//search for all configs by mspID
		return cmngr.deleteConfigs(configKey)
	}

	if len(configKey.ComponentName) > 0 && len(configKey.ComponentVersion) == 0 {
		configs, err := cmngr.getConfigs(configKey)
		if err != nil {
			return err
		}
		for _, value := range configs {
			logger.Debugf("Deleting state for key: %+v", value.Key)
			keyStr, err := ConfigKeyToString(value.Key)
			if err != nil {
				return err
			}
			if value.Key.ComponentName == configKey.ComponentName && value.Key.AppName == configKey.AppName {
				if err := cmngr.stub.DelState(keyStr); err != nil {
					return errors.Wrap(errors.GeneralError, err, "DeleteState failed")
				}
			}
		}
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
func ParseConfigMessage(configData []byte, txID string) (map[api.ConfigKey][]byte, error) {
	configMap := make(map[api.ConfigKey][]byte)
	var parsedConfig api.ConfigMessage
	if err := json.Unmarshal(configData, &parsedConfig); err != nil {
		return nil, errors.Errorf(errors.GeneralError, "Cannot unmarshal config message %s %s", string(configData[:]), err)
	}
	//validate config
	if err := parsedConfig.IsValid(); err != nil {
		return nil, err
	}
	mspID := parsedConfig.MspID
	for _, config := range parsedConfig.Peers {
		for _, appConfig := range config.App {
			key, err := CreateConfigKey(mspID, config.PeerID, appConfig.AppName, appConfig.Version, "", "")
			if err != nil {
				return nil, err
			}
			configMap[key] = []byte(appConfig.Config)
		}
	}
	var key api.ConfigKey
	var err error
	for _, app := range parsedConfig.Apps {
		if len(app.Components) == 0 {
			key, err = CreateConfigKey(mspID, "", app.AppName, app.Version, "", "")
			if err != nil {
				return nil, err
			}
			configMap[key] = []byte(app.Config)
		} else {
			for _, v := range app.Components {
				v.TxID = txID
				key, err = CreateConfigKey(mspID, "", app.AppName, app.Version, v.Name, v.Version)
				if err != nil {
					return nil, err
				}
				bytes, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				configMap[key] = bytes
			}
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
			return errors.Errorf(errors.GeneralError, "error adding index [%s]: %s", index, err)
		}
	}
	return nil
}

//addIndex for configKey
func (cmngr *configManagerImpl) addIndex(index string, configKey api.ConfigKey) error {
	if index == "" {
		return errors.Errorf(errors.GeneralError, "Index is empty")
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
		return "", errors.Errorf(errors.GeneralError, "Index is empty")
	}
	if key == "" {
		return "", errors.Errorf(errors.GeneralError, "Key is empty")
	}
	if len(fields) == 0 {
		return "", errors.Errorf(errors.GeneralError, "Field list is empty")
	}
	attributes := append(fields, key)
	indexKey, err := cmngr.stub.CreateCompositeKey(index, attributes)
	if err != nil {
		return "", errors.Wrapf(errors.GeneralError, err, "Error creating comnposite key: %v", err)
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
		return nil, errors.Errorf(errors.GeneralError, "unknown index [%s]", index)
	}
}

func (cmngr *configManagerImpl) search(key api.ConfigKey) ([]*api.ConfigKV, error) {
	//verify if key has MspID
	if key.MspID == "" {
		return nil, errors.Errorf(errors.GeneralError, "Invalid config key %+v", key)
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
		return nil, errors.Errorf(errors.GeneralError, "Unexpected error retrieving message statuses with index [%s]: %s", index, err)
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
			return nil, errors.Wrapf(errors.GeneralError, err, "Unexpected error splitting composite key. Key: [%s], Error: %s", compositeKey, err)
		}
		configID := compositeKeyParts[len(compositeKeyParts)-1]
		ck, err := StringToConfigKey(configID)
		if err != nil {
			return nil, err
		}
		//get config for key
		config, err := cmngr.getConfig(ck)
		configKV := api.ConfigKV{Key: ck, Value: config}

		if err != nil {
			return nil, err
		}
		configKeys = append(configKeys, &configKV)
	}
	return configKeys, nil
}

//unmarshalConfig unmarshals messages
func unmarshalConfig(configBytes []byte) (string, error) {
	var appConfig string
	if len(configBytes) == 0 {
		return "", errors.Errorf(errors.GeneralError, "No configuration passed to unmarshaller")
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
		return nil, errors.Errorf(errors.GeneralError, "Invalid key %+v", key)
	}
	var fields []string
	fields = append(fields, key.MspID)
	return fields, nil
}
