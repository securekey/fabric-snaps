/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/configmanager/api"
)

var logger = logging.MustGetLogger("config-manager")

const (
	keyDivider = "!"
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
		return fmt.Errorf("Configuration must be provided")
	}
	//parse configuration request
	configMessageMap, err := parseConfigMessage(configData)
	if err != nil {
		return err
	}
	for key, value := range configMessageMap {
		strkey := configKeyToString(key)
		if err = cmngr.stub.PutState(strkey, value); err != nil {
			return fmt.Errorf("PutState failed, err %s", err)
		}
		//add index for saved state
		if err := cmngr.addIndexes(key); err != nil {
			return fmt.Errorf("Got error while adding index for %v", key)
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
	var parsedConfig api.ConfigMessage

	if err := json.Unmarshal(configData, &parsedConfig); err != nil {
		return nil, fmt.Errorf("Cannot unmarshal config message %v", err)
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

//createConfigKey creates key using mspID, peerID and appName
func createConfigKey(mspID string, peerID string, appName string) (api.ConfigKey, error) {
	configKey := api.ConfigKey{MspID: mspID, PeerID: peerID, AppName: appName}
	if err := validateConfigKey(configKey); err != nil {
		return configKey, err
	}
	return configKey, nil
}

//validateConfigKey validates component parts of ConfigKey
func validateConfigKey(configKey api.ConfigKey) error {
	if len(configKey.MspID) == 0 {
		return fmt.Errorf("Cannot create config key using empty MspId")
	}
	if len(configKey.PeerID) == 0 {
		return fmt.Errorf("Cannot create config key using empty PeerID")
	}
	if len(configKey.AppName) == 0 {
		return fmt.Errorf("Cannot create config key using empty AppName")
	}
	return nil
}

//configKeyToString converts configKey to string
func configKeyToString(configKey api.ConfigKey) string {
	return strings.Join([]string{configKey.MspID, configKey.PeerID, configKey.AppName}, keyDivider)
}

//addIndexes for configKey
func (cmngr *configManagerImpl) addIndexes(key api.ConfigKey) error {
	if err := validateConfigKey(key); err != nil {
		return err
	}
	for _, index := range indexes {
		if err := cmngr.addIndex(index, key); err != nil {
			return fmt.Errorf("error adding index [%s]: %v", index, err)
		}
	}
	return nil
}

//addIndex for configKey
func (cmngr *configManagerImpl) addIndex(index string, configKey api.ConfigKey) error {
	if index == "" {
		return fmt.Errorf("Index is empty")
	}
	if err := validateConfigKey(configKey); err != nil {
		return err
	}
	fields, err := getFieldsForIndex(index, configKey)
	if err != nil {
		return err
	}
	indexKey, err := cmngr.getIndexKey(index, configKeyToString(configKey), fields)
	if err != nil {
		return err
	}
	logger.Debugf("Adding index [%s]\n", indexKey)
	return cmngr.stub.PutState(indexKey, []byte{0x00})
}

//getIndexKey uses CreateCompositeKey to create key using index, key and fields
func (cmngr *configManagerImpl) getIndexKey(index string, key string, fields []string) (string, error) {
	if index == "" {
		return "", fmt.Errorf("Index is empty")
	}
	if key == "" {
		return "", fmt.Errorf("Key is empty")
	}
	if len(fields) == 0 {
		return "", fmt.Errorf("Field list is empty")
	}
	attributes := append(fields, key)
	indexKey, err := cmngr.stub.CreateCompositeKey(index, attributes)
	if err != nil {
		return "", fmt.Errorf("Error creating comnposite key for message status: %v", err)
	}
	return indexKey, nil
}

//getFieldsForIndex returns collection of fields to be indexed
func getFieldsForIndex(index string, key api.ConfigKey) ([]string, error) {
	if err := validateConfigKey(key); err != nil {
		return nil, err
	}
	switch index {
	case indexMspID:
		return []string{key.MspID}, nil
	default:
		return nil, fmt.Errorf("unknown index [%s]", index)
	}
}

// QueryForConfigs gets configs based on criteria
func (cmngr *configManagerImpl) QueryForConfigs(criteria api.SearchCriteria) (*map[string]string, error) {
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

//getConfigurations
func (cmngr *configManagerImpl) getConfigurations(index string, fields []string) (*map[string]string, error) {
	it, err := cmngr.stub.GetStateByPartialCompositeKey(index, fields)
	if err != nil {
		return nil, fmt.Errorf("Unexpected error retrieving message statuses with index [%s]: %v", index, err)
	}
	defer it.Close()
	configsMap := make(map[string]string)
	//var configsForMspID *string
	for it.HasNext() {
		compositeKey, err := it.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeyParts, err := cmngr.stub.SplitCompositeKey(compositeKey.Key)
		if err != nil {
			return nil, fmt.Errorf("Unexpected error splitting composite key. Key: [%s], Error: %v", compositeKey, err)
		}
		ck := api.ConfigKey{}
		configID := compositeKeyParts[len(compositeKeyParts)-1]
		keyParts := strings.Split(configID, keyDivider)
		ck.MspID = keyParts[0]
		ck.PeerID = keyParts[1]
		ck.AppName = keyParts[2]
		//get config for key
		configForMspID, err := cmngr.getConfigForKey(ck)
		if err != nil {
			return nil, err
		}
		if configForMspID == "" {
			return nil, fmt.Errorf("unable to find config [%s] using index [%s]", configForMspID, compositeKey)
		}
		mapKey := configKeyToString(ck)
		configsMap[mapKey] = configForMspID
		//configsForMspID = append(configsForMspID, configForMspID)
	}
	return &configsMap, nil
}

func (cmngr *configManagerImpl) getConfigForKey(key api.ConfigKey) (string, error) {
	configBytes, err := cmngr.Get(key)
	if err != nil {
		return "", fmt.Errorf("error getting config for ID %v: %v", key, err)
	}
	return unmarshalConfig(configBytes)
}

func unmarshalConfig(configBytes []byte) (string, error) {
	var appConfig string
	if len(configBytes) == 0 {
		return "", fmt.Errorf("No configuration passed to unmarshaller")
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
		return "", nil, fmt.Errorf("invalid search criteria: %v", criteria)
	}
}
func getFieldsByMspID(criteria api.SearchCriteria) []string {
	var fields []string
	fields = append(fields, criteria.GetMspID())
	return fields
}
