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

	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"

	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"

	"strings"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var snapConfig *viper.Viper
var c httpsnapApi.Config

var relConfigPath = "/fabric-snaps/httpsnap/cmd/config/"
var channelID = "testChannel"
var peerConfigChannelID = "testChannel-peerConfig"
var mspID = "Org1MSP"

func TestGetClientCert(t *testing.T) {
	clientCert, err := c.GetClientCert()
	if err != nil {
		t.Fatal("Not supposed to get error for getting client cert")
	}
	verifyEqual(t, clientCert, snapConfig.GetString("tls.clientCert"), "Failed to get client cert.")
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

func TestIsHeaderAllowed(t *testing.T) {

	value, err := c.IsHeaderAllowed("not-configured")
	if err != nil {
		t.Fatal(err)
	}
	if value == true {
		t.Fatal("Expected false, got true for not-configured header")
	}

	// Test exact match
	value, err = c.IsHeaderAllowed("Content-Type")
	if err != nil {
		t.Fatal(err)
	}
	if value == false {
		t.Fatal("Expected true, got false for 'Content-Type' header")
	}

	// Test mixed case (http headers are not case sensitive)
	value, err = c.IsHeaderAllowed("CONTENT-Type")
	if err != nil {
		t.Fatal(err)
	}

	if value == false {
		t.Fatal("Expected true, got false for 'content-type' header")
	}

}

func TestGetCaCerts(t *testing.T) {
	values, err := c.GetCaCerts()
	if err != nil {
		t.Fatal("Not supposed to get error for getting ca certs")
	}
	if len(values) != 2 {
		t.Fatalf("Expecting 2 certs, got %d", len(values))
	}
}

func TestIsSystemCertPoolEnabled(t *testing.T) {
	enabled := c.IsSystemCertPoolEnabled()
	if enabled == false {
		t.Fatal("Expecting system cert pool enabled")
	}
}

func TestTimeouts(t *testing.T) {

	timeout := c.TimeoutOrDefault(httpsnapApi.Global)
	if timeout.Seconds() != 10 {
		t.Fatalf("Failed to retrieve global client timeout. Expected: 10, got %f", timeout.Seconds())
	}

	timeout = c.TimeoutOrDefault(httpsnapApi.TransportTLSHandshake)
	if timeout.Seconds() != 3 {
		t.Fatalf("Failed to retrieve transport TLS handshake timeout. Expected: 3, got %f", timeout.Seconds())
	}

	timeout = c.TimeoutOrDefault(httpsnapApi.TransportResponseHeader)
	if timeout.Seconds() != 5 {
		t.Fatalf("Failed to retrieve transport response header timeout. Expected: 5, got %f", timeout.Seconds())
	}

	timeout = c.TimeoutOrDefault(httpsnapApi.TransportExpectContinue)
	if timeout.Seconds() != 5 {
		t.Fatalf("Failed to retrieve transport expect continue timeout. Expected: 5, got %f", timeout.Seconds())
	}

	timeout = c.TimeoutOrDefault(httpsnapApi.TransportIdleConn)
	if timeout.Seconds() != 10 {
		t.Fatalf("Failed to retrieve transport idle connection timeout. Expected: 10, got %f", timeout.Seconds())
	}

	timeout = c.TimeoutOrDefault(httpsnapApi.DialerTimeout)
	if timeout.Seconds() != 10 {
		t.Fatalf("Failed to retrieve dialer timeout. Expected: 10, got %f", timeout.Seconds())
	}

	timeout = c.TimeoutOrDefault(httpsnapApi.TransportIdleConn)
	if timeout.Seconds() != 10 {
		t.Fatalf("Failed to retrieve dialer keep alive. Expected: 10, got %f", timeout.Seconds())
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
	config := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe",
		App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Version: configmanagerApi.VERSION, Config: string(configData)}}}}}
	stub := getMockStub(channelID)
	configBytes, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	//upload valid message to HL
	err = uploadConfigToHL(stub, configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub, mspID)

	//Setup config for second channel where use peer tls config is enabled
	configDataStr := string(configData)
	configDataStr = strings.Replace(configDataStr, "allowPeerConfig: false", "allowPeerConfig: true", -1)
	configDataStr = strings.Replace(configDataStr, "caCerts:", "caCerts-invalid:", -1)
	configDataStr = strings.Replace(configDataStr, "clientCert:", "clientCert-invalid:", -1)
	config2 := &configmanagerApi.ConfigMessage{MspID: mspID, Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "jdoe",
		App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: "httpsnap", Version: configmanagerApi.VERSION, Config: string(configDataStr)}}}}}
	configBytes2, err := json.Marshal(config2)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	stub2 := getMockStub(peerConfigChannelID)
	//upload valid message to HL
	err = uploadConfigToHL(stub2, configBytes2)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub2, mspID)

	c, _, err = NewConfig("../sampleconfig", channelID)
	if err != nil {
		panic(err.Error())
	}

	snapConfig = viper.New()
	snapConfig.SetConfigFile("./config.yaml")
	snapConfig.ReadInConfig()

	os.Exit(m.Run())
}

