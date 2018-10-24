/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"fmt"

	"github.com/spf13/viper"

	"sync"
	"time"

	"math/rand"

	"encoding/json"

	"crypto/sha256"
	"encoding/base64"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	"github.com/securekey/fabric-snaps/metrics/cmd/filter/metrics"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("configsnap")

type cache map[string][]byte

//ConfigServiceImpl used to create cache instance
type ConfigServiceImpl struct {
	mtx          sync.RWMutex
	cacheMap     map[string]cache
	configHashes map[string]string
}

var instance = newConfigService()

//GetInstance gets instance of cache for snaps
func GetInstance() api.ConfigService {
	return instance
}

func newConfigService() *ConfigServiceImpl {
	service := &ConfigServiceImpl{}
	service.cacheMap = make(map[string]cache)
	service.configHashes = make(map[string]string)
	return service
}

//Initialize will be called from config snap
func Initialize(stub shim.ChaincodeStubInterface, mspID string) *ConfigServiceImpl {
	instance.Refresh(stub, mspID)
	return instance
}

//Get items from cache
func (csi *ConfigServiceImpl) Get(channelID string, configKey api.ConfigKey) ([]byte, bool, errors.Error) {
	if csi == nil {
		return nil, false, errors.New(errors.SystemError, "ConfigServiceImpl was not initialized")
	}
	if configKey.AppVersion == "" {
		configKey.AppVersion = api.VERSION
	}

	channelCache := csi.getCache(channelID, configKey.MspID)
	if channelCache == nil {
		logger.Debugf("Config cache is not initialized for channel [%s]. Getting config from ledger.\n", channelID)
		return csi.GetConfigFromLedger(channelID, configKey)
	}

	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, false, err
	}

	val := channelCache[keyStr]
	if len(val) == 0 {
		logger.Debugf("Config cache does not contain config for key [%s] on channel [%s]. Getting config from ledger.\n", keyStr, channelID)
		//not in cache get from ledger
		return csi.GetConfigFromLedger(channelID, configKey)
	}
	return val, csi.isConfigDirty(keyStr, val), nil
}

//GetFromCache get items from cache
func (csi *ConfigServiceImpl) GetFromCache(channelID string, configKey api.ConfigKey) ([]byte, errors.Error) {
	if csi == nil {
		return nil, errors.New(errors.SystemError, "ConfigServiceImpl was not initialized")
	}
	if configKey.AppVersion == "" {
		configKey.AppVersion = api.VERSION
	}

	channelCache := csi.getCache(channelID, configKey.MspID)
	if channelCache == nil {
		return nil, errors.Errorf(errors.SystemError, "Config cache is not initialized for channel [%s]", channelID)
	}

	keyStr, err := mgmt.ConfigKeyToString(configKey)
	if err != nil {
		return nil, err
	}

	val := channelCache[keyStr]
	if len(val) == 0 {
		return nil, errors.Errorf(errors.SystemError, "Config cache does not contain config for key [%s] on channel [%s]", keyStr, channelID)
	}
	return channelCache[keyStr], nil
}

//GetViper configuration as Viper
func (csi *ConfigServiceImpl) GetViper(channelID string, configKey api.ConfigKey, configType api.ConfigType) (*viper.Viper, bool, errors.Error) {
	configData, dirty, err := csi.Get(channelID, configKey)
	if err != nil {
		return nil, false, err
	}
	if len(configData) == 0 {
		// No config found for the key. Return nil instead of an error so that the caller can differentiate between the two cases
		return nil, false, nil
	}

	v := viper.New()
	v.SetConfigType(string(configType))
	e := v.ReadConfig(bytes.NewBuffer(configData))
	if e != nil {
		return nil, false, errors.WithMessage(errors.InitializeConfigError, e, "snap_config_init_error")
	}
	return v, dirty, err
}

//Refresh adds new items into cache and refreshes existing ones
func (csi *ConfigServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) errors.Error {
	if metrics.IsDebug() {
		stopwatch := metrics.RootScope.Timer("config_service_refresh_time_seconds").Start()
		defer stopwatch.Stop()
	}

	logger.Debugf("***Refreshing mspid %s at %v\n", mspID, time.Unix(time.Now().Unix(), 0))
	if csi == nil {
		return errors.New(errors.SystemError, "ConfigServiceImpl was not initialized")
	}
	if stub == nil {
		return errors.New(errors.SystemError, "Stub is nil")
	}

	configManager := mgmt.NewConfigManager(stub)
	//get all by mspID
	configKey := api.ConfigKey{MspID: mspID}
	configMessages, err := configManager.Get(configKey)
	if err != nil {
		return err
	}

	if len(configMessages) == 0 {
		return errors.Errorf(errors.SystemError, "Cannot create criteria for search by mspID %v", configMessages)
	}

	return csi.refreshCache(stub.GetChannelID(), configMessages, mspID)
}

