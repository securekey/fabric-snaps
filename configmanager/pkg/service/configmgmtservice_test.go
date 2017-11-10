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
	var adminService api.ConfigServiceAdmin
	var adminCache AdminServiceImpl
	adminService = &adminCache
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")

	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if err := adminService.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}
	key := api.ConfigKey{MspID: mspID, PeerID: "peer.zero.example.com", AppName: "testAppName"}
	originalConfig, err := adminService.Get(key)
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
	if err := adminService.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}

	refreshedConfig, err := adminService.Get(key)
	if !bytes.Equal(refreshedConfig, []byte(refreshCongifg)) {
		t.Fatalf("Expected to retrieve from cache %v from cache  but got %s", refreshCongifg, string(originalConfig[:]))
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestRefreshOnNilStub(t *testing.T) {
	var adminService api.ConfigServiceAdmin
	var adminCache AdminServiceImpl
	adminService = &adminCache
	//do refresh cache
	if err := adminService.Refresh(nil, mspID); err == nil {
		t.Fatalf("Error expected: 'Stub is nil'")
	}
}

func TestRefreshCache(t *testing.T) {
	configMessages := make(map[string]string)
	configMessages["someKey"] = "someValue"
	var adminCache AdminServiceImpl
	//adminService = &adminCache
	if err := refreshCache(&configMessages, &adminCache); err == nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	//reset map
	configMessages = make(map[string]string)

	key := "msp.one!peer.zero.example.com!testAppName"
	configMessages[key] = "someValue"
	//key value does not exist - combination will be saved
	if err := refreshCache(&configMessages, &adminCache); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	//key value exists - it will not be refreshed
	if err := refreshCache(&configMessages, &adminCache); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}
	configMessages[key] = "after This Cache Should be Refreshed"
	//key value exists - it will not be refreshed
	if err := refreshCache(&configMessages, &adminCache); err != nil {
		t.Fatalf("Expecting error 'Invalid config key'")
	}

}

func TestMngmtServiceRefreshSameConfig(t *testing.T) {
	var adminService api.ConfigServiceAdmin
	var adminCache AdminServiceImpl
	adminService = &adminCache
	//cacheInstance := GetInstance()
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if err := adminService.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}
	//do it again
	if err := adminService.Refresh(stub, mspID); err != nil {
		t.Fatalf("Error %v", err)
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestCreateSearchCriteriaForNonexistingMspID(t *testing.T) {
	var adminService api.ConfigServiceAdmin
	var adminCache AdminServiceImpl
	adminService = &adminCache
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, inValidMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if err := adminService.Refresh(stub, mspID); err == nil {
		//Found no configs for criteria ByMspID error
		t.Fatalf("Expected error: 'Cannot create criteria for search by mspID &map[]'")
	}

	stub.MockTransactionEnd("saveConfiguration")

}

func TestMngmtServiceRefreshValidNonExistingKey(t *testing.T) {
	var adminService api.ConfigServiceAdmin
	var adminCache AdminServiceImpl
	adminService = &adminCache
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	//upload valid message to HL
	_, err := uplaodConfigToHL(t, stub, validMsg)
	if err != nil {
		t.Fatalf("Cannot upload %s", err)
	}
	//do refresh cache
	if err := adminService.Refresh(stub, mspID); err != nil {
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
	var adminService api.ConfigServiceAdmin
	var adminCache AdminServiceImpl
	adminService = &adminCache
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
