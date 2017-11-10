/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"bytes"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
)

const (
	mspID           = "msp.one"
	originalCongifg = "ConfigForAppOne"
	refreshCongifg  = "ConfigForAppOneWas Refresh. Just for fun"
	validMsg        = `{"MspID":"msp.one","Peers":
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
				"Config":"ConfigForAppOneWas Refresh. Just for fun"
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

func TestMngmtServiceRefreshDifferentConfig(t *testing.T) {

	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	cacheInstance := Initialize(stub, mspID)

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if _, err := cacheInstance.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}
	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com", AppName: "testAppName"}
	originalConfig, err := cacheInstance.Get(key)
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	//verify that original config is 'ConfigForAppOne'
	if !bytes.Equal(originalConfig, []byte(originalCongifg)) {
		t.Fatalf("Expected to retrieve from cache  %v but got %s", originalCongifg, string(originalConfig[:]))
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

	refreshedConfig, err := cacheInstance.Get(key)
	if !bytes.Equal(refreshedConfig, []byte(refreshCongifg)) {
		t.Fatalf("Expected to retrieve from cache %v from cache  but got %s", refreshCongifg, string(originalConfig[:]))
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestCacheWasNotInitialized(t *testing.T) {
	adminService := ConfigServiceImpl{}
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if _, err := adminService.Refresh(stub, mspID); err == nil {
		t.Fatalf("Expected (Refresh) 'Cache was not initialized'")
	}
	configMessages := make(map[string]string)
	if _, err := adminService.refreshCache(&configMessages); err == nil {
		t.Fatalf("Expected (refreshCache) 'Cache was not initialized'")

	}
	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com.does.not.exist", AppName: "testAppName"}
	if _, err := adminService.Get(key); err == nil {
		t.Fatalf("Expected (Get) 'Cache was not initialized'")
	}

}

func TestRefreshOnNilStub(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	cacheInstance := Initialize(stub, mspID)
	//do refresh cache
	if _, err := cacheInstance.Refresh(nil, mspID); err == nil {
		t.Fatalf("Error expected: 'Stub is nil'")
	}
}

func TestRefreshCache(t *testing.T) {
	configMessages := make(map[string]string)
	configMessages["someKey"] = "someValue"
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	cacheInstance := Initialize(stub, mspID)

	if _, err := cacheInstance.refreshCache(&configMessages); err == nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	//reset map
	configMessages = make(map[string]string)

	key := "msp.one!peer.zero.example.com!testAppName"
	configMessages[key] = "someValue"
	//key value does not exist - combination will be saved
	if _, err := cacheInstance.refreshCache(&configMessages); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	//key value exists - it will not be refreshed
	if _, err := cacheInstance.refreshCache(&configMessages); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	configMessages[key] = "after This Cache Should be Refreshed"
	//key value exists - it will not be refreshed
	if _, err := cacheInstance.refreshCache(&configMessages); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}

}

func TestMngmtServiceRefreshSameConfig(t *testing.T) {

	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
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

	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
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

	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
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
	originalConfig, err := adminService.Get(key)
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
	_, err := adminService.Get(key)
	if err == nil {
		t.Fatalf("Error expected 'Cannot create config key using empty MspId'")
	}
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(t *testing.T, stub *shim.MockStub, message string) (*map[string]string, error) {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(message)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save state %s", err)
	}
	criteria, err := api.NewSearchCriteriaByMspID(mspID)
	if err != nil {
		return nil, err
	}
	stub.MockTransactionStart("queryConfiguration")
	//use criteria by mspID=msp.one
	configMessages, err := configManager.QueryForConfigs(criteria)
	if err != nil {
		return nil, err
	}
	return configMessages, nil
}
