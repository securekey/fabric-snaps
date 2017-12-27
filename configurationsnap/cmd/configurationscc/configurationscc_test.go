/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"

	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configapi "github.com/securekey/fabric-snaps/configurationsnap/api"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/configdata"
	"github.com/stretchr/testify/assert"
)

const (
	validMsg = `{"MspID":"msp.one","Peers":
		[{"PeerID":    
				"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`
	validMsgMultiplePeersAndApps = `{"MspID":"msp.one","Peers":[{"PeerID":"peer.one.one.example.com","App":[{"AppName":"appNameR","Config":"configstringgoeshere"},{"AppName":"appNameB","Config":"config for appNametwo"},{"AppName":"appNameC","Config":"mnopq"}]},{"PeerID":"peer.two.two.example.com","App":[{"AppName":"appNameHH","Config":"config for appNameTwoOnPeerOne goes here"},{"AppName":"appNameMM","Config":"config for appNameOneTwo goes here"},{"AppName":"appNameQQ","Config":"BLTwo"}]}]}`
	invalidJSONMsg               = `{"MspID":"msp.one","Peers":this willnot fly
		[{"PeerID":    
				"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`
)

func TestInit(t *testing.T) {
	stub := newMockStub(nil, nil)
	res := stub.MockInit("txID", [][]byte{})
	if res.Status != shim.OK {
		t.Fatalf("Init failed: %v", res.Message)
	}

}

func TestInvoke(t *testing.T) {

	stub := newMockStub(nil, nil)

	testInvalidFunctionName(t, stub)

	testGetPublicKeyForLogging(t, stub)

	testHealthcheck(t, stub)
}

func testInvalidFunctionName(t *testing.T, stub *shim.MockStub) {

	// Test function name not provided
	_, err := invoke(stub, [][]byte{})
	if err == nil {
		t.Fatalf("Function name is mandatory")
	}

	// Test wrong function name provided
	_, err = invoke(stub, [][]byte{[]byte("test")})
	if err == nil {
		t.Fatalf("Should have failed due to wrong function name")
	}

}

func testGetPublicKeyForLogging(t *testing.T, stub *shim.MockStub) {

	testKey := "Sample-Key"
	testKeyID := "Sample-KeyID"
	configdata.PublicKeyForLogging = testKey
	configdata.KeyIDForLogging = testKeyID

	// Retrieve public key for logging
	registrationBytes, err := invoke(stub, [][]byte{[]byte("getPublicKeyForLogging")})
	if err != nil {
		t.Fatalf("Failed to get public key for logging, reason :%v", err)
	}

	response := &configapi.PublicKeyForLogging{}
	json.Unmarshal(registrationBytes, response)

	if response.PublicKey != testKey {
		t.Fatalf("Unexpected public key found, expected '%s' but got '%s'", testKey, response.PublicKey)
	}

	if response.KeyID != testKeyID {
		t.Fatalf("Unexpected key id found, expected '%s' but got '%s'", testKeyID, response.KeyID)
	}

}

func testHealthcheck(t *testing.T, stub *shim.MockStub) {
	// configuration Scc healthcheck call
	echoBytes, err := invoke(stub, [][]byte{[]byte("healthCheck")})
	if err != nil {
		t.Fatalf("Failed to call healthcheck, reason :%v", err)
	}

	logger.Infof("Message received from healthcheck: %s", echoBytes)
}

func invoke(stub *shim.MockStub, args [][]byte) ([]byte, error) {
	res := stub.MockInvoke("1", args)
	stub.ChannelID = "testChannel"
	if res.Status != shim.OK {
		return nil, fmt.Errorf("MockInvoke failed %s", string(res.Message))
	}
	return res.Payload, nil
}

func newMockStub(configErr error, httpErr error) *shim.MockStub {
	return shim.NewMockStub("configurationsnap", new(ConfigurationSnap))
}

func TestSave(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	_, stub := saveConfigsForTesting(t)

	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "msp.one", PeerID: "peer.zero.example.com", AppName: "testAppName"}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}

	response, err := invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}
	expected := &[]*mgmtapi.ConfigKV{}
	//configKV := &mngmtapi.ConfigKV{}
	json.Unmarshal(response, expected)
	for _, config := range *expected {
		fmt.Printf("Response %s", *config)
	}

}

