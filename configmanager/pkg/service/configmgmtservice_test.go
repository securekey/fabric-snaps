/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
)

const (
	mspID             = "msp.one"
	channelID         = "testChannel"
	originalConfigStr = "\"ConfigForAppOne\""
	refreshCongifgStr = "\"ConfigForAppOneWas Refreshed. Just for fun\""
	validMsg          = `{"MspID":"msp.one","Peers":
		[{"PeerID":    
				"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`
	validMsgUpgradedConfig = `{"MspID":"msp.one","Peers":
					[{"PeerID":    
							"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOneChangedHere"}]}]}`
	invalidJSONMsg = `{"MspID":"msp.one","Peers":this willnot fly
					[{"PeerID":    
							"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`

	inValidMsg = `{"MspID":"msp.one.bogus","Peers":
				[{"PeerID":    
							"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`
	validMsgRefresh = `{  
	"MspID":"msp.one",
	"Peers":[  
	   {  
		  "PeerID":"peer.zero.example.com",
		  "App":[  
			 {  
				"AppName":"testAppName",
				"Config":"ConfigForAppOneWas Refreshed. Just for fun"
			 },
			 {  
				"AppName":"appNameOne",
				"Config":"config for appNameOne"
			 },
			 {  
				"AppName":"appNameTwo",
				"Config":"mnopq"
			 }
		  ]
	   },
	   {  
		  "PeerID":"peer.one.example.com",
		  "App":[  
			 {  
				"AppName":"appNameOneOnPeerOne",
				"Config":"config for appNameOneOnPeerOne goes here"
			 },
			 {  
				"AppName":"appNameOneOne",
				"Config":"config for appNameOneOne goes here"
			 },
			 {  
				"AppName":"appNameTwo",
				"Config":"BLOne"
			 }
		  ]
	   }
	]
 }`
)

func TestMngmtServiceRefreshSameKeyDifferentConfig(t *testing.T) {
	stub := getMockStub()

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	cacheInstance := Initialize(stub, mspID)

	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com", AppName: "testAppName"}
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
	if _, err := cacheInstance.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}
	fmt.Printf("%+v", cacheInstance)
	refreshedConfig, err := cacheInstance.Get(stub.GetChannelID(), key)
	if !bytes.Equal(refreshedConfig, []byte(refreshCongifgStr)) {
		t.Fatalf("Expected from cache %s from cache  but got %s", refreshCongifgStr, string(refreshedConfig[:]))
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestCacheWasNotInitialized(t *testing.T) {
	adminService := ConfigServiceImpl{}
	stub := getMockStub()

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if _, err := adminService.Refresh(stub, mspID); err == nil {
		t.Fatalf("Expected (Refresh) 'Cache was not initialized'")
	}
	configMessages := make(map[string][]byte)
	if _, err := adminService.refreshCache(stub.GetChannelID(), &configMessages); err == nil {
		t.Fatalf("Expected (refreshCache) 'Cache was not initialized'")

	}
	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com.does.not.exist", AppName: "testAppName"}
	if _, err := adminService.Get(stub.GetChannelID(), key); err == nil {
		t.Fatalf("Expected (Get) 'Cache was not initialized'")
	}

}

func TestTwoChannels(t *testing.T) {

	stub := getMockStub()
	key := "msp.one!peer.zero.example.com!testAppName"
	configK, err := mgmt.StringToConfigKey(key)
	if err != nil {

	}
	cacheInstance := Initialize(stub, mspID)
	_, err = uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cacheRefresh
	_, err = cacheInstance.Refresh(stub, mspID)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	stub.MockTransactionEnd("saveConfiguration")

	b, err := cacheInstance.Get(channelID, configK)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//second channel
	stub1 := shim.NewMockStub("testConfigState", nil)
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
	if _, err := cacheInstance.Refresh(nil, mspID); err == nil {
		t.Fatalf("Error expected: 'Stub is nil'")
	}

}

func TestRefreshCache(t *testing.T) {
	configMessages := make(map[string][]byte)
	configMessages["someKey"] = []byte("someValue")
	stub := getMockStub()
	cacheInstance := Initialize(stub, mspID)

	if _, err := cacheInstance.refreshCache(stub.GetChannelID(), &configMessages); err == nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	//reset map
	configMessages = make(map[string][]byte)

	key := "msp.one!peer.zero.example.com!testAppName"
	configMessages[key] = []byte("someValue")
	//key value does not exist - combination will be saved
	if _, err := cacheInstance.refreshCache(stub.GetChannelID(), &configMessages); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	//key value exists - it will not be refreshed
	if _, err := cacheInstance.refreshCache(stub.GetChannelID(), &configMessages); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	configMessages[key] = []byte("after This Cache Should be Refreshed")
	//key value exists - it will not be refreshed
	if _, err := cacheInstance.refreshCache(stub.GetChannelID(), &configMessages); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
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
	wasItRefreshed, err := cacheInstance.Refresh(stub, mspID)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	if !wasItRefreshed {
		t.Fatalf("Cache should be refreshed")
	}
	//do it again
	wasItRefreshed, err = cacheInstance.Refresh(stub, mspID)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	if wasItRefreshed {
		t.Fatalf("Cache should NOT be refreshed")
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestCreateSearchCriteriaForNonexistingMspID(t *testing.T) {

	stub := getMockStub()

	cacheInstance := Initialize(stub, mspID)
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, inValidMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if _, err := cacheInstance.Refresh(stub, mspID); err == nil {
		//Found no configs for criteria ByMspID error
		t.Fatalf("Expected error: 'Cannot create criteria for search by mspID &map[]'")
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestMngmtServiceRefreshValidNonExistingKey(t *testing.T) {
	adminService := GetInstance()

	stub := getMockStub()

	cacheInstance := Initialize(stub, mspID)
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if _, err := cacheInstance.Refresh(stub, mspID); err != nil {
		//Found no configs for criteria ByMspID error
		t.Fatalf("Error %v", err)
	}
	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com.does.not.exist", AppName: "testAppName"}
	originalConfig, err := adminService.Get(stub.GetChannelID(), key)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	if len(originalConfig) > 0 {
		t.Fatalf("Expected nil config content for non existing key")
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestGetWithInvalidKey(t *testing.T) {
	adminService := ConfigServiceImpl{}

	key := api.ConfigKey{MspID: "", PeerID: "peer.zero.example.com", AppName: "testAppName"}
	_, err := adminService.Get("channelID", key)
	if err == nil {
		t.Fatalf("Error expected 'Cannot create config key using empty MspId'")
	}
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(t *testing.T, stub *shim.MockStub, message string) (*map[string][]byte, error) {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(message)
	if err := configManager.Save(b); err != nil {
		return nil, err
	}
	criteria, err := api.NewSearchCriteriaByMspID(mspID)
	if err != nil {
		return nil, err
	}
	stub.MockTransactionStart("queryConfiguration")
	//use criteria by mspID=msp.onestub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = stub.GetChannelID()
	configMessages, err := configManager.Query(criteria)
	if err != nil {
		return nil, err
	}
	return configMessages, nil
}

func getMockStub() *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	return stub
}
