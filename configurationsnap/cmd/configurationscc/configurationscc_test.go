/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"strings"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/stretchr/testify/assert"
)

const (
	validMsgMultiplePeersAndApps = `{"MspID":"Org1MSP","Peers":[{"PeerID":"peer.one.one.example.com","App":[{"AppName":"appNameR","Version":"$v","Config":"configstringgoeshere"},{"AppName":"appNameB","Version":"$v","Config":"config for appNametwo"},{"AppName":"appNameC","Version":"$v","Config":"mnopq"}]},{"PeerID":"peer.two.two.example.com","App":[{"AppName":"appNameHH","Version":"1","Config":"config for appNameTwoOnPeerOne goes here"},{"AppName":"appNameMM","Version":"$v","Config":"config for appNameOneTwo goes here"},{"AppName":"appNameQQ","Version":"$v","Config":"BLTwo"}]}]}`
	invalidJSONMsg               = `{"MspID":"Org1MSP","Peers":this willnot fly[{"PeerID":"peer.zero.example.com","App":[{"AppName":"testAppName","Config":"ConfigForAppOne"}]}]}`
	validWithAppComponents       = `{"MspID":"Org1MSP","Apps":[{"AppName":"app1","Version":"1","Components":[{"Name":"comp1","Config":"{comp1 data ver 1}","Version":"1"},{"Name":"comp1","Config":"{comp1 data ver 2}","TxID":"2","Version":"2"},{"Name":"comp2","Config":"{comp2 data ver 1}","TxID":"1","Version":"1"}]}]}`
)

var aclCheckCalled bool

func TestInit(t *testing.T) {
	stub := newMockStub(nil, nil)
	res := stub.MockInit("txID", [][]byte{})
	if res.Status != shim.OK {
		t.Fatalf("Init failed: %v", res.Message)
	}
	stub.ChannelID = "testChannel"
	args := [][]byte{[]byte("testChannel")}
	res = stub.MockInit("txID", args)
	if res.Message == "" {
		t.Fatalf("Expected error peer config path ... ")
	}
	peerConfigPath = "./sampleconfig"
	res = stub.MockInit("txID", args)
	if res.Status != shim.OK {
		t.Fatalf("Init failed: %v", res.Message)
	}

}

func TestRefreshACLSuccess(t *testing.T) {
	stub := newMockStub(nil, nil)

	stub.ChannelID = "testChannel"
	args := [][]byte{[]byte("testChannel")}
	peerConfigPath = "./sampleconfig"

	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	response := refresh(stub, args)
	if response.Status != 200 {
		t.Fatalf("Refresh failed: %v", response.Message)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
}

func TestRefreshACLFailure(t *testing.T) {
	stub := newMockStub(nil, nil)

	stub.ChannelID = "testChannel"
	args := [][]byte{[]byte("testChannel")}
	peerConfigPath = "./sampleconfig"

	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: true}
	response := refresh(stub, args)
	if response.Status != 500 {
		t.Fatal("Refresh should have failed for ACL with 500 status")
	}
}

func TestInvoke(t *testing.T) {

	stub := newMockStub(nil, nil)
	testInvalidFunctionName(t, stub)

	testHealthcheck(t, stub)

}

func testInvalidFunctionName(t *testing.T, stub *mockstub.MockStub) {

	// Test function name not provided
	_, err := invoke(stub, [][]byte{})
	if err == nil {
		t.Fatal("Function name is mandatory")
	}

	// Test wrong function name provided
	_, err = invoke(stub, [][]byte{[]byte("test")})
	if err == nil {
		t.Fatal("Should have failed due to wrong function name")
	}

}

