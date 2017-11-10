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
func (cmngr *configManagerImpl) Get(configKey api.ConfigKey) ([]byte, error) {

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
	return config, nil
}

//Delete delets configuration from the ledger using config key
func (cmngr *configManagerImpl) Delete(configKey api.ConfigKey) error {

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
		return nil, errors.Errorf("Cannot unmarshal config message %v", err)
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

// QueryForConfigs gets configs based on criteria
func (cmngr *configManagerImpl) Query(criteria api.SearchCriteria) (*map[string][]byte, error) {
	logger.Debugf("Query configs with criteria: %s.\n", criteria)

	index, fields, err := getIndexAndFieldsFromCriteria(criteria)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Retrieving config using index [%s] and the following fields [%v].\n", index, fields)

	configsMap, err := cmngr.getConfigurations(index, fields)
	if err != nil {
		return nil, err
	}

	logger.Debugf("Found [%d] configs.\n", len(*configsMap))

	return configsMap, nil
}

//getConfigurations for given index and indexed fields
func (cmngr *configManagerImpl) getConfigurations(index string, fields []string) (*map[string][]byte, error) {
	it, err := cmngr.stub.GetStateByPartialCompositeKey(index, fields)
	if err != nil {
		return nil, errors.Errorf("Unexpected error retrieving message statuses with index [%s]: %v", index, err)
	}
	defer it.Close()
	configsMap := make(map[string][]byte)
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
		configForMspID, err := cmngr.getConfigForKey(ck)
		if err != nil {
			return nil, err
		}
		if len(configForMspID) == 0 {
			return nil, errors.Errorf("unable to find config [%s] using index [%s]", configForMspID, compositeKey)
		}

		configsMap[configID] = configForMspID
	}
	return &configsMap, nil
}

//getConfigForKey get configs for key
func (cmngr *configManagerImpl) getConfigForKey(key api.ConfigKey) ([]byte, error) {
	return cmngr.Get(key)

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

// getIndexAndFieldsFromCriteria examines the search criteria and determines the appropriate
// index and fields to use for the query
func getIndexAndFieldsFromCriteria(criteria api.SearchCriteria) (index string, fields []string, err error) {
	switch criteria.GetSearchType() {
	case api.SearchByMspID:
		return indexMspID, getFieldsByMspID(criteria), nil
	default:
		return "", nil, errors.Errorf("invalid search criteria: %v", criteria)
	}
}

//getFieldsByMspID returns fields defined for search criteria
func getFieldsByMspID(criteria api.SearchCriteria) []string {
	var fields []string
	fields = append(fields, criteria.GetMspID())
	return fields
}
