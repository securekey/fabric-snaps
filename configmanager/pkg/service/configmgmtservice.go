/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"

	"github.com/spf13/viper"

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
	logger.Debugf("ConfigManagerService was intialized")
	once.Do(func() {
		instance = &ConfigServiceImpl{}
		instance.cacheMap = make(map[string]*gc.Cache)
		logger.Infof("Created cache instance %v", time.Unix(time.Now().Unix(), 0))
	})
	instance.Refresh(stub, mspID)
	return instance
}

//Get items from cache
func (csi *ConfigServiceImpl) Get(channelID string, configKey api.ConfigKey) ([]byte, error) {
	if csi == nil {
		return nil, errors.New("ConfigServiceImpl was not initialized")
	}

	channelCache := csi.getCache(channelID)
	if channelCache == nil {
		return nil, nil
	}

	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
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

//GetViper configuration as Viper
func (csi *ConfigServiceImpl) GetViper(channelID string, configKey api.ConfigKey, configType api.ConfigType) (*viper.Viper, error) {
	configData, err := csi.Get(channelID, configKey)
	if err != nil {
		return nil, err
	}
	if len(configData) == 0 {
		// No config found for the key. Return nil instead of an error so that the caller can differentiate between the two cases
		return nil, nil
	}

	v := viper.New()
	v.SetConfigType(string(configType))
	v.ReadConfig(bytes.NewBuffer(configData))

	return v, err
}

//Refresh adds new items into cache and refreshes existing ones
func (csi *ConfigServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) error {
	logger.Debugf("***Refreshing %v\n", time.Unix(time.Now().Unix(), 0))
	if csi == nil {
		return errors.New("ConfigServiceImpl was not initialized")
	}
	if stub == nil {
		return errors.New("Stub is nil")
	}

	configManager := mgmt.NewConfigManager(stub)
	//get all by mspID
	configKey := api.ConfigKey{MspID: mspID}
	configMessages, err := configManager.Get(configKey)
	if err != nil {
		return errors.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}

	if len(configMessages) == 0 {
		return errors.Errorf("Cannot create criteria for search by mspID %v", configMessages)
	}

	return csi.refreshCache(stub.GetChannelID(), configMessages)
}

func (csi *ConfigServiceImpl) refreshCache(channelID string, configMessages []*api.ConfigKV) error {
	if csi == nil {
		return errors.New("ConfigServiceImpl was not initialized")
	}

	cache := gc.New(defaultExpirationTime, purgeExpiredTime)

	for _, val := range configMessages {
		keyStr, err := mgmt.ConfigKeyToString(val.Key)
		if err != nil {
			return err
		}
		logger.Debugf("Adding [%s]=[%s] for channel [%s]\n", keyStr, val.Value, channelID)
		cache.Set(keyStr, val.Value, gc.NoExpiration)
	}

	logger.Debugf("Updating cache for channel %s", channelID)

	csi.mtx.Lock()
	defer csi.mtx.Unlock()
	instance.cacheMap[channelID] = cache

	return nil
}

func (csi *ConfigServiceImpl) getCache(channelID string) *gc.Cache {
	csi.mtx.RLock()
	defer csi.mtx.RUnlock()
	return csi.cacheMap[channelID]
}