func getMockStub(channelID string) *mockstub.MockStub {
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.SetMspID("Org1MSP")
	stub.MockTransactionStart("startTxn")
	stub.ChannelID = channelID
	return stub
}

//uploadConfigToHL to upload key&config to repository
func uploadConfigToHL(stub *mockstub.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}

func TestNoConfig(t *testing.T) {
	viper.Reset()
	_, dirty, err := NewConfig("abc", channelID)
	if err == nil {
		t.Fatalf("Init config should have failed.")
	}
	assert.False(t, dirty)

}

func TestTLSPeerConfig(t *testing.T) {
	peerTLSPrefix := "-----BEGIN CERTIFICATE-----"
	testConfig, dirty, err := NewConfig("../sampleconfig", peerConfigChannelID)
	if err != nil {
		panic(err.Error())
	}
	assert.True(t, dirty)

	//Test get client cert
	clientCert, err := testConfig.GetClientCert()
	if err != nil {
		t.Fatalf("Not supposed to get error while getting client cert when 'tls.allowPeerConfig' enabled and 'tls.clientCert' is missing, but got : %v", err)
	}
	if clientCert == "" {
		t.Fatal("Got empty peer config client cert when 'tls.allowPeerConfig' enabled and 'tls.clientCert' is missing")
	}
	if !strings.HasPrefix(clientCert, peerTLSPrefix) {
		t.Fatalf("Supposed to get peer config client cert when 'tls.allowPeerConfig' enabled and 'tls.clientCert' is missing, but got %v", clientCert)
	}

	//Test get ca certs
	caCerts, err := testConfig.GetCaCerts()
	if err != nil {
		t.Fatalf("Not supposed to get error while getting ca certs when 'tls.allowPeerConfig' enabled and 'tls.caCerts' is missing, but got : %v", err)
	}
	if len(caCerts) == 0 {
		t.Fatal("Got empty peer config ca certs when 'tls.allowPeerConfig' enabled and 'tls.caCerts' is missing")
	}

	for _, cacert := range caCerts {
		if !strings.HasPrefix(cacert, peerTLSPrefix) {
			t.Fatalf("Supposed to get peer config client cert when 'tls.allowPeerConfig' enabled and 'tls.caCerts' is missing, but got %v", cacert)
		}
	}

}

func TestGetCryptoProvider(t *testing.T) {
	swCryptoProvider, err := c.GetCryptoProvider()

	if err != nil {
		t.Fatal("Not supposed to get error for GetCryptoProvider for SW")
	}

	if swCryptoProvider != "SW" {
		t.Fatalf(" GetCryptoProvider expected to return 'SW' but got '%s'", swCryptoProvider)
	}
}