//GetConfigFromLedger - gets snaps configs from ledger
func (csi *ConfigServiceImpl) GetConfigFromLedger(channelID string, configKey api.ConfigKey) ([]byte, bool, errors.Error) {

	logger.Debugf("Getting key [%#v] on channel [%s]", configKey, channelID)
	lgr := peer.GetLedger(channelID)

	if lgr != nil {
		logger.Debugf("****Ledger is set for channelID %s\n", channelID)
		r := generateRandomAlphaOnlyString(12)
		txsim, err := lgr.NewTxSimulator(r)
		if err != nil {
			errObj := errors.WithMessage(errors.SystemError, err, "Cannot create transaction simulator")
			logger.Errorf("Get config from ledger failed: %s", errObj.GenerateLogMsg())
			return nil, false, errObj
		}
		defer txsim.Done()

		keyStr, e := mgmt.ConfigKeyToString(configKey)
		if e != nil {
			return nil, false, e
		}
		config, err := txsim.GetState("configurationsnap", keyStr)
		if err != nil {
			errObj := errors.WithMessage(errors.SystemError, err, fmt.Sprintf("Error getting state for app %s %s", keyStr, err))
			logger.Errorf("Get config from ledger failed: %s", errObj.GenerateLogMsg())
			return nil, false, errObj
		}
		return config, csi.isConfigDirty(keyStr, config), nil
	}
	return nil, false, errors.Errorf(errors.SystemError, "Cannot obtain ledger for channel %s", channelID)
}

//isConfigDirty checks if config retrieved for given key string is updated since its last retrieval.
// it checks hash of config bytes previously to current one, if there is a mismatch then returns true
func (csi *ConfigServiceImpl) isConfigDirty(keyStr string, config []byte) bool {

	if len(config) == 0 {
		return false
	}

	var dirtyFlag bool
	currentHash := csi.generateHash(config)

	csi.mtx.RLock()
	hash, ok := csi.configHashes[keyStr]
	if ok {
		dirtyFlag = !(currentHash == hash)
	} else {
		dirtyFlag = true
	}
	csi.mtx.RUnlock()

	if dirtyFlag {
		//if there is config update then update hash values in map
		csi.mtx.Lock()
		csi.configHashes[keyStr] = currentHash
		csi.mtx.Unlock()
	}

	return dirtyFlag
}

// generateHash generates hash for give bytes
func (csi *ConfigServiceImpl) generateHash(bytes []byte) string {
	digest := sha256.Sum256(bytes)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func (csi *ConfigServiceImpl) refreshCache(channelID string, configMessages []*api.ConfigKV, mspID string) errors.Error {
	if csi == nil {
		return errors.New(errors.SystemError, "ConfigServiceImpl was not initialized")
	}

	logger.Debugf("Updating cache for channel %s\n", channelID)

	cache := make(map[string][]byte)
	compCache := make(map[string][]*api.ComponentConfig)

	for _, val := range configMessages {
		keyStr, err := mgmt.ConfigKeyToString(val.Key)
		if err != nil {
			return err
		}
		logger.Debugf("Adding item for key [%s] and channel [%s] to cache\n", keyStr, channelID)
		cache[keyStr] = val.Value
		if val.Key.ComponentName != "" {
			key := val.Key
			key.ComponentVersion = ""
			keyStr, err = mgmt.ConfigKeyToString(key)
			if err != nil {
				return err
			}
			compConfig := api.ComponentConfig{}
			err := json.Unmarshal(val.Value, &compConfig)
			if err != nil {
				return errors.Wrap(errors.UnmarshalError, err, "Error occurred while un-marshalling")
			}
			if _, ok := compCache[keyStr]; !ok {
				compCache[keyStr] = make([]*api.ComponentConfig, 0)
			}
			compCache[keyStr] = append(compCache[keyStr], &compConfig)
		}
	}
	for key, comps := range compCache {
		compsBytes, e := json.Marshal(comps)
		if e != nil {
			return errors.WithMessage(errors.SystemError, e, "Failed to marshal component")
		}
		cache[key] = compsBytes
	}
	csi.mtx.Lock()
	defer csi.mtx.Unlock()
	instance.cacheMap[channelID+"_"+mspID] = cache

	logger.Debugf("Updated cache for channel %s\n", channelID)
	return nil
}

func (csi *ConfigServiceImpl) getCache(channelID, mspID string) cache {
	csi.mtx.RLock()
	defer csi.mtx.RUnlock()
	return csi.cacheMap[channelID+"_"+mspID]
}

// generateRandomAlphaOnlyString generates an alphabetical random string with length n.
func generateRandomAlphaOnlyString(n int) string {
	// make a really long string
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
