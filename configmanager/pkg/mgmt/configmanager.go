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
func (cmngr *configManagerImpl) Save(configData []byte) errors.Error {

	if len(configData) == 0 {
		return errors.New(errors.MissingRequiredParameterError, "Configuration must be provided")
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
func (cmngr *configManagerImpl) saveConfigs(configMessageMap map[api.ConfigKey][]byte) errors.Error {
	for key, value := range configMessageMap {
		logger.Debugf("Saving configs %v,%s", key, string(value[:]))
		strkey, err := ConfigKeyToString(key)
		if err != nil {
			return err
		}
		if e := cmngr.stub.PutState(strkey, value); e != nil {
			return errors.Wrap(errors.SystemError, e, "PutState has failed")
		}
		//add index for saved state
		if err := cmngr.addIndexes(key); err != nil {
			return err
		}
	}
	return nil
}

// Get gets configuration from the ledger using config key
func (cmngr *configManagerImpl) Get(configKey api.ConfigKey) ([]*api.ConfigKV, errors.Error) {
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
	configKeys := []*api.ConfigKV{{Key: configKey, Value: config}}
	return configKeys, nil
}

//getConfig to get config for valid key
func (cmngr *configManagerImpl) getConfig(configKey api.ConfigKey) ([]byte, errors.Error) {
	logger.Debugf("Getting config for %v", configKey)

	key, codedErr := ConfigKeyToString(configKey)
	if codedErr != nil {
		return nil, codedErr
	}
	//get configuration for valid key
	config, err := cmngr.stub.GetState(key)
	if err != nil {
		return nil, errors.Wrap(errors.SystemError, err, "GetState failed")
	}
	if config == nil && len(config) == 0 {
		logger.Debugf("Nothing there for key %s", key)
	}
	return config, nil
}

//getConfigs to get configs for MspId
func (cmngr *configManagerImpl) getConfigs(configKey api.ConfigKey) ([]*api.ConfigKV, errors.Error) {
	if configKey.MspID == "" {
		return nil, errors.Errorf(errors.InvalidConfigKey, "Invalid config key %v. MspID is required. ", configKey)
	}
	logger.Debugf("Getting configs for %v", configKey)

	configs, err := cmngr.search(configKey)
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (cmngr *configManagerImpl) deleteConfigs(configKey api.ConfigKey) errors.Error {
	if configKey.MspID == "" {
		return errors.Errorf(errors.InvalidConfigKey, "Invalid config key %+v. MspID is required.", configKey)
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
			return errors.Wrap(errors.SystemError, err, "DeleteState failed")
		}
	}
	return nil
}

//Delete deletes configuration from the ledger using config key
func (cmngr *configManagerImpl) Delete(configKey api.ConfigKey) errors.Error {
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
					return errors.Wrap(errors.SystemError, err, "DeleteState failed")
				}
			}
		}
	}

	key, err := ConfigKeyToString(configKey)
	if err != nil {
		return err
	}
	//delete configuration for valid key
	e := cmngr.stub.DelState(key)
	if e != nil {
		return errors.Wrap(errors.SystemError, e, "DelState failed")
	}

	return nil
}

//ParseConfigMessage unmarshals supplied config message and returns
//map[compositekey]configurationbytes to the caller
func ParseConfigMessage(configData []byte, txID string) (map[api.ConfigKey][]byte, errors.Error) {
	configMap := make(map[api.ConfigKey][]byte)
	var parsedConfig api.ConfigMessage
	if err := json.Unmarshal(configData, &parsedConfig); err != nil {
		return nil, errors.Errorf(errors.UnmarshalError, "Cannot unmarshal config message %s %s", string(configData[:]), err)
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
	var err errors.Error
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
				bytes, e := json.Marshal(v)
				if e != nil {
					return nil, errors.WithMessage(errors.SystemError, e, "Failed to marshal app component")
				}
				configMap[key] = bytes
			}
		}
	}
	return configMap, nil
}

//addIndexes for configKey
func (cmngr *configManagerImpl) addIndexes(key api.ConfigKey) errors.Error {
	if err := ValidateConfigKey(key); err != nil {
		return err
	}
	for _, index := range indexes {
		if err := cmngr.addIndex(index, key); err != nil {
			return err
		}
	}
	return nil
}

