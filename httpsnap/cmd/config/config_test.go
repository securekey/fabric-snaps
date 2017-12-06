/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"

	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"

	"github.com/spf13/viper"
)

var snapConfig *viper.Viper
var c httpsnapApi.Config

var relConfigPath = "/fabric-snaps/httpsnap/cmd/config/"
var channelID = "testChannel"
var mspID = "Org1MSP"

func TestGetClientCert(t *testing.T) {
	verifyEqual(t, c.GetClientCert(), snapConfig.GetString("tls.clientCert"), "Failed to get client cert.")
}
func TestGetClientKey(t *testing.T) {
	key, err := c.GetClientKey()
	if err != nil {
		t.Fatalf("GetClientKey return error %v", err)
	}

	verifyEqual(t, key, "clientKey", "Failed to get client key.")
}

func TestGetNamedClientOverride(t *testing.T) {
	clientMap, err := c.GetNamedClientOverride()
	if err != nil {
		t.Fatalf("Error from GetNamedClientOverride %v", err)
	}
	if _, exist := clientMap["abc"]; !exist {
		t.Fatalf("abc client not exist")
	}
	verifyEqual(t, clientMap["abc"].Ca, "abcCA", "Failed to get client override CA.")
	verifyEqual(t, clientMap["abc"].Crt, "abcCert", "Failed to get client override Crt.")
	verifyEqual(t, clientMap["abc"].Key, "abcKey", "Failed to get client override Key.")

}

func TestGetShemaConfig(t *testing.T) {

	value, err := c.GetSchemaConfig("non-existent/type")
	if err == nil {
		t.Fatalf("Should have failed to retrieve schema config for non-existent type.")
	}

	expected := httpsnapApi.SchemaConfig{Type: "application/json", Request: `{ "$schema": "http://json-schema.org/draft-04/schema#", "title": "Request Schema", "description": "Some product", "type": "object"}`, Response: `{ "$schema": "http://json-schema.org/draft-04/schema#", "title": "Response Schema"}`}
	value, err = c.GetSchemaConfig(expected.Type)
	if err != nil {
		t.Fatalf("Failed to get schema config. err=%s ", err)
	}
	if value.Type != expected.Type {
		t.Fatalf("Failed to get schema config. Expecting %s, got %s ", expected.Type, value.Type)
	}
	if value.Request != expected.Request {
		t.Fatalf("Failed to get schema config. Expecting %s, got %s ", expected.Request, value.Request)
	}
	if value.Response != expected.Response {
		t.Fatalf("Failed to get schema config. Expecting %s, got %s ", expected.Response, value.Response)
	}
}

func TestGetCaCerts(t *testing.T) {
	values := c.GetCaCerts()
	if len(values) != 2 {
		t.Fatalf("Expecting 2 certs, got %d", len(values))
	}
}

func verifyEqual(t *testing.T, value string, expected string, errMsg string) {
	if value != expected {
		t.Fatalf("%s. Expecting %s, got %s", errMsg, expected, value)
	}
}

func TestMain(m *testing.M) {
	configData, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	config := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe", App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Config: string(configData)}}}}}
	stub := getMockStub()
	configBytes, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	//upload valid message to HL
	err = uplaodConfigToHL(stub, configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub, mspID)

	c, err = NewConfig("../sampleconfig", channelID)
	if err != nil {
		panic(err.Error())
	}

	snapConfig = viper.New()
	snapConfig.SetConfigFile("./config.yaml")
	snapConfig.ReadInConfig()

	os.Exit(m.Run())
}

func getMockStub() *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	return stub
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(stub *shim.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}

func TestNoConfig(t *testing.T) {
	viper.Reset()
	_, err := NewConfig("abc", channelID)
	if err == nil {
		t.Fatalf("Init config should have failed.")
	}

}