func TestGenerateCSR(t *testing.T) {
	stub := newMockStub(nil, nil)
	peerConfigPath = "./sampleconfig"
	// configuration Scc call generateCSR
	_, err := invoke(stub, [][]byte{[]byte("generateCSR")})
	if err == nil {
		t.Fatal("Expected: 'Required arguments are: [key type,ephemeral flag and CSR's signature algorithm")
	}
	_, err = invoke(stub, [][]byte{[]byte("generateCSR"),
		[]byte("keyType"), []byte("false"), []byte("sigalg"), []byte("CSRCommoName")})
	if err == nil {
		t.Fatal("Expected: 'The key algorithm is invalid. Supported options: ECDSA,ECDSAP256,ECDSAP384,RSA,RSA1024,RSA2048,RSA3072,RSA4096'")
	}

	_, err = invoke(stub, [][]byte{[]byte("generateCSR"), []byte("ECDSA"), []byte("false"), []byte("ECDSA"), []byte("CSRCommoName")})
	if err == nil {
		t.Fatal("Expected: 'Could not initialize BCCSP'")
	}

	_, err = invoke(stub, [][]byte{[]byte("generateCSR"), []byte("ECDSA"), []byte("false"), []byte("ECDSA"), []byte("CSRCommoName")})
	if err == nil {
		t.Fatal("Expected: 'Could not initialize BCCSP'")
	}

}

func TestSendRefreshRequest(t *testing.T) {
	sendRefreshRequest("testChannel", "peer1", "Org1MSP")
}

func TestNew(t *testing.T) {
	cc := New()
	if cc == nil {
		t.Fatal("Chain code is not created")
	}
}

func TestParseKey(t *testing.T) {
	var jsonBCCSP *factory.FactoryOpts
	jsonCFG := []byte(
		`{ "default": "SW", "SW":{ "security": 384, "hash": "SHA3" } }`)

	err := json.Unmarshal(jsonCFG, &jsonBCCSP)
	if err != nil {
		fmt.Printf("Could not parse JSON config [%s]", err)
		os.Exit(-1)
	}
	factory.InitFactories(jsonBCCSP)
	bccspDef := factory.GetDefault()
	testOpts := &bccsp.ECDSAKeyGenOpts{Temporary: true}
	k, err := bccspDef.KeyGen(testOpts)
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	response := parseKey(k)
	if response.Status != 200 {
		t.Fatalf("Error %v", response.Message)
	}

}
func TestCreateSnapTxRequest(t *testing.T) {
	req := createTransactionSnapRequest("ccid", "testchannel", nil, nil, nil)
	if req == nil {
		t.Fatal("Request should have been created ")
	}
}

func TestGetCSRTemplate(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	//	getCSRTemplate(channelID string, keys bccsp.Key, keyType string, sigAlgType string, csrCommonName string) (x509.CertificateRequest, error) {
	_, err := getCSRTemplate("testChannel", nil, "ECDSA", "ECDSA", "csrCommonName")
	if err == nil {
		t.Fatal("Expected: ' Alg is not supported'")
	}
	_, err = getCSRTemplate("testChannel", nil, "ECDSA", "ECDSAWithSHA1", "csrCommonName")
	if err == nil {
		t.Fatal("Expected 'Error Invalid key'")
	}

	var jsonBCCSP *factory.FactoryOpts
	jsonCFG := []byte(
		`{ "default": "SW", "SW":{ "security": 384, "hash": "SHA3" } }`)

	err = json.Unmarshal(jsonCFG, &jsonBCCSP)
	if err != nil {
		fmt.Printf("Could not parse JSON config [%s]", err)
		os.Exit(-1)
	}
	factory.InitFactories(jsonBCCSP)
	bccspDef := factory.GetDefault()
	testOpts := &bccsp.ECDSAKeyGenOpts{Temporary: true}
	k, err := bccspDef.KeyGen(testOpts)
	if err != nil {
		t.Fatalf("Error  %s", err)
	}

	_, err = getCSRTemplate("testChannel", k, "ECDSA", "ECDSAWithSHA1", "csrCommonName")
	if err != nil {
		t.Fatalf("Expected 'Error Invalid key' %s", err)
	}
}

func testHealthcheck(t *testing.T, stub *mockstub.MockStub) {
	// configuration Scc healthcheck call
	echoBytes, err := invoke(stub, [][]byte{[]byte("healthCheck")})
	if err != nil {
		t.Fatalf("Failed to call healthcheck, reason :%s", err)
	}

	logger.Infof("Message received from healthcheck: %s", echoBytes)
}

