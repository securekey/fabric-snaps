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

	"github.com/docker/docker/pkg/stringutils"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	errors "github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("configsnap")

type cache map[string][]byte

//ConfigServiceImpl used to create cache instance
type ConfigServiceImpl struct {
	mtx      sync.RWMutex
	cacheMap map[string]cache
}

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
		instance.cacheMap = make(map[string]cache)
		logger.Infof("Created cache instance %v", time.Unix(time.Now().Unix(), 0))
	})
	instance.Refresh(stub, mspID)
	return instance
}

//Get items from cache
func (csi *ConfigServiceImpl) Get(channelID string, configKey api.ConfigKey) ([]byte, error) {
	if csi == nil {
		return nil, errors.New(errors.GeneralError, "ConfigServiceImpl was not initialized")
	}

	channelCache := csi.getCache(channelID)
	if channelCache == nil {
		logger.Debugf("Config cache is not initialized for channel [%s]. Getting config from ledger.\n", channelID)
		return csi.GetConfigFromLedger(channelID, configKey)
	}

	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}

	val := channelCache[keyStr]
	if len(val) == 0 {
		logger.Debugf("Config cache does not contain config for key [%s] on channel [%s]. Getting config from ledger.\n", keyStr, channelID)
		//not in cache get from ledger
		return csi.GetConfigFromLedger(channelID, configKey)
	}
	return channelCache[keyStr], nil
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
	err = v.ReadConfig(bytes.NewBuffer(configData))
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "snap_config_init_error")
	}
	return v, err
}

//Refresh adds new items into cache and refreshes existing ones
func (csi *ConfigServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) error {
	logger.Debugf("***Refreshing mspid %s at %v\n", mspID, time.Unix(time.Now().Unix(), 0))
	if csi == nil {
		return errors.New(errors.GeneralError, "ConfigServiceImpl was not initialized")
	}
	if stub == nil {
		return errors.New(errors.GeneralError, "Stub is nil")
	}

	configManager := mgmt.NewConfigManager(stub)
	//get all by mspID
	configKey := api.ConfigKey{MspID: mspID}
	configMessages, err := configManager.Get(configKey)
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Cannot create criteria for search by mspID %v", configMessages)
	}

	if len(configMessages) == 0 {
		return errors.Errorf(errors.GeneralError, "Cannot create criteria for search by mspID %v", configMessages)
	}

	return csi.refreshCache(stub.GetChannelID(), configMessages)
}

//GetConfigFromLedger - gets snaps configs from ledger
func (csi *ConfigServiceImpl) GetConfigFromLedger(channelID string, configKey api.ConfigKey) ([]byte, error) {

	lgr := peer.GetLedger(channelID)

	if lgr != nil {
		logger.Debugf("****Ledger is set for channelID %s\n", channelID)
		r := stringutils.GenerateRandomAlphaOnlyString(12)
		txsim, err := lgr.NewTxSimulator(r)
		if err != nil {
			logger.Errorf("Cannot create transaction simulator %v", err)
			return nil, errors.WithMessage(errors.GeneralError, err, "Cannot create transaction simulator")
		}
		defer txsim.Done()

		keyStr, err := mgmt.ConfigKeyToString(configKey)
		config, err := txsim.GetState("configurationsnap", keyStr)
		if err != nil {
			logger.Errorf("Error getting state for app %s %v", keyStr, err)
			return nil, errors.Wrap(errors.GeneralError, err, "Error getting state")
		}
		return config, nil
	}

	return nil, errors.Errorf(errors.GeneralError, "Cannot obtain ledger for channel %s", channelID)
}

func (csi *ConfigServiceImpl) refreshCache(channelID string, configMessages []*api.ConfigKV) error {
	if csi == nil {
		return errors.New(errors.GeneralError, "ConfigServiceImpl was not initialized")
	}

	logger.Debugf("Updating cache for channel %s\n", channelID)

	cache := make(map[string][]byte)

	for _, val := range configMessages {
		keyStr, err := mgmt.ConfigKeyToString(val.Key)
		if err != nil {
			return err
		}
		logger.Debugf("Adding item for key [%s] and channel [%s] to cache\n", keyStr, channelID)
		cache[keyStr] = val.Value
	}
	csi.mtx.Lock()
	defer csi.mtx.Unlock()
	instance.cacheMap[channelID] = cache

	logger.Debugf("Updated cache for channel %s\n", channelID)
	return nil
}

func (csi *ConfigServiceImpl) getCache(channelID string) cache {
	csi.mtx.RLock()
	defer csi.mtx.RUnlock()
	return csi.cacheMap[channelID]
}
