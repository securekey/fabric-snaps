/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	gc "github.com/patrickmn/go-cache"
	"github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
)

var logger = logging.MustGetLogger("configmngmt-service")

//ConfigServiceImpl used to create cache instance
type ConfigServiceImpl struct {
	cache *gc.Cache
}

const (
	defaultExpirationTime = 300
	purgeExpiredTime      = 600
)

var instance *ConfigServiceImpl

//GetInstance gets instance of cache for snaps
func GetInstance() api.ConfigService {
	return instance
}

//Initialize will be called from config snap
func Initialize(stub shim.ChaincodeStubInterface, mspID string) *ConfigServiceImpl {
	if instance != nil {
		logger.Warningf("Cache instance was alreday initialized")
		return instance
	}
	instance = &ConfigServiceImpl{}
	instance.cache = gc.New(defaultExpirationTime, purgeExpiredTime)
	logger.Infof("****Created cache instance %v", time.Unix(time.Now().Unix(), 0))
	return instance
}

//Get items from cache
func (csi *ConfigServiceImpl) Get(configKey api.ConfigKey) ([]byte, error) {
	if csi.cache == nil {
		return nil, fmt.Errorf("Cache was not initialized")
	}

	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}
	config, found := csi.cache.Get(keyStr)
	if found {
		if _, ok := config.(string); ok {
			configStr := config.(string)
			return []byte(configStr), nil
		}
	}
	return nil, nil
}

//Refresh adds new items into cache and refreshes existing ones only if value for key was changed
func (csi *ConfigServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) (bool, error) {
	if csi.cache == nil {
		return false, fmt.Errorf("Cache was not initialized")
	}
	if stub == nil {
		return false, fmt.Errorf("Stub is nil")
	}
	configManager := mgmt.NewConfigManager(stub)

	//get search criteria
	criteria, err := getSearchCriteria(stub, mspID)
	if err != nil {
		return false, fmt.Errorf("Cannot create criteria for search by mspID %v", err)
	}
	configMessages, err := configManager.QueryForConfigs(criteria)
	if err != nil {
		return false, fmt.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}

	if len(*configMessages) == 0 {
		return false, fmt.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}
	return csi.refreshCache(configMessages)

}

//refreshCache only when value for key was updated or when key does not exist in repository
func (csi *ConfigServiceImpl) refreshCache(configMessages *map[string]string) (bool, error) {
	if csi.cache == nil {
		return false, fmt.Errorf("Cache was not initialized")
	}
	cacheUpdated := false
	for key, value := range *configMessages {
		//before setting verify that content was changed
		configKey, err := mgmt.StringToConfigKey(key)
		if err != nil {
			logger.Debugf("Error: %s", err)
			return false, err
		}
		valueFromCache, err := csi.Get(configKey)
		if err != nil {
			logger.Debugf("Error in get from cache: %s", err)
			return false, err
		}
		if len(valueFromCache) == 0 {
			//cache does not have this combination of key value -do set
			cacheUpdated = true
			csi.cache.Set(key, value, gc.NoExpiration)
		}
		if len(valueFromCache) > 0 && !bytes.Equal(valueFromCache, []byte(value)) {
			logger.Debugf("Refreshing cache for key: %s", key)
			csi.cache.Set(key, value, gc.NoExpiration)
			cacheUpdated = true
		}
	}
	return cacheUpdated, nil
}

func getSearchCriteria(stub shim.ChaincodeStubInterface, mspID string) (api.SearchCriteria, error) {

	criteria, err := api.NewSearchCriteriaByMspID(strings.TrimSpace(mspID))
	if err != nil {
		return nil, err
	}
	return criteria, nil
}