func invoke(stub *mockstub.MockStub, args [][]byte) ([]byte, error) {
	res := stub.MockInvoke("1", args)
	stub.ChannelID = "testChannel"
	if res.Status != shim.OK {
		return nil, fmt.Errorf("MockInvoke failed %s", string(res.Message))
	}
	return res.Payload, nil
}

func newMockStub(configErr error, httpErr error) *mockstub.MockStub {
	return mockstub.NewMockStub("configurationsnap", new(ConfigurationSnap))
}

func TestSavedConfigs(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	stub := getMockStub("testChannel")
	//verify that saved configs are accessible
	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "peer1", AppName: "configurationsnap"}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	response, err := invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not get saved configuration :%s", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
	expected := &[]*mgmtapi.ConfigKV{}
	json.Unmarshal(response, expected)
	for _, config := range *expected {
		if config == nil {
			t.Fatalf("Expected config")
		}
	}
}

func TestSaveACLSuccess(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	stub := getMockStub("testChannel")
	configMsgBytes := []byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1))
	funcName := []byte("save")
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	_, err := invoke(stub, [][]byte{funcName, configMsgBytes})
	if err != nil {
		t.Fatalf("Could not save configuration :%v", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
}

func TestSaveACLFailure(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	stub := getMockStub("testChannel")
	configMsgBytes := []byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1))
	funcName := []byte("save")
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: true}
	_, err := invoke(stub, [][]byte{funcName, configMsgBytes})
	if err == nil {
		t.Fatal("Save should have failed with ACL check error")
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
}

func TestGetACLSuccess(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	stub := getMockStub("testChannel")
	uplaodConfigToHL(stub, []byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1)))
	//get configuration - pass config key that has only MspID field set
	//implicitly designed criteria by MspID
	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "", AppName: "", AppVersion: ""}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	response, err := invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not get configuration :%s", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
	expected := &[]*mgmtapi.ConfigKV{}
	json.Unmarshal(response, expected)

	if len(*expected) != 6 {
		t.Fatalf("Expected six records, but got  %d", len(*expected))
	}
	//config key is explicit - expect to get only one record back
	configKey = mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "peer.one.one.example.com", AppName: "appNameB", AppVersion: api.VERSION}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	response, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not get configuration :%s", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
	expected = &[]*mgmtapi.ConfigKV{}
	json.Unmarshal(response, expected)

	if len(*expected) != 1 {
		t.Fatalf("Expected six records, but got  %d", len(*expected))
	}
}

func TestGetACLFailure(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	stub := getMockStub("testChannel")
	uplaodConfigToHL(stub, []byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1)))
	//get configuration - pass config key that has only MspID field set
	//implicitly designed criteria by MspID
	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "", AppName: "", AppVersion: ""}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: true}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("Save should have failed with ACL check error")
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
}

func TestGetFromCacheACLSuccess(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	stub := getMockStub("testChannel")

	//get configuration - pass config key that has only MspID field set
	//implicitly designed criteria by MspID
	funcName := []byte("getFromCache")
	//config key is explicit - expect to get only one record back
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", AppName: "app1", AppVersion: "1", ComponentName: "comp1"}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	response, err := invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not get configuration :%s", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
	compsConfig := &[]*api.ComponentConfig{}
	json.Unmarshal(response, &compsConfig)
	if len(*compsConfig) != 2 {
		t.Fatalf("Expected return compsConfig 2")
	}
}

func TestGetFromCacheACLFailure(t *testing.T) {
	peerConfigPath = "./sampleconfig"

	stub := getMockStub("testChannel")
	//get configuration - pass config key that has only MspID field set
	//implicitly designed criteria by MspID
	funcName := []byte("getFromCache")
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", AppName: "app1", AppVersion: "1", ComponentName: "comp1"}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %v", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: true}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("Save should have failed with ACL check error")
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
}