func TestGet(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	response, stub := saveConfigsForTesting(t)
	//get configuration - pass config key that has only MspID field set
	//implicitly designed criteria by MspID
	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "msp.one", PeerID: "", AppName: ""}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	response, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}
	expected := &[]*mgmtapi.ConfigKV{}
	json.Unmarshal(response, expected)
	for ind, config := range *expected {
		fmt.Printf("Response %d %s\n", ind, *config)
	}
	if len(*expected) != 6 {
		t.Fatalf("Expected six records, but got  %d", len(*expected))
	}
	//config key is explicit - expect to get only one record back
	configKey = mgmtapi.ConfigKey{MspID: "msp.one", PeerID: "peer.zero.example.com", AppName: "testAppName"}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	response, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}
	expected = &[]*mgmtapi.ConfigKV{}
	json.Unmarshal(response, expected)
	for ind, config := range *expected {
		fmt.Printf("Response %d %s\n", ind, *config)
	}
	if len(*expected) != 1 {
		t.Fatalf("Expected six records, but got  %d", len(*expected))
	}
}

func TestDelete(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	_, stub := saveConfigsForTesting(t)
	funcName := []byte("delete")
	configKey := mgmtapi.ConfigKey{MspID: "msp.one", PeerID: "peer.zero.example.com", AppName: "testAppName"}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}

	configKey = mgmtapi.ConfigKey{MspID: "msp.one", PeerID: "", AppName: ""}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}

	configKey = mgmtapi.ConfigKey{MspID: "", PeerID: "", AppName: ""}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatalf("Expect error: Cannot create config key using empty MspID")
	}

	_, err = invoke(stub, [][]byte{funcName, nil})
	if err == nil {
		t.Fatalf("Expect error: Config is empty (no key)")
	}

}

func TestGetKey(t *testing.T) {
	_, err := getKey(nil)
	if err == nil {
		t.Fatalf("Expected error: Config is empty (no key)")
	}
	b := [][]byte{[]byte(""), []byte("")}
	_, err = getKey(b)
	if err == nil {
		t.Fatalf("Expected error: Config is empty (no key)")
	}
	b = [][]byte{[]byte("a"), []byte("")}
	_, err = getKey(b)
	if err == nil {
		t.Fatalf("Expected error:Got error unmarshalling config key")
	}
	b = [][]byte{[]byte(""), []byte("b")}
	_, err = getKey(b)
	if err == nil {
		t.Fatalf("Expected error:Got error unmarshalling config key")
	}

	b = [][]byte{[]byte(""), []byte("b")}
	_, err = getKey(b)
	if err == nil {
		t.Fatalf("Expected error:Got error unmarshalling config key")
	}
	ch := make(chan int)
	p, err := json.Marshal(ch)
	if err != nil {
		errStr := fmt.Sprintf("Got error while marshalling config %v", err)
		logger.Error(errStr)

	}
	fmt.Printf("%s", p)
}

