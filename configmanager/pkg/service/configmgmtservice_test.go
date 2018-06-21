/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"encoding/json"

	"testing"

	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
)

const (
	mspID                  = "msp.one"
	channelID              = "testChannel"
	originalConfigStr      = "ConfigForAppOne"
	refreshCongifgStr      = "ConfigForAppOneWas Refreshed. Just for fun"
	validMsg               = `{"MspID":"msp.one","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testAppName","Version":"1","Config":"ConfigForAppOne"}]}]}`
	invalidJSONMsg         = `{"MspID":"msp.one","Peers":this willnot fly[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`
	inValidMsg             = `{"MspID":"msp.one.bogus","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testAppName","Version":"1","Config":"ConfigForAppOne"}]}]}`
	validMsgRefresh        = `{"MspID":"msp.one","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testAppName","Version":"1","Config":"ConfigForAppOneWas Refreshed. Just for fun"},{"AppName":"appNameOne","Version":"1","Config":"config for appNameOne"},{"AppName":"appNameTwo","Version":"1","Config":"mnopq"}]},{"PeerID":"peer.one.example.com","App":[{"AppName":"appNameOneOnPeerOne","Version":"1","Config":"config for appNameOneOnPeerOne goes here"},{"AppName":"appNameOneOne","Version":"1","Config":"config for appNameOneOne goes here"},{"AppName":"appNameTwo","Version":"1","Config":"BLOne"}]}]}`
	validWithAppComponents = `{"MspID":"msp.one","Apps":[{"AppName":"app1","Version":"1","Components":[{"Name":"comp1","Config":"{comp1 data ver 1}","Version":"1"},{"Name":"comp1","Config":"{comp1 data ver 2}","TxID":"2","Version":"2"},{"Name":"comp2","Config":"{comp2 data ver 1}","TxID":"1","Version":"1"}]}]}`
)

func TestMngmtServiceRefreshSameKeyDifferentConfig(t *testing.T) {
	stub := getMockStub()
	stub.SetMspID(mspID)
	_, err := stub.GetCreator()
	if err != nil {
		t.Fatalf("Creator err %s", err)
	}
	//upload valid message to HL
	_, err = uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	cacheInstance := Initialize(stub, mspID)

	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com", AppName: "testAppName", AppVersion: "1"}
	originalConfig, err := cacheInstance.Get(stub.GetChannelID(), key)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//verify that original config is 'ConfigForAppOne'
	if !bytes.Equal(originalConfig, []byte(originalConfigStr)) {
		t.Fatalf("Expected to retrieve from cache  %v but got %s", originalConfigStr, string(originalConfig[:]))
	}

	//lets upload another config for the same MSP
	_, err = uplaodConfigToHL(t, stub, validMsgRefresh)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if err := cacheInstance.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}
	refreshedConfig, err := cacheInstance.Get(stub.GetChannelID(), key)
	if !bytes.Equal(refreshedConfig, []byte(refreshCongifgStr)) {
		t.Fatalf("Expected from cache %s from cache  but got %s", refreshCongifgStr, string(refreshedConfig[:]))
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestGetCacheByMspID(t *testing.T) {
	stub := getMockStub()

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	cacheInstance := Initialize(stub, mspID)

	key := api.ConfigKey{MspID: mspID, PeerID: "", AppName: ""}
	_, err = cacheInstance.Get(stub.GetChannelID(), key)
	if err == nil {
		t.Fatalf("Expected error: 'Config Key is not valid Cannot create config key using empty PeerID'")
	}

}

func TestGetCacheByCompID(t *testing.T) {
	stub := getMockStub()

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validWithAppComponents)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	cacheInstance := Initialize(stub, mspID)

	key := api.ConfigKey{MspID: mspID, AppName: "app1", AppVersion: "1", ComponentName: "comp1"}
	value, err := cacheInstance.Get(stub.GetChannelID(), key)
	if err != nil {
		t.Fatalf("Get return error %s", err)
	}
	compsConfig := &[]*api.ComponentConfig{}
	json.Unmarshal(value, &compsConfig)
	if len(*compsConfig) != 2 {
		t.Fatalf("Expected return compsConfig 2")
	}

	key = api.ConfigKey{MspID: mspID, AppName: "app1", AppVersion: "1", ComponentName: "comp1", ComponentVersion: "1"}
	value, err = cacheInstance.Get(stub.GetChannelID(), key)
	if err != nil {
		t.Fatalf("Get return error %s", err)
	}
	compConfig := api.ComponentConfig{}
	json.Unmarshal(value, &compConfig)
	if compConfig.Name != "comp1" || compConfig.Version != "1" {
		t.Fatalf("Expected return compConfig with name comp1 and version 1")
	}
}

func TestGetViper(t *testing.T) {

	peerID := "peer1"
	appName := "app1"
	version := "1"
	appConfig := `
someconfig:
  somestring: SomeValue
  someint: 10
`
	configMsg := &api.ConfigMessage{
		MspID: mspID,
		Peers: []api.PeerConfig{
			api.PeerConfig{
				PeerID: peerID,
				App: []api.AppConfig{
					api.AppConfig{
						AppName: appName,
						Version: version,
						Config:  appConfig,
					},
				},
			},
		},
	}

	msgBytes, err := json.Marshal(configMsg)
	if err != nil {
		t.Fatalf("error marshalling message to JSON: %s", err)
	}

	stub := getMockStub()

	if _, err := uplaodConfigToHL(t, stub, string(msgBytes)); err != nil {
		t.Fatalf("cannot upload %s", err)
	}

	cacheInstance := Initialize(stub, mspID)

	config, err := cacheInstance.GetViper(stub.GetChannelID(), api.ConfigKey{MspID: mspID, PeerID: peerID, AppName: "unknown app"}, api.YAML)
	if err == nil {
		t.Fatalf("Expected: Getting channel cache from ledge ")
	}
	if config != nil {
		t.Fatalf("expecting nil config")
	}
	config, err = cacheInstance.GetViper(stub.GetChannelID(), api.ConfigKey{MspID: mspID, PeerID: peerID, AppName: appName, AppVersion: "1"}, api.YAML)
	if err != nil {
		t.Fatalf("expecting error for unknown config key but got none")
	}
	if value := config.GetInt("someconfig.someint"); value != 10 {
		t.Fatalf("expected value to be [10] but got [%d]", value)
	}
	if value := config.GetString("someconfig.somestring"); value != "SomeValue" {
		t.Fatalf("expected value to be [somevalue] but got [%s]", value)
	}
}