func TestDeleteACLSuccess(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	stub := getMockStub("testChannel")

	configManager := mgmt.NewConfigManager(stub)
	err := configManager.Save([]byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1)))

	funcName := []byte("delete")
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "peer.zero.example.com", AppName: "testAppName", AppVersion: api.VERSION}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not delete configuration :%s", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}

	configKey = mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "", AppName: "", AppVersion: ""}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err != nil {
		t.Fatalf("Could not delete configuration :%s", err)
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}

	configKey = mgmtapi.ConfigKey{MspID: "", PeerID: "", AppName: "", AppVersion: ""}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("Expect error: 'Config Key does not have valid MSPId'")
	}
	if aclCheckCalled {
		t.Fatal("ACL check call was NOT expected")
	}

	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: false}
	_, err = invoke(stub, [][]byte{funcName, nil})
	if err == nil {
		t.Fatal("Expect error: Config is empty (no key)")
	}
	if aclCheckCalled {
		t.Fatal("ACL check call was NOT expected")
	}
}

func TestDeleteACLFailure(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	stub := getMockStub("testChannel")

	configManager := mgmt.NewConfigManager(stub)
	err := configManager.Save([]byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1)))

	funcName := []byte("delete")
	configKey := mgmtapi.ConfigKey{MspID: "Org1MSP", PeerID: "peer.zero.example.com", AppName: "testAppName", AppVersion: api.VERSION}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	aclCheckCalled = false
	aclProvider = &mockACLProvider{aclFailed: true}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("Save should have failed with ACL check error")
	}
	if !aclCheckCalled {
		t.Fatal("ACL check call was expected")
	}
}

func TestGetKey(t *testing.T) {
	_, err := getKey(nil)
	if err == nil {
		t.Fatal("Expected error: Config is empty (no key)")
	}
	b := [][]byte{[]byte(""), []byte("")}
	_, err = getKey(b)
	if err == nil {
		t.Fatal("Expected error: Config is empty (no key)")
	}
	b = [][]byte{[]byte("a"), []byte("")}
	_, err = getKey(b)
	if err == nil {
		t.Fatal("Expected error:Got error unmarshalling config key")
	}
	b = [][]byte{[]byte(""), []byte("b")}
	_, err = getKey(b)
	if err == nil {
		t.Fatal("Expected error:Got error unmarshalling config key")
	}

	b = [][]byte{[]byte(""), []byte("b")}
	_, err = getKey(b)
	if err == nil {
		t.Fatal("Expected error:Got error unmarshalling config key")
	}
	ch := make(chan int)
	_, err = json.Marshal(ch)
	if err != nil {
		errStr := fmt.Sprintf("Got error while marshalling config %s", err)
		logger.Error(errStr)

	}
}

func TestGetConfigUsingInvalidKey(t *testing.T) {
	stub := getMockStub("testChannel")
	configManager := mgmt.NewConfigManager(stub)
	err := configManager.Save([]byte(strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1)))

	funcName := []byte("get")
	configKey := mgmtapi.ConfigKey{MspID: "", PeerID: "", AppName: "", AppVersion: ""}
	keyBytes, err := json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("expected error: Cannot create config key using empty MspId")
	}

	configKey = mgmtapi.ConfigKey{MspID: ""}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("expected error: Cannot create config key using empty MspId")
	}

	configKey = mgmtapi.ConfigKey{}
	keyBytes, err = json.Marshal(&configKey)
	if err != nil {
		t.Fatalf("Could not marshal key: %s", err)
	}
	_, err = invoke(stub, [][]byte{funcName, keyBytes})
	if err == nil {
		t.Fatal("expected error: Cannot create config key using empty MspId")
	}

}
func TestSaveErrors(t *testing.T) {
	stub := getMockStub("testChannel")

	aclProvider = &mockACLProvider{aclFailed: false}
	_, err := invoke(stub, getBytes("save", []string{strings.Replace(validMsgMultiplePeersAndApps, "$v", api.VERSION, -1)}))
	if err != nil {
		t.Fatalf("Could not save configuration :%s", err)
	}

	configKey := mgmtapi.ConfigKey{MspID: "", PeerID: "b", AppName: "b", AppVersion: api.VERSION}
	configKeyStr, err := mgmt.ConfigKeyToString(configKey)
	if err == nil {
		t.Fatal("expected error: Cannot create config key using empty MspId")
	}

	_, err = invoke(stub, getBytes("getConfiguration", []string{configKeyStr}))
	if err == nil {
		t.Fatalf("expected error: Cannot create config key using empty MspId  %s", err)
	}
	configKey = api.ConfigKey{MspID: "Org1MSP", PeerID: "peerOne", AppName: "AppName", AppVersion: api.VERSION}
	//pass key string instead of configkey struct
	configKeyStr, err = mgmt.ConfigKeyToString(configKey)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	_, err = invoke(stub, getBytes("getConfiguration", []string{configKeyStr}))
	if err == nil {
		t.Fatal("expected error: invalid character 'm' looking for beginning of value unmarshalling Org1MSP!peerOne!AppName")
	}
}

