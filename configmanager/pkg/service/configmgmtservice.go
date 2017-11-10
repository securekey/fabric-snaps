/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

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

//AdminServiceImpl admin service implementation
type AdminServiceImpl struct {
}

var instance *ConfigServiceImpl
var initialized uint32

var mu sync.Mutex

const (
	defaultExpirationTime = 300
	purgeExpiredTime      = 600
)

//GetInstance gets instance of cache
func GetInstance() *ConfigServiceImpl {

	if atomic.LoadUint32(&initialized) == 1 {
		return instance
	}

	mu.Lock()
	defer mu.Unlock()

	if initialized == 0 {
		instance = &ConfigServiceImpl{}
		instance.cache = gc.New(defaultExpirationTime, purgeExpiredTime)
		logger.Debugf("Created cache instance")
		atomic.StoreUint32(&initialized, 1)
	}

	return instance
}

//Get items from cache
func (asi *AdminServiceImpl) Get(configKey api.ConfigKey) ([]byte, error) {

	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}
	config, found := GetInstance().cache.Get(keyStr)
	if found {
		if _, ok := config.(string); ok {
			configStr := config.(string)
			return []byte(configStr), nil
		}
	}
	return nil, nil
}

//Refresh expires existing cache and load new one (config snap)
func (asi *AdminServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) error {
	if stub == nil {
		return fmt.Errorf("Stub is nil")
	}
	configManager := mgmt.NewConfigManager(stub)

	//get search criteria
	criteria, err := getSearchCriteria(stub, mspID)
	if err != nil {
		return fmt.Errorf("Cannot create criteria for search by mspID %v", err)
	}
	configMessages, err := configManager.QueryForConfigs(criteria)
	if err != nil {
		return fmt.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}

	if len(*configMessages) == 0 {
		return fmt.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}
	return refreshCache(configMessages, asi)

}

//refreshCache only when value for key was updated or
//when key does not exist in repository
func refreshCache(configMessages *map[string]string, asi *AdminServiceImpl) error {
	for key, value := range *configMessages {
		//before setting verify that content was changed
		configKey, err := mgmt.StringToConfigKey(key)
		if err != nil {
			logger.Debugf("Error: %s", err)
			return err
		}
		valueFromCache, err := asi.Get(configKey)
		if err != nil {
			logger.Debugf("Error in get from cache: %s", err)
			return err
		}
		if err == nil && len(valueFromCache) == 0 {
			fmt.Printf("No value for key: %d %s\n", len(valueFromCache), valueFromCache)
			//cache does not have this combination of key value - save
			GetInstance().cache.Set(key, value, gc.NoExpiration)
		}
		if len(valueFromCache) > 0 && !bytes.Equal(valueFromCache, []byte(value)) {
			logger.Debugf("Refreshing cache for key: %s", key)
			GetInstance().cache.Set(key, value, gc.NoExpiration)
		}
	}
	return nil
}

func getSearchCriteria(stub shim.ChaincodeStubInterface, mspID string) (api.SearchCriteria, error) {

	criteria, err := api.NewSearchCriteriaByMspID(strings.TrimSpace(mspID))
	if err != nil {
		return nil, err
	}
	return criteria, nil
}