func TestTwoChannels(t *testing.T) {

	stub := getMockStub()
	stub.SetMspID("msp.one")
	key := "msp.one!peer.zero.example.com!testAppName!1!!"
	configK, err := mgmt.StringToConfigKey(key)
	if err != nil {

	}
	cacheInstance := Initialize(stub, mspID)
	_, err = uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cacheRefresh
	err = cacheInstance.Refresh(stub, mspID)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	stub.MockTransactionEnd("saveConfiguration")

	b, err := cacheInstance.Get(channelID, configK)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//second channel
	stub1 := mockstub.NewMockStub("testConfigState", nil)
	stub1.MockTransactionStart("testTX")
	stub1.ChannelID = "channelIDTwo"
	_, err = uplaodConfigToHL(t, stub1, validMsgRefresh)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	cacheInstance = Initialize(stub1, mspID)
	b, err = cacheInstance.Get("channelIDTwo", configK)
	if len(b) == 0 {
		t.Fatalf("Error expected value here for key %s ", configK)
	}
	stub.MockTransactionEnd("testTX")
}

func TestRefreshOnNilStub(t *testing.T) {
	stub := getMockStub()

	cacheInstance := Initialize(stub, mspID)
	//do refresh cache
	if err := cacheInstance.Refresh(nil, mspID); err == nil {
		t.Fatalf("Error expected: 'Stub is nil'")
	}

}

func TestRefreshCache(t *testing.T) {
	stub := getMockStub()
	key := api.ConfigKey{}
	key.MspID = "msp.one"
	key.PeerID = "peer.zero.example.com"
	key.AppName = "testAppName"
	key.AppVersion = "1"
	cacheInstance := Initialize(stub, mspID)
	configKV := api.ConfigKV{Key: key, Value: []byte("someValue")}
	configMessages := []*api.ConfigKV{&configKV}

	err := cacheInstance.refreshCache(stub.GetChannelID(), configMessages)
	if err != nil {
		t.Fatalf("Error 'refreshing cache %s", err)
	}

	//new key
	key.MspID = "msp.one.fake"
	key.PeerID = "peer.zero.example.com"
	key.AppName = "testAppName"
	key.AppVersion = "1"
	configKV = api.ConfigKV{Key: key, Value: []byte("someValue")}
	configMessages = []*api.ConfigKV{&configKV}

	err = cacheInstance.refreshCache(stub.GetChannelID(), configMessages)
	if err != nil {
		t.Fatalf("Error 'refreshing cache %s", err)
	}

}

func TestUploadingInvalidConfig(t *testing.T) {
	stub := getMockStub()

	//uploading invalid message to HL
	_, err := uplaodConfigToHL(t, stub, invalidJSONMsg)
	if err == nil {
		t.Fatalf("Cannot upload %s", err)
	}
}
func TestMngmtServiceRefreshSameConfig(t *testing.T) {

	stub := getMockStub()

	cacheInstance := Initialize(stub, mspID)
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	err = cacheInstance.Refresh(stub, mspID)
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	//do it again
	err = cacheInstance.Refresh(stub, mspID)
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestMngmtServiceRefreshValidNonExistingKey(t *testing.T) {

	stub := getMockStub()

	cacheInstance := Initialize(stub, mspID)
	adminService := GetInstance()
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if err := cacheInstance.Refresh(stub, mspID); err != nil {
		//Found no configs for criteria ByMspID error
		t.Fatalf("Error %v", err)
	}
	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com", AppName: "testAppName", AppVersion: "1"}
	_, err = cacheInstance.Get(stub.GetChannelID(), key)
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	key = api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com.does.not.exist", AppName: "testAppName", AppVersion: "1"}
	originalConfig, err := adminService.Get(stub.GetChannelID(), key)
	//key does not exist in cache - should come from ledger
	if err == nil {
		t.Fatalf("Expected: 'Cannot obtain ledger for channel testChannel'")
	}
	if len(originalConfig) > 0 {
		t.Fatalf("Expected nil config content for non existing key")
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestGetWithInvalidKey(t *testing.T) {
	adminService := ConfigServiceImpl{}

	key := api.ConfigKey{MspID: "", PeerID: "peer.zero.example.com", AppName: "testAppName", AppVersion: "1"}
	_, err := adminService.Get("channelID", key)
	if err == nil {
		t.Fatalf("Error expected 'Cannot obtain ledger for channel'")
	}
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(t *testing.T, stub *mockstub.MockStub, message string) ([]*api.ConfigKV, error) {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(message)
	if err := configManager.Save(b); err != nil {
		return nil, err
	}
	key := api.ConfigKey{}
	key.MspID = mspID
	key.PeerID = ""
	key.AppName = ""
	configsKV, err := configManager.Get(key)
	if err != nil {
		return nil, err
	}
	return configsKV, nil
}

func getMockStub() *mockstub.MockStub {
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	stub.SetMspID("msp.one")
	return stub
}