func TestSaveConfigurationsWithEmptyPayload(t *testing.T) {
	stub := mockstub.NewMockStub("configurationsnap", new(ConfigurationSnap))
	_, err := invoke(stub, getBytes("save", []string{""}))
	if err == nil {
		t.Fatal("Expected error : 'Config is empty-cannot be saved'")
	}

}

func TestSaveConfigurationsWithBogusPayload(t *testing.T) {
	stub := mockstub.NewMockStub("configurationsnap", new(ConfigurationSnap))
	funcName := []byte("save")
	payload := []byte(invalidJSONMsg)
	_, err := invoke(stub, [][]byte{funcName, payload})
	if err == nil {
		t.Fatalf("Expected error : 'Cannot unmarshal config message ....'%s", err)
	}

}

func TestGenerateKeyArgs(t *testing.T) {

	stub := getMockStub("testChannel")

	funcName := []byte("generateKeyPair")
	_, err := invoke(stub, [][]byte{funcName, []byte("ECDSA")})
	if err == nil {
		t.Fatal("Expected: 'Required arguments are: key type and ephemeral flag'")
	}
	_, err = invoke(stub, [][]byte{funcName, []byte("ECDSA-FAKE"), []byte("false")})
	if err == nil {
		t.Fatal("Expected: 'The key option is invalid. Valid options: [ECDSA, ECDSAP256,ECDSAP384]' ")
	}
	_, err = invoke(stub, [][]byte{funcName, []byte("ECDSA"), []byte("notbool")})
	if err == nil {
		t.Fatal("Expected: 'Ephemeral flag is not set'")
	}
	_, err = invoke(stub, [][]byte{funcName, []byte("ECDSA"), []byte("")})
	if err == nil {
		t.Fatal("Expected: 'Ephemeral flag is not set'")
	}

}
func TestGetCSRSubject(t *testing.T) {
	stub := newMockStub(nil, nil)
	peerConfigPath = "./sampleconfig"
	raw, err := getCSRSubject("testChannel", "CSRCommonName")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	peerConfigPath = "./sampleconfig"
	// configuration Scc call generateCSR
	_, err = invoke(stub, [][]byte{[]byte("generateCSR")})
	if err == nil {
		t.Fatal("Expected: 'Required arguments are: [key type,ephemeral flag and CSR's signature algorithm")
	}
	_, err = invoke(stub, [][]byte{[]byte("generateCSR"),
		[]byte("keyType"), []byte("false"), []byte("sigalg"), []byte("CSRCommoName")})
	if err == nil {
		t.Fatal("Expected: 'The key algorithm is invalid. Supported options: ECDSA,ECDSAP256,ECDSAP384,RSA,RSA1024,RSA2048,RSA3072,RSA4096'")
	}

	_, err = invoke(stub, [][]byte{[]byte("generateCSR"), []byte("ECDSA"), []byte("false"), []byte("ECDSA"), []byte("CSRCommoName")})
	if err == nil {
		t.Fatal("Expected: 'Could not initialize BCCSP'")
	}

	_, err = invoke(stub, [][]byte{[]byte("generateCSR"), []byte("ECDSA"), []byte("false"), []byte("ECDSA"), []byte("CSRCommoName")})
	if err == nil {
		t.Fatal("Expected: 'Could not initialize BCCSP'")
	}

	csr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: raw,
	})
	fmt.Printf("CSR was created: [%v]", string(csr))

	if csr == nil {
		t.Fatalf("Error %s", err)
	}

}
func TestGetBCCSPAndKeyPair(t *testing.T) {

	peerConfigPath = "./sampleconfig"
	_, _, err := getBCCSPAndKeyPair("", nil)
	if err == nil {
		t.Fatal("Expected error: 'Channel is required '")
	}
	_, _, err = getBCCSPAndKeyPair("testChannel", nil)
	if err == nil {
		t.Fatal("Expected error: 'The key gen option is required '")
	}
}

