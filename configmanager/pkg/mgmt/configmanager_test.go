/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"bytes"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
)

const (
	//number of records to be inserted in hyperledger for valid configuration test
	numOfRecords = 6
	//config contentCannot save state Configuration must be provided
	testConfigValue = "configstringgoeshere"
	//valid message contains one MSP, two peers each having 3 apps
	validMsg = `{"MspID":"msp.one","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"appName","Config":"configstringgoeshere"},{"AppName":"appNameOne","Config":"abcde"},{"AppName":"appNameTwo","Config":"mnopq"}]},{"PeerID":"peer.one.example.com","App":[{"AppName":"appName","Config":"Q29uZmln"},{"AppName":"appNameOne","Config":"ZXMgaGVyZQ"},{"AppName":"appNameTwo","Config":"BL"}]}]}`
	//misconfigured messages
	nonsenseMsg     = `{"MspID":"asd"}`
	noAppMsg        = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com"}]}`
	noConfigMsg     = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testApp"}]}]}`
	noAppIDMsg      = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"Config":"Qkw="}]}]}`
	emptyAppNameMsg = `{"MspID":"asd","Peers":[{"PeerID":"peer.zero.example.com","App":[{"AppName":""}]}]}`
	noPeerIDMsg     = `{"MspID":"","Peers":[{"App":[{"AppName":"aaa"}]}]}`
	emptyPeerIDMsg  = `{"MspID":"asd","Peers":[{"PeerID":"","App":[{"AppName":"name","Config":"Qkw="}]}]}`
	emptyPeersMsg   = `{"MspID":"asd","Peers":[]}`
)

func TestValidConfiguration(t *testing.T) {
	b := []byte(validMsg)
	keyConfigMap, err := parseConfigMessage(b)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	//verify that key exists in map
	key, _ := createConfigKey("msp.one", "peer.zero.example.com", "appName")
	_, present := keyConfigMap[key]
	if present == false {
		t.Fatalf("Key : %s should be in the map", key)
	}

	//verify that key does not exists in map
	key, _ = createConfigKey("non.existing.msp", "peer.zero.example.com", "appName")
	_, present = keyConfigMap[key]
	if present == true {
		t.Fatalf("Key : %s should NOT be in map", key)
	}

	//verify number of records to be saved to hyperledger
	if len(keyConfigMap) != numOfRecords {
		t.Fatalf("Expected : %d key/value records. Got %d", numOfRecords, len(keyConfigMap))
	}
}

func TestInvalidConfigurations(t *testing.T) {
	//loop through list of misconfigured configuration message
	invalidMessages := []string{nonsenseMsg, noAppMsg,
		noConfigMsg, noAppIDMsg,
		emptyAppNameMsg, noPeerIDMsg,
		emptyPeerIDMsg, emptyPeersMsg}

	for _, message := range invalidMessages {
		b := []byte(message)
		_, err := parseConfigMessage(b)
		if err == nil {
			t.Fatalf("Expected error for message %s", message)
		}
	}

}

func TestConfigKeyToString(t *testing.T) {
	key := api.ConfigKey{MspID: "abc", PeerID: "peer.zero.sk.example", AppName: "testApp"}
	keyStr := configKeyToString(key)
	expectedKeyString := "abc_peer.zero.sk.example_testApp"
	if keyStr != expectedKeyString {
		t.Fatalf("Expected key string %s. Got %s", expectedKeyString, keyStr)
	}
}

func TestInstantiateConfigManager(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
}

func TestSaveValidConfig(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save state %s", err)
	}
	stub.MockTransactionEnd("saveConfiguration")
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

func TestGetWithValidKey(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save configuration message %s", err)
	}
	key, _ := createConfigKey("msp.one", "peer.zero.example.com", "appName")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot get config for key %s %s", key, err)
	}
	if len(config) == 0 {
		t.Fatalf("Cannot get config content for key %s", key)
	}
	if bytes.Equal([]byte(testConfigValue), config[:]) {
		t.Fatalf("Stored and retrieved content are not the same.Expected %s received %s", testConfigValue, string(config[:]))
	}
	stub.MockTransactionEnd("saveConfiguration")
}

func TestGetWithInvalidKey(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save configuration message %s", err)
	}
	key := api.ConfigKey{MspID: "abc", PeerID: ""}
	if _, err := configManager.Get(key); err == nil {
		t.Fatalf("Expected error 'Cannot create key using mspID:abc , peerID , appName'")
	}
	stub.MockTransactionEnd("saveConfiguration")
}

func TestGetWithNonExistingKey(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save configuration message %s", err)
	}
	key, _ := createConfigKey("msp.one.does.not.exist", "peer.zero.example.com", "appName")
	config, err := configManager.Get(key)
	if err != nil {
		t.Fatalf("Cannot get config for key %s", err)
	}
	if len(config) > 0 {
		t.Fatalf("Should not get any config for non-existing key %s", key)
	}
	stub.MockTransactionEnd("saveConfiguration")
}

func TestDeleteWithValidKey(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save configuration message %s", err)
	}
	key, _ := createConfigKey("msp.one", "peer.zero.example.com", "appName")
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Cannot delete config for  key %s %s", key, err)
	}
	stub.MockTransactionEnd("saveConfiguration")
}

func TestDeleteWithNonExistingKey(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	b := []byte(validMsg)
	if err := configManager.Save(b); err != nil {
		t.Fatalf("Cannot save configuration message %s", err)
	}
	key, _ := createConfigKey("msp.one.some.bogus.key", "peer.zero.example.com", "appName")
	if err := configManager.Delete(key); err != nil {
		t.Fatalf("Cannot delete config for  key %s %s", key, err)
	}
	stub.MockTransactionEnd("saveConfiguration")
}
func TestDeleteWithInvalidKey(t *testing.T) {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	configManager := NewConfigManager(stub)
	if configManager == nil {
		t.Fatal("Cannot instantiate config manager")
	}
	key := api.ConfigKey{MspID: ""}
	if err := configManager.Delete(key); err == nil {
		t.Fatalf("Expected error 'Cannot create key using mspID: , peerID , appName'")
	}
	stub.MockTransactionEnd("saveConfiguration")
}
