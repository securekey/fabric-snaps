/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"encoding/json"
	"errors"
<<<<<<< HEAD
	"fmt"
=======
	"os"
>>>>>>> aebcc67... Add txID and version for config component
	"testing"

	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
)

const (
	mspID = "msp.one"
	//number of records to be inserted in hyperledger for valid configuration test
	numOfRecords = 8
	//valid messages contain one MSP, one or more peers each having one or more apps
	//for testing valid messages were configured with one msp: two peers each having three apps
	validMsg               = `{"MspID":"msp.one","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testAppName","Version":"1","Config":"config for test app name on peer zero v1"},{"AppName":"testAppName","Version":"2","Config":"config for test app name on peer zero v2"},{"AppName":"appNameOne","Version":"1","Config":"config for appNameOne v1"},{"AppName":"appNameOne","Version":"2","Config":"config for appNameOne v2"},{"AppName":"appNameTwo","Version":"1","Config":"mnopq"}]},{"PeerID":"peer.one.example.com","App":[{"AppName":"appNameOneOnPeerOne","Version":"1","Config":"config for appNameOneOnPeerOne goes here"},{"AppName":"appNameOneOne","Version":"1","Config":"config for appNameOneOne goes here"},{"AppName":"appNameTwo","Version":"1","Config":"BLOne"}]}]}`
	validMsgOne            = `{"MspID":"msp.one","Peers":[{"PeerID":"peer.one.one.example.com","App":[{"AppName":"appNameR","Version":"1","Config":"configstringgoeshere"},{"AppName":"appNameTwo","Version":"1","Config":"config for appNametwo"},{"AppName":"appNameTwo","Version":"1","Config":"mnopq"}]},{"PeerID":"peer.two.two.example.com","App":[{"AppName":"appNameTwoOnPeerOne","Version":"1","Config":"config for appNameTwoOnPeerOne goes here"},{"AppName":"appNameOneTwo","Version":"1","Config":"config for appNameOneTwo goes here"},{"AppName":"appNameTwo","Version":"1","Config":"BLTwo"}]}]}`
	validMsgForMspTwo      = `{"MspID":"msp.two","Peers":[{"PeerID":"peer.one.one.example.com","App":[{"AppName":"appNameP","Version":"1","Config":"msptwoconfigforfirstpeer"},{"AppName":"appNameThree","Version":"1","Config":"config for appNameThree"},{"AppName":"appNameTwo","Version":"1","Config":"mnopq"}]},{"PeerID":"peer.two.two.example.com","App":[{"AppName":"appNameThreeOnPeerOne","Version":"1","Config":"config for appNameThreeOnPeerOne goes here"},{"AppName":"appNameOneThree","Version":"1","Config":"config for appNameOneOnThree goes here"},{"AppName":"appNameTwo","Version":"1","Config":"BLThree"}]}]}`
	noPeerWithAppAndConfig = `{"MspID":"msp.one", "Apps": [{"AppName": "publickey", "Version": "1", "Config": "{type:a, key:b}" }]}`
	noPeerWithAppNoConfig  = `{"MspID":"msp.one", "Apps": [{"AppName": "publickey", "Version": "1" }]}`
	validWithAppComponents = `{"MspID":"msp.one","Apps":[{"AppName":"app1","Version":"1","Components":[{"Name":"comp1","Config":"{comp1 data ver 1}","TxID":"1","Version":"1"},{"Name":"comp1","Config":"{comp1 data ver 2}","TxID":"2","Version":"2"},{"Name":"comp2","Config":"{comp2 data ver 1}","TxID":"1","Version":"1"}]}]}`

	//misconfigured messages
	noPeersMsg      = `{"MspID":"asd"}`
	noPeerIDMsg     = `{"MspID":"asd","Peers":[{"App":[{"AppName":"aaa"}]}]}`
	emptyPeerIDMsg  = `{"MspID":"asd","Peers":[{"PeerID":"","App":[{"AppName":"app","Version":"1","Config":"data"}]}]}`
	noAppMsg        = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com"}]}`
	noAppIDMsg      = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"Version":"1","Config":"data"}]}]}`
	emptyAppNameMsg = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"","Version":"1","Config":"data"}]}]}`
	noVersionMsg    = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testApp","Config":"data"}]}]}`
	emptyVersionMsg = `{"MspID":"asd","Peers":[{"PeerID":"peer1","App":[{"AppName":"app","Version":"","Config":"data"}]}]}`
	noConfigMsg     = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"appname","Version":"1"}]}]}`
)

func TestConfigWithPublicKeyAppNoConfig(t *testing.T) {
	b := []byte(noPeerWithAppNoConfig)
	keyConfigMap, err := ParseConfigMessage(b)

	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	//verify that key exists in map
	key, err := CreateConfigKey(mspID, "", "publickey", "1", "", "")
	if err != nil {
		t.Fatalf("Cannot create key %v", err)
	}
	config, present := keyConfigMap[key]
	if !present {
		t.Fatalf("Key : %s should be in the map", key)
	}
	if len(config) != 0 {
		t.Fatalf("Expected no config ")
	}

}

func TestConfigWithPublicKeyApp(t *testing.T) {
	b := []byte(noPeerWithAppAndConfig)
	keyConfigMap, err := ParseConfigMessage(b)

	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	//verify that key exists in map
	key, err := CreateConfigKey(mspID, "", "publickey", "1", "", "")
	if err != nil {
		t.Fatalf("Cannot create key %v", err)
	}
	config, present := keyConfigMap[key]
	if !present {
		t.Fatalf("Key : %s should be in the map", key)
	}
	if strings.Compare(string(config), "{type:a, key:b}") != 0 {
		t.Fatalf("Expected `{type:a, key:b}` in config field, but got %s", string(config))
	}

}

func TestConfigWithComponents(t *testing.T) {
	b := []byte(validWithAppComponents)
	keyConfigMap, err := ParseConfigMessage(b)

	if err != nil {
		t.Fatalf("Error: %s", err)
	}

	//verify that key exists in map
	key, err := CreateConfigKey("msp.one", "", "app1", "1", "comp1", "1")
	if err != nil {
		t.Fatalf("Cannot create key %v", err)
	}
	config, present := keyConfigMap[key]
	if !present {
		t.Fatalf("Key : %s should be in the map", key)
	}
	value := `{"Name":"comp1","Config":"{comp1 data ver 1}","Version":"1","TxID":"1"}`
	if strings.Compare(string(config), value) != 0 {
		t.Fatalf("Expected %s in config field, but got %s", value, string(config))
	}

}

func TestValidConfiguration(t *testing.T) {
	b := []byte(validMsg)
	keyConfigMap, err := ParseConfigMessage(b)

	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	//verify that key exists in map
	key, err := CreateConfigKey(mspID, "peer.zero.example.com", "appNameTwo", "1", "", "")
	if err != nil {
		t.Fatalf("Cannot create key %v", err)
	}
	_, present := keyConfigMap[key]
	if !present {
		t.Fatalf("Key : %s should be in the map", key)
	}

	//verify that key does not exists in map
	key, _ = CreateConfigKey("non.existing.msp", "peer.zero.example.com", "appName", "1", "", "")
	_, present = keyConfigMap[key]
	if present {
		t.Fatalf("Key : %s should NOT be in map", key)
	}

	//verify that all records were inserted
	if len(keyConfigMap) != numOfRecords {
		t.Fatalf("Expected : %d key/value records. Got %d", numOfRecords, len(keyConfigMap))
	}
}

func TestInvalidConfigurations(t *testing.T) {
	//loop through list of misconfigured configuration message
	invalidMessages := []string{noPeersMsg, noPeerIDMsg,
		noAppMsg, noAppIDMsg,
		noVersionMsg, noConfigMsg, emptyAppNameMsg, emptyPeerIDMsg, emptyVersionMsg}

	for _, message := range invalidMessages {
		b := []byte(message)
		_, err := ParseConfigMessage(b)
		if err == nil {
			t.Fatalf("ExCannot create config key usingpected error for message %s", message)
		}
	}

}

func TestInstantiateConfigManager(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
}

func TestGetConfigForKey(t *testing.T) {
	key := api.ConfigKey{}
	key.MspID = "ssss"
	key.PeerID = "peerID"
	key.AppName = ""
	key.AppVersion = ""
	key.ComponentName = ""
	key.ComponentVersion = ""

	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	cmimpl := configManagerImpl{stub: stub}

	config, err := cmimpl.getConfig(key)
	if err == nil {
		t.Fatalf("Did not expect error. Key is valid %v ", key)
	}
	if len(config) > 0 {
		t.Fatalf("Did not expect any config for bogus key %v ", key)
	}

	key.PeerID = ""
	configs, err := cmimpl.getConfigs(key)
	if err != nil {
		t.Fatalf("Did not expect error. Key is valid %v ", key)
	}
	if len(configs) > 0 {
		t.Fatalf("Did not expect any config for bogus key %v ", key)
	}

}

func TestGetConfigurations(t *testing.T) {

	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	cmimpl := configManagerImpl{stub: stub}

	configs, err := cmimpl.getConfigurations("index", []string{""})
	if err != nil {
		t.Fatalf("Error %v ", err)
	}

	if len(configs) > 0 {
		t.Fatalf("no configs expected for bogus index")
	}
	configs, err = cmimpl.getConfigurations("index", []string{"abc"})
	if err != nil {
		t.Fatalf("Error %v ", err)
	}
	if len(configs) > 0 {
		t.Fatalf("no configs expected for bogus index")
	}

}

func TestPutStateSuccess(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("id")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager") //var configKey *api.ConfigKV

	}
	b := []byte(validMsg)
	//put state should pass
	if err := configManager.Save(b); err != nil {
		t.Fatalf("PutState failed %s", err)
	}
}

func TestGetFieldsForIndex(t *testing.T) {
	key := api.ConfigKey{}
	if _, err := getFieldsForIndex("abc", key); err == nil {
		t.Fatalf("Expected error:'unknown index'")
	}
	_, err := getFieldsForIndex(indexMspID, key)
	if err == nil {
		t.Fatalf("Error 'invalid key' expected")
	}
	key.MspID = "ssss"
	key.PeerID = "peerID"
	key.AppName = "appname"
	key.AppVersion = "1"
	key.ComponentName = ""
	key.ComponentVersion = ""
	_, err = getFieldsForIndex("index", key)
	if err == nil {
		t.Fatalf("Error 'unknown index' expected")
	}

}

func TestAddIndexes(t *testing.T) {
	key := api.ConfigKey{}
	key.MspID = "msp"
	key.PeerID = "peer"
	key.AppName = "appname"
	key.AppVersion = "1"
	key.ComponentName = ""
	key.ComponentVersion = ""
	configManagerImpl := configManagerImpl{}

	if err := configManagerImpl.addIndex("", key); err == nil {
		t.Fatalf("Expected error:'Index is empty' ")
	}

	if err := configManagerImpl.addIndex("dddd", key); err == nil {
		t.Fatalf("Expected error:'Cannot create config key using ...' ")
	}
	key.AppName = ""
	if err := configManagerImpl.addIndexes(key); err == nil {
		t.Fatalf("Expected error:'Cannot create config key using empty AppName")
	}

	key.AppName = "appName"
	indexes = [...]string{"abc"}
	if err := configManagerImpl.addIndexes(key); err == nil {
		t.Fatalf("Expected error:'Cannot create config error adding index [abc]: unknown index [abc]")
	}

	indexes = [...]string{""}
	if err := configManagerImpl.addIndexes(key); err == nil {
		t.Fatalf("Expected error:'error adding index []: Index is empty")
	}

	//reset to valid index
	indexes = [...]string{indexMspID}
	key = api.ConfigKey{}
	if err := configManagerImpl.addIndexes(key); err == nil {
		t.Fatalf("Expected error:'Cannot create empty config key")
	}

}

func TestGetIndexKey(t *testing.T) {
	configManagerImpl := configManagerImpl{}

	if _, err := configManagerImpl.getIndexKey("", "", nil); err == nil {
		t.Fatalf("Expected error:'Cannot create config key using ...' ")
	}
	if _, err := configManagerImpl.getIndexKey("aaa", "", nil); err == nil {
		t.Fatalf("Expected error:'Cannot create config key using ...' ")
	}

	if _, err := configManagerImpl.getIndexKey("aaa", "sdfsdfs", nil); err == nil {
		t.Fatalf("Expected error:'Cannot create config key using ...' ")
	}
	if _, err := configManagerImpl.getIndexKey("", "", []string{"a"}); err == nil {
		t.Fatalf("Expected error:'Cannot create config key using ...' ")
	}

}

func TestGetForValidComponentsOnValidKey(t *testing.T) {

	configManager, err := uploadTestMessagesToHL(validWithAppComponents)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	key, _ := CreateConfigKey("msp.one", "", "app1", "1", "comp1", "1")

	configMessages, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 1 {
		t.Fatalf("Expect exactly one config for key %v", key)
	}

	key, _ = CreateConfigKey("msp.one", "", "app1", "1", "comp1", "")

	configMessages, err = configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 2 {
		t.Fatalf("Expect exactly two config for key %v", key)
	}

}

func TestGetForValidConfigsOnValidKey(t *testing.T) {

	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	// get two Versions
	key, _ := CreateConfigKey("msp.one", "peer.zero.example.com", "testAppName", "1", "", "")
	configMessages, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 1 {
		t.Fatalf("Expect exactly one config for key %v", key)
	}
	configMsg := "config for test app name on peer zero v1"
	if string(configMessages[0].Value) != configMsg {
		t.Fatalf("Expect config (%v) but got (%v)", configMsg, string(configMessages[0].Value))
	}
	key.AppVersion = "2"
	configMessages, err = configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 1 {
		t.Fatalf("Expect exactly one config for key %v", key)
	}
	configMsg = "config for test app name on peer zero v2"
	if string(configMessages[0].Value) != configMsg {
		t.Fatalf("Expect config (%v) but got (%v)", configMsg, string(configMessages[0].Value))
	}

	//store another  config messages for msp.one
	configManager, err = uploadTestMessagesToHL(validMsgOne)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	key, _ = CreateConfigKey(mspID, "peer.zero.example.com", "appNameTwo", "1", "", "")
	configMessages, err = configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}

	configManager, err = uploadTestMessagesToHL(validMsgForMspTwo)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//look for this key
	key, _ = CreateConfigKey("msp.two", "peer.zero.example.com", "appNameTwo", "1", "", "")
	configMessages, err = configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 1 {
		t.Fatalf("Expect exactly one config for key %v", key)
	}

	//look for another key
	key, _ = CreateConfigKey("msp.two", "peer.one.example.com", "appNameP", "1", "", "")
	configMessages, err = configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 1 {
		t.Fatalf("Expect exactly one config for key %v", key)
	}
}

//valid config key has mspID, peer ID and App Name
//valid partial config key has mspID
func TestGetForValidConfigsOnPartialValidKey(t *testing.T) {

	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//look for this key
	key, _ := CreateConfigKey("msp.one", "", "", "", "", "")
	configMessages, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot query for configs %v", err)
	}
	if len(configMessages) != 8 {
		t.Fatalf("Expected 8 configs. Got %d", len(configMessages))
	}

}

func TestGetForValidConfigsOnInvalidPartialKey(t *testing.T) {

	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//look for this key
	key, _ := CreateConfigKey("", "aaaa", "", "", "", "")
	_, err = configManager.Get(key)
	if err == nil {
		t.Fatalf("Expected error: ' Error Invalid config key { aaaa }. MspID is required. ")
	}

}

func TestParseConfigMessage(t *testing.T) {
	apConfig := api.AppConfig{}
	apConfig.AppName = "abc"
	b, err := json.Marshal(apConfig)
	if err != nil {
		t.Fatalf("Cannot get ApiConfig bytes %v", err)
	}
	if _, err := ParseConfigMessage(b); err == nil {
		t.Fatalf("Expected error: 'Cannot unmarshal config message...'")
	}
	if _, err := ParseConfigMessage(nil); err == nil {
		t.Fatalf("Expected error 'Cannot unmarshal config message...'%v", err)
	}
	var config []byte
	if _, err := ParseConfigMessage(config); err == nil {
		t.Fatalf("Expected error 'Cannot unmarshal config message'")
	}

}

func TestSaveEmptyConfig(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte("")
	if err := configManager.Save(b); err == nil {
		t.Fatalf("Expected error 'Cannot save state Configuration must be provided")
	}
	stub.MockTransactionEnd("saveConfiguration")
}

func TestSaveInvalidConfig(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(noAppIDMsg)
	if err := configManager.Save(b); err == nil {
		t.Fatalf("Expected error 'Configuration message does not have proper App'")
	}
	stub.MockTransactionEnd("saveConfiguration")
}

func TestSaveConfigs(t *testing.T) {
	configs := make(map[api.ConfigKey][]byte)
	key, _ := CreateConfigKey(mspID, "peer.zero.example.com", "appNameTwo", "1", "", "")
	//value := []byte("adsf")
	//nil value is accepted
	configs[key] = nil
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	cmimpl := configManagerImpl{stub: stub}
	if err := cmimpl.saveConfigs(configs); err != nil {
		t.Fatalf("Error %v", err)
	}
	cfgKey := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com"}
	configs[cfgKey] = nil
	if err := cmimpl.saveConfigs(configs); err == nil {
		t.Fatalf("Expected 'Cannot put state. Invalid key ...")
	}

}

func TestGetWithValidKey(t *testing.T) {
	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	key, _ := CreateConfigKey(mspID, "peer.zero.example.com", "appNameTwo", "1", "", "")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot get config for key %s %s", key, err)
	}
	if len(config) == 0 {
		t.Fatalf("Cannot get config content for key %s", key)
	}

}

func TestGetWithInvalidKey(t *testing.T) {
	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	key := api.ConfigKey{MspID: "", PeerID: "peer.zero.example.com"}
	_, err = configManager.Get(key)
	if err == nil {
		t.Fatalf("Expected error 'MspID is required'")
	}

}

func TestGetWithNonExistingValidKey(t *testing.T) {
	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
<<<<<<< HEAD
	key, _ := CreateConfigKey("msp.one.does.not.exist", "peer.zero.example.com", "appName", "1", "")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot get config for key %s %s", key, err)
	}
	if len(config) == 0 {
		t.Fatalf("Cannot get config content for key %s", key)
	}
	if len(config[0].Value) != 0 {
		t.Fatalf("Value should have been empty for key %s", key)
=======
	key, _ := CreateConfigKey("msp.one.does.not.exist", "peer.zero.example.com", "appName", "1", "", "")
	_, err = configManager.Get(key)
	if err == nil {
		t.Fatalf("Expected 'Caller identity is not same as peer's MSPId'")
>>>>>>> aebcc67... Add txID and version for config component
	}
}

func TestDeleteWithValidKey(t *testing.T) {

	stub := shim.NewMockStub("testConfigStateRefresh", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatalf("Cannot instantiate config manager")
	}
	//store  config messages
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save state ")
	}

	key, _ := CreateConfigKey(mspID, "peer.zero.example.com", "testAppName", "1", "", "")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Error %v ", err)
	}
	if len(config[0].Value) == 0 {
		t.Fatalf("Config should exist for key %v ", key)
	}
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Cannot delete config for  key %s %s", key, err)
	}
	stub.MockTransactionEnd("saveConfiguration")
	stub.MockTransactionStart("a")

	config, err = configManager.Get(key)
	if len(config[0].Value) != 0 {
		t.Fatalf("Config should be deleted for key %v ", key)
	}

	stub.MockTransactionEnd("a")

}

func TestDeleteComponentWithVersion(t *testing.T) {

	stub := shim.NewMockStub("testConfigStateRefresh", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatalf("Cannot instantiate config manager")
	}
	//store config messages
	b := []byte(validWithAppComponents)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save state ")
	}

	key, _ := CreateConfigKey(mspID, "", "app1", "1", "comp1", "1")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Error %v ", err)
	}
	if len(config[0].Value) == 0 {
		t.Fatalf("Config should exist for key %v ", key)
	}
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Cannot delete config for  key %s %s", key, err)
	}
	stub.MockTransactionEnd("saveConfiguration")
	stub.MockTransactionStart("a")

	config, err = configManager.Get(key)
	if len(config[0].Value) != 0 {
		t.Fatalf("Config should be deleted for key %v ", key)
	}

	key, _ = CreateConfigKey(mspID, "", "app1", "1", "comp1", "2")
	config, err = configManager.Get(key)
	if len(config[0].Value) == 0 {
		t.Fatalf("Config should exist for key %v ", key)
	}

	stub.MockTransactionEnd("a")

}

func TestDeleteComponentWithoutVersion(t *testing.T) {

	stub := shim.NewMockStub("testConfigStateRefresh", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatalf("Cannot instantiate config manager")
	}
	//store config messages
	b := []byte(validWithAppComponents)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save state ")
	}

	key, _ := CreateConfigKey(mspID, "", "app1", "1", "comp1", "")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Error %v ", err)
	}
	if len(config[0].Value) == 0 {
		t.Fatalf("Config should exist for key %v ", key)
	}
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Cannot delete config for  key %s %s", key, err)
	}
	stub.MockTransactionEnd("saveConfiguration")
	stub.MockTransactionStart("a")

	config, err = configManager.Get(key)
	if len(config[0].Value) != 0 {
		t.Fatalf("Config should be deleted for key %v ", key)
	}

	key, _ = CreateConfigKey(mspID, "", "app1", "1", "comp1", "2")
	config, err = configManager.Get(key)
	if len(config[0].Value) != 0 {
		t.Fatalf("Config should be deleted for key %v ", key)
	}

	key, _ = CreateConfigKey(mspID, "", "app1", "1", "comp2", "1")
	config, err = configManager.Get(key)
	if len(config[0].Value) == 0 {
		t.Fatalf("Config should exist for key %v ", key)
	}

	stub.MockTransactionEnd("a")

}

func TestDeleteByMspID(t *testing.T) {

	stub := shim.NewMockStub("testConfigStateRefresh", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatalf("Cannot instantiate config manager")
	}
	//store  config messages
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save state ")
	}

	key, _ := CreateConfigKey(mspID, "", "", "1", "", "")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Error %v ", err)
	}
	if len(config) != 8 {
		t.Fatalf("Six messages should be uploaded for msp %v ", key)
	}
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Cannot delete config for  key %s %s", key, err)
	}
	stub.MockTransactionEnd("saveConfiguration")

}

func TestDeleteWithNonExistingValidKey(t *testing.T) {
	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
<<<<<<< HEAD
	key, _ := CreateConfigKey("msp.one.some.bogus.key", "peer.zero.example.com", "appName", "1", "")
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Error %v ", err)
=======
	key, _ := CreateConfigKey("msp.one.some.bogus.key", "peer.zero.example.com", "appName", "1", "", "")
	if err := configManager.Delete(key); err == nil {
		t.Fatalf("Expected 'Caller identity is not same as peer's MSPId'")
>>>>>>> aebcc67... Add txID and version for config component
	}
}

func TestDeleteWithInvalidKey(t *testing.T) {
	configManager, err := uploadTestMessagesToHL(validMsg)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	key := api.ConfigKey{MspID: ""}
	if err := configManager.Delete(key); err == nil {
		t.Fatalf("Expected error 'Invalid key....'")
	}
}

func TestSearch(t *testing.T) {
	stub := shim.NewMockStub("testConfigStateRefresh", nil)
	stub.MockTransactionStart("saveConfiguration")
	cmimpl := configManagerImpl{stub: stub}
	key := api.ConfigKey{MspID: ""}
	_, err := cmimpl.search(key)
	if err == nil {
		t.Fatalf("Expected error 'Invalid key....'")
	}
}

func TestUnmarshalConfig(t *testing.T) {
	if _, err := unmarshalConfig(nil); err == nil {
		t.Fatalf("Expected error 'No configuration passed to unmarshaller'")
	}
	var config []byte
	if _, err := unmarshalConfig(config); err == nil {
		t.Fatalf("Expected error 'No configuration passed to unmarshaller'")
	}
	v := []byte("whatever")
	if _, err := unmarshalConfig(v); err == nil {
		t.Fatalf("Expected error 'No configuration passed to unmarshaller'")
	}
}

func uploadTestMessagesToHL(msgName string) (api.ConfigManager, error) {
	stub := shim.NewMockStub("testConfigStateRefresh", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		return nil, errors.New("Cannot instantiate config manager")
	}
	//store  config messages
	b := []byte(msgName)
	if err := configManager.Save(b); err != nil {
		return nil, errors.New("Cannot save state ")
	}

	stub.MockTransactionEnd("saveConfiguration")
	return configManager, nil
}