func TestGenerateKeyWithOpts(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	rsp := generateKeyWithOpts("", nil)
	if rsp.Message == "" {
		t.Fatal("Expected: Cannot obtain ledger for channel")
	}
	rsp = generateKeyWithOpts("testChannel", nil)
	if rsp.Message == "" {
		t.Fatal("Expected: The key gen option is required")
	}
	opts, _ := getKeyOpts("ECDSA", false)
	rsp = generateKeyWithOpts("testChannel", opts)
	if rsp.Message == "" {
		t.Fatal("Expected: Failed initializing PKCS11 library")
	}
}

func TestGetPublicKeyAlg(t *testing.T) {
	var alg x509.PublicKeyAlgorithm
	var err error
	peerConfigPath = "./sampleconfig"
	alg, err = getPublicKeyAlg("FAKE")
	if err == nil {
		t.Fatal("Expected error: 'Public key algorithm is not supported FAKE")
	}
	if alg != 0 {
		t.Fatal("Alg should be nil")

	}
	_, err = getPublicKeyAlg("RSA")
	if err != nil {
		t.Fatalf("Error:  %s", err)
	}
	_, err = getPublicKeyAlg("ECDSA")
	if err != nil {
		t.Fatalf("Error:  %s", err)
	}
	_, err = getPublicKeyAlg("DSA")
	if err != nil {
		t.Fatalf("Error:  %s", err)
	}
}

func TestGetCSRConfig(t *testing.T) {
	peerConfigPath = "./sampleconfig"
	cfg, err := getCSRConfig("", peerConfigPath)
	if err == nil {
		t.Fatal("Expected Error: Channel is required")
	}
	cfg, err = getCSRConfig("testChannel", peerConfigPath)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	if cfg.CommonName == "" {
		t.Fatal("Error: common name is required")
	}
	if cfg.Country == "" {
		t.Fatal("Error: country name is required")

	}
	if cfg.StateProvince == "" {
		t.Fatal("Error: province name is required")

	}
	if cfg.Locality == "" {
		t.Fatal("Error: locality name is required")

	}
	if cfg.Org == "" {
		t.Fatal("Error: organization name is required")

	}

	if cfg.OrgUnit == "" {
		t.Fatal("Error: org init name is required")

	}
	if len(cfg.DNSNames) == 0 {
		t.Fatal("Error: DNS names are required")

	}
	if len(cfg.EmailAddresses) == 0 {
		t.Fatal("Error: EmailAddresses are required")

	}
	if len(cfg.IPAddresses) == 0 {
		t.Fatal("Error: IPAddresses are required")
	}

}
func TestGetSignatureAlg(t *testing.T) {

	_, err := getSignatureAlg("ECDSAWithSHA256")
	if err != nil {
		t.Fatalf("Valid alg errors out: %s", err)
	}
	_, err = getSignatureAlg("SHA256WithRSAPSS")
	if err != nil {
		t.Fatalf("Valid alg errors out: %s", err)
	}
	_, err = getSignatureAlg("SHA256WithRSAPSS-FAKE")
	if err == nil {
		t.Fatal("Expected error invalid alg")
	}
	_, err = getSignatureAlg("ECDSAWithSHA1")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("ECDSAWithSHA1")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("ECDSAWithSHA384")
	if err != nil {
		t.Fatalf("Error %s", err)
	}

	_, err = getSignatureAlg("ECDSAWithSHA512")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA256WithRSAPSS")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA384WithRSAPSS")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA512WithRSAPSS")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("DSAWithSHA256")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("DSAWithSHA1")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA512WithRSA")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA384WithRSA")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA256WithRSA")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("SHA1WithRSA")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("MD5WithRSA")
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	_, err = getSignatureAlg("MD2WithRSA")
	if err != nil {
		t.Fatalf("Error %s", err)
	}

}

