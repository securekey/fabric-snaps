/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"

	"strings"

	"github.com/spf13/viper"

	"sync"
	"time"

	"github.com/docker/docker/pkg/stringutils"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configmanager/api"
	cfgapi "github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
)

var logger = logging.MustGetLogger("configmngmt-service")

type cache map[string][]byte

//ConfigServiceImpl used to create cache instance
type ConfigServiceImpl struct {
	initialized bool
	mtx         sync.RWMutex
	cacheMap    map[string]cache
}

var instance *ConfigServiceImpl
var once sync.Once

//list of snaps
var apps = []string{"configurationsnap", "txnsnap", "eventsnap", "httpsnap"}

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
	v.ReadConfig(bytes.NewBuffer(configData))

	return v, err
}

//Refresh adds new items into cache and refreshes existing ones
func (csi *ConfigServiceImpl) Refresh(stub shim.ChaincodeStubInterface, mspID string) error {
	logger.Infof("***Refreshing mspid %s at %v\n", mspID, time.Unix(time.Now().Unix(), 0))
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

//GetConfigsFromLedger is udes to download snap config from ledger and put it
//in cache instance. Once cache instance was access to ledger will stop
func GetConfigsFromLedger(channelID string, mspID string, peerID string) error {

	csi := GetInstance()
	instance := csi.(*ConfigServiceImpl)
	var stateMessages []*cfgapi.ConfigKV
	keyDivider := mgmt.KeyDivider
	if !instance.initialized {
		lgr := peer.GetLedger(channelID)

		if lgr != nil {
			logger.Debugf("****Ledger is set for channelID %s\n", channelID)
			r := stringutils.GenerateRandomAlphaOnlyString(12)
			txsim, err := lgr.NewTxSimulator(r)
			if err != nil {
				logger.Errorf("Cannot create transaction simulator %v", err)
				return errors.Errorf("Cannot create transaction simulator %v ", err)
			}
			defer txsim.Done()

			//config for listed apps will be downloaded from HL
			for _, app := range apps {

				keyParts := []string{mspID, peerID, app}
				stateKey := strings.Join(keyParts, keyDivider)
				state, err := txsim.GetState("configurationsnap", stateKey)
				if err != nil {
					logger.Errorf("Error getting state for app %s %v", app, err)
					return errors.Errorf("Error getting state %v", err)
				}
				//only valid state (non empty) will be put in cache instance
				if len(state) > 0 {
					apiConfigKey := cfgapi.ConfigKey{MspID: mspID, PeerID: peerID, AppName: app}
					kv := cfgapi.ConfigKV{Key: apiConfigKey, Value: state}
					stateMessages = append(stateMessages, &kv)
					logger.Debugf("***Config message (HL state) %s=%s\n", stateKey, string(state))
				}
			}
			if len(stateMessages) == len(apps) {
				logger.Debugf("Instance was initialized")
				instance.initialized = true
			}
			return instance.refreshCache(channelID, stateMessages)
		}
	}
	return nil
}

func (csi *ConfigServiceImpl) refreshCache(channelID string, configMessages []*api.ConfigKV) error {
	if csi == nil {
		return errors.New("ConfigServiceImpl was not initialized")
	}

	logger.Infof("Updating cache for channel %s\n", channelID)

	cache := make(map[string][]byte)

	for _, val := range configMessages {
		keyStr, err := mgmt.ConfigKeyToString(val.Key)
		if err != nil {
			return err
		}
		if len(val.Value) > 0 {
			logger.Infof("Adding item to cache [%s]=[%s] for channel [%s]\n", keyStr, val.Value, channelID)
			cache[keyStr] = val.Value
		}
	}

	csi.mtx.Lock()
	defer csi.mtx.Unlock()
	instance.cacheMap[channelID] = cache

	logger.Infof("Updated cache for channel %s\n", channelID)
	return nil
}

func (csi *ConfigServiceImpl) getCache(channelID string) cache {
	csi.mtx.RLock()
	defer csi.mtx.RUnlock()
	return csi.cacheMap[channelID]
}