//addIndex for configKey
func (cmngr *configManagerImpl) addIndex(index string, configKey api.ConfigKey) errors.Error {
	if index == "" {
		return errors.Errorf(errors.SystemError, "Index is empty")
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
	e := cmngr.stub.PutState(indexKey, []byte{0x00})
	if e != nil {
		return errors.WithMessage(errors.SystemError, e, "Failed to create index")
	}

	return nil
}

//getIndexKey uses CreateCompositeKey to create key using index, key and fields
func (cmngr *configManagerImpl) getIndexKey(index string, key string, fields []string) (string, errors.Error) {
	if index == "" {
		return "", errors.New(errors.MissingRequiredParameterError, "Index is empty")
	}
	if key == "" {
		return "", errors.New(errors.MissingRequiredParameterError, "Key is empty")
	}
	if len(fields) == 0 {
		return "", errors.New(errors.MissingRequiredParameterError, "Field list is empty")
	}
	attributes := append(fields, key)
	indexKey, err := cmngr.stub.CreateCompositeKey(index, attributes)
	if err != nil {
		return "", errors.Wrapf(errors.SystemError, err, "Error creating comnposite key: %v", err)
	}
	return indexKey, nil
}

//getFieldsForIndex returns collection of fields to be indexed
func getFieldsForIndex(index string, key api.ConfigKey) ([]string, errors.Error) {
	if err := ValidateConfigKey(key); err != nil {
		return nil, err
	}
	switch index {
	case indexMspID:
		return []string{key.MspID}, nil
	default:
		return nil, errors.Errorf(errors.SystemError, "unknown index [%s]", index)
	}
}

func (cmngr *configManagerImpl) search(key api.ConfigKey) ([]*api.ConfigKV, errors.Error) {
	//verify if key has MspID
	if key.MspID == "" {
		return nil, errors.Errorf(errors.InvalidConfigKey, "Invalid config key %+v", key)
	}
	index, fields, err := getIndexAndFields(key)
	configsMap, err := cmngr.getConfigurations(index, fields)
	if err != nil {
		return nil, err
	}

	return configsMap, nil
}

//getConfigurations for given index and indexed fields
func (cmngr *configManagerImpl) getConfigurations(index string, fields []string) ([]*api.ConfigKV, errors.Error) {
	it, err := cmngr.stub.GetStateByPartialCompositeKey(index, fields)
	if err != nil {
		return nil, errors.Errorf(errors.SystemError, "Unexpected error retrieving message statuses with index [%s]: %s", index, err)
	}
	defer it.Close()
	configKeys := []*api.ConfigKV{}
	for it.HasNext() {
		compositeKey, e := it.Next()
		if e != nil {
			return nil, errors.WithMessage(errors.SystemError, e, "Failed to get next value from iterator")
		}
		_, compositeKeyParts, e := cmngr.stub.SplitCompositeKey(compositeKey.Key)
		if e != nil {
			return nil, errors.Wrapf(errors.SystemError, err, "Unexpected error splitting composite key. Key: [%s], Error: %s", compositeKey, err)
		}
		configID := compositeKeyParts[len(compositeKeyParts)-1]
		ck, err := StringToConfigKey(configID)
		if err != nil {
			return nil, err
		}
		//get config for key
		config, err := cmngr.getConfig(ck)
		if err != nil {
			return nil, err
		}

		configKV := api.ConfigKV{Key: ck, Value: config}
		configKeys = append(configKeys, &configKV)
	}
	return configKeys, nil
}

//getIndexAndFields index and fields for search
func getIndexAndFields(key api.ConfigKey) (string, []string, errors.Error) {
	fields, err := getIndexedFields(key)
	if err != nil {
		return "", nil, err
	}
	return indexMspID, fields, nil

}

//getIndexedFields returns fields defined for search criteria
func getIndexedFields(key api.ConfigKey) ([]string, errors.Error) {
	if key.MspID == "" {
		return nil, errors.Errorf(errors.InvalidConfigKey, "Invalid key %+v", key)
	}
	var fields []string
	fields = append(fields, key.MspID)
	return fields, nil
}