func TestGetIdentity(t *testing.T) {
	_, err := getIdentity(nil)
	if err == nil {
		t.Fatalf("expected error 'Sub is nil'")
	}
	stub := newMockStub(nil, nil)
	_, err = getIdentity(stub)
	if err != nil {
		t.Fatalf("error %v", err)
	}

}
func TestGetConfigUsingInvalidKey(t *testing.T) {
	_, stub := saveConfigsForTesting(t)

	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "", PeerID: "", AppName: ""}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatalf("expected error: Cannot create config key using empty MspId")
	}

	configKey = mgmtapi.ConfigKey{MspID: ""}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatalf("expected error: Cannot create config key using empty MspId")
	}

	configKey = mgmtapi.ConfigKey{}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatalf("expected error: Cannot create config key using empty MspId")
	}

}
func TestSaveErrors(t *testing.T) {
	stub := shim.NewMockStub("configurationsnap", new(ConfigurationSnap))

	_, err := invoke(stub, getBytes("save", []string{validMsgMultiplePeersAndApps}))
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}

	configKey := mgmtapi.ConfigKey{MspID: "", PeerID: "b", AppName: "b"}
	configKeyStr, err := mgmt.ConfigKeyToString(configKey)
	if err == nil {
		t.Fatalf("expected error: Cannot create config key using empty MspId")
	}

	_, err = invoke(stub, getBytes("getConfiguration", []string{configKeyStr}))
	if err == nil {
		t.Fatalf("expected error: Cannot create config key using empty MspId  %v", err)
	}
	configKey = api.ConfigKey{MspID: "msp.one", PeerID: "peerOne", AppName: "AppName"}
	//pass key string instead of configkey struct
	configKeyStr, err = mgmt.ConfigKeyToString(configKey)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	_, err = invoke(stub, getBytes("getConfiguration", []string{configKeyStr}))
	if err == nil {
		t.Fatalf("expected error: invalid character 'm' looking for beginning of value unmarshalling msp.one!peerOne!AppName")
	}

}

func TestSaveConfigurationsWithEmptyPayload(t *testing.T) {
	stub := shim.NewMockStub("configurationsnap", new(ConfigurationSnap))
	_, err := invoke(stub, getBytes("save", []string{""}))
	if err == nil {
		t.Fatalf("Expected error : 'Config is empty-cannot be saved'")
	}

}

func TestSaveConfigurationsWithBogusPayload(t *testing.T) {
	stub := shim.NewMockStub("configurationsnap", new(ConfigurationSnap))
	funcName := []byte("save")
	payload := []byte(invalidJSONMsg)
	_, err := invoke(stub, [][]byte{funcName, payload})
	if err == nil {
		t.Fatalf("Expected error : 'Cannot unmarshal config message ....'%v", err)
	}

}

func TestNew(t *testing.T) {
	ccsnap := New()
	assert.NotNil(t, ccsnap, "ccsnap should not be nil")

}
func TestConversion(t *testing.T) {
	key := api.ConfigKey{MspID: "msp.one", PeerID: "peerOne", AppName: "AppName"}
	fmt.Printf("%v ", []byte("whatever"))
	c := api.ConfigKV{Key: key, Value: []byte("whatever")}
	key1 := api.ConfigKey{MspID: "msp.one", PeerID: "peerwo", AppName: "AppNameTwo"}
	c1 := api.ConfigKV{Key: key1, Value: []byte("whateverTwo")}
	a := []*api.ConfigKV{&c, &c1}
	fmt.Printf("***%s\n", a)
	b, err := json.Marshal(a)
	if err != nil {

	}
	r := []*api.ConfigKV{}
	json.Unmarshal(b, &r)
	for _, config := range r {
		fmt.Printf("unmarshaled: %+v\n", config)
	}

}

func getBytes(function string, args []string) [][]byte {
	bytes := make([][]byte, 0, len(args)+1)
	bytes = append(bytes, []byte(function))
	for _, s := range args {
		bytes = append(bytes, []byte(s))
	}
	return bytes
}

func saveConfigsForTesting(t *testing.T) ([]byte, *shim.MockStub) {
	stub := shim.NewMockStub("configurationsnap", new(ConfigurationSnap))
	stub.ChannelID = "testChannel"
	funcName := []byte("save")
	payload := []byte(validMsgMultiplePeersAndApps)
	response, err := invoke(stub, [][]byte{funcName, payload})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}
	return response, stub
}

func uplaodConfigToHL(stub *shim.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}

func TestMain(m *testing.M) {

	configData, err := ioutil.ReadFile("./sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	fmt.Printf("Configuration for config snap %s", string(configData))
	configMsg := &configmanagerApi.ConfigMessage{MspID: "Org1MSP",
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "peer1", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "configurationsnap", Config: string(configData)}}}}}
	stub := getMockStub()
	configBytes, err := json.Marshal(configMsg)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	//upload valid message to HL
	err = uplaodConfigToHL(stub, configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub, "Org1MSP")

	os.Exit(m.Run())
}
func getMockStub() *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = "testChannel"
	return stub
}