func TestGetKeyOpts(t *testing.T) {
	key, err := getKeyOpts("ECDSA", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "ECDSA" {
		t.Fatal("Expected ECDSA alg")
	}
	key, err = getKeyOpts("RSA", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "RSA" {
		t.Fatal("Expected RSA alg")
	}
	key, err = getKeyOpts("ECDSA-XXX", false)
	if err == nil {
		t.Fatal("Expected Supported options: ECDSA,ECDSAP256 ... ")
	}

	key, err = getKeyOpts("ECDSAP256", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "ECDSAP256" {
		t.Fatal("Expected ECDSAP256 alg")
	}
	key, err = getKeyOpts("ECDSAP384", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "ECDSAP384" {
		t.Fatal("Expected ECDSAP384 alg")
	}

	key, err = getKeyOpts("RSA1024", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "RSA1024" {
		t.Fatal("Expected RSA1024 alg")
	}
	key, err = getKeyOpts("RSA2048", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "RSA2048" {
		t.Fatal("Expected RSA2048 alg")
	}
	key, err = getKeyOpts("RSA3072", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "RSA3072" {
		t.Fatal("Expected RSA3072 alg")
	}
	key, err = getKeyOpts("RSA4096", false)
	if err != nil {
		t.Fatal(err.Error())
	}
	if key.Algorithm() != "RSA4096" {
		t.Fatal("Expected RSA4096 alg")
	}

}

func testNew(t *testing.T) {
	ccsnap := New()
	assert.NotNil(t, ccsnap, "ccsnap should not be nil")

}
func testConversion(t *testing.T) {
	key := api.ConfigKey{MspID: "Org1MSP", PeerID: "peerOne", AppName: "AppName"}
	c := api.ConfigKV{Key: key, Value: []byte("whatever")}
	key1 := api.ConfigKey{MspID: "Org1MSP", PeerID: "peerwo", AppName: "AppNameTwo"}
	c1 := api.ConfigKV{Key: key1, Value: []byte("whateverTwo")}
	a := []*api.ConfigKV{&c, &c1}
	b, err := json.Marshal(a)
	if err != nil {

	}
	r := []*api.ConfigKV{}
	json.Unmarshal(b, &r)
	for _, config := range r {
		if config == nil {
			t.Fatalf("Config is null")
		}
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

func uplaodConfigToHL(stub *mockstub.MockStub, message []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	err := configManager.Save(message)
	return err

}

func TestMain(m *testing.M) {
	configData, err := ioutil.ReadFile("./sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %s\n", err))
	}
	configMsg := &configmanagerApi.ConfigMessage{MspID: "Org1MSP",
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "peer1", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "configurationsnap", Version: api.VERSION, Config: string(configData)}}}}}

	stub := getMockStub("testChannel")

	configBytes, err := json.Marshal(configMsg)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	//upload valid message to HL
	configManager := mgmt.NewConfigManager(stub)
	err = configManager.Save(configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	err = uplaodConfigToHL(stub, []byte(validWithAppComponents))
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s", err))
	}
	//initialize and refresh
	configmgmtService.Initialize(stub, "Org1MSP")
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)
	instance.Refresh(stub, "Org1MSP")

	os.Exit(m.Run())
}

func getMockStub(channelID string) *mockstub.MockStub {
	stub := mockstub.NewMockStub("configurationsnap", new(ConfigurationSnap))
	stub.SetMspID("Org1MSP")
	stub.MockTransactionStart("startTxn")
	stub.ChannelID = channelID
	return stub
}

type mockACLProvider struct {
	aclFailed bool
}

func (m *mockACLProvider) CheckACL(resName string, channelID string, idinfo interface{}) error {
	aclCheckCalled = true
	if m.aclFailed {
		return fmt.Errorf("ACL failed")
	}
	return nil
}
