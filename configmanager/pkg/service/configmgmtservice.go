/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"fmt"

	"sync"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	logging "github.com/op/go-logging"
	gc "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
)

var logger = logging.MustGetLogger("configmngmt-service")

//ConfigServiceImpl used to create cache instance
type ConfigServiceImpl struct {
	mtx      sync.RWMutex
	cacheMap map[string]*gc.Cache
}

const (
	defaultExpirationTime = 300
	purgeExpiredTime      = 600
)

var instance *ConfigServiceImpl
var once sync.Once

//GetInstance gets instance of cache for snaps
func GetInstance() api.ConfigService {
	return instance
}

//Initialize will be called from config snap
func Initialize(stub shim.ChaincodeStubInterface, mspID string) *ConfigServiceImpl {

	once.Do(func() {
		instance = &ConfigServiceImpl{}
		instance.cacheMap = make(map[string]*gc.Cache)
		logger.Infof("Created cache instance %v", time.Unix(time.Now().Unix(), 0))
	})
	instance.createCache(stub.GetChannelID())
	instance.Refresh(stub, mspID)
	return instance
}

//Get items from cache
func (csi *ConfigServiceImpl) Get(channelID string, configKey api.ConfigKey) ([]byte, error) {
	if len(csi.cacheMap) == 0 {
		return nil, errors.New("Cache was not initialized")
	}
	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}
	channelCache := csi.getCache(channelID)
	if channelCache == nil {
		return nil, errors.New("No cache exists for '%s' channelID")
	}
	//find item in cache
	config, found := channelCache.Get(keyStr)
	if found {
		v, ok := config.([]byte)
		if ok {
			return v, nil
		}
		//cannot serialize config context
		logger.Debugf("Error getting config from cache. %v", config)
		return nil, errors.Errorf("Error getting config from cache. %v", config)
	}
	return nil, nil
}

//Refresh adds new items into cache and refreshes existing ones only if value for key was changed
func (csi *ConfigServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) (bool, error) {
	fmt.Printf("***Refresh %v\n", time.Unix(time.Now().Unix(), 0))
	if len(csi.cacheMap) == 0 {
		return false, errors.New("Cache was not initialized")
	}
	if stub == nil {
		return false, errors.New("Stub is nil")
	}

	configManager := mgmt.NewConfigManager(stub)
	//get all by mspID
	configKey := api.ConfigKey{MspID: mspID}
	configMessages, err := configManager.Get(configKey)
	if err != nil {
		return false, errors.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}

	if len(configMessages) == 0 {
		return false, errors.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}

	return csi.refreshCache(stub.GetChannelID(), configMessages)

}

//refreshCache only when value for key was updated or when key does not exist in repository
func (csi *ConfigServiceImpl) refreshCache(channelID string, configMessages []*api.ConfigKV) (bool, error) {
	if len(csi.cacheMap) == 0 {
		return false, errors.New("Cache was not initialized")
	}
	var cacheChanged = false
	for key, val := range configMessages {
		//get item from cache based on channel and configKey
		cachedConfig, err := csi.Get(channelID, val.Key)
		if err != nil {
			logger.Debugf("Error in get from cache: %s", err)
			return false, err
		}
		keyStr, err := mgmt.ConfigKeyToString(val.Key)
		if err != nil {
			return false, err
		}
		if len(cachedConfig) == 0 {
			//cache does not have this config - add it
			logger.Debugf("Adding cache for channel: %s", channelID)
			csi.put(channelID, keyStr, val.Value)
			cacheChanged = true
		}
		if !bytes.Equal(cachedConfig, val.Value) {
			//update only in case when config value is anew
			logger.Debugf("Refreshing cache for key: %s", key)
			csi.put(channelID, keyStr, val.Value)
			cacheChanged = true
		}
	}
	return cacheChanged, nil
}

//to add new config to cache or to update existing one
func (csi *ConfigServiceImpl) getCache(channelID string) *gc.Cache {
	csi.mtx.RLock()
	defer csi.mtx.RUnlock()
	return csi.cacheMap[channelID]
}

func (csi *ConfigServiceImpl) createCache(channelID string) {
	csi.mtx.Lock()
	defer csi.mtx.Unlock()
	logger.Debugf("Created cache for channel %s", channelID)
	instance.cacheMap[channelID] = gc.New(defaultExpirationTime, purgeExpiredTime)
}

func (csi *ConfigServiceImpl) put(channelID string, key string, value []byte) {
	cache := csi.getCache(channelID)
	logger.Debugf("Putting in cache: %s %s %s", channelID, key, string(value[:]))
	cache.Set(key, value, gc.NoExpiration)
}
