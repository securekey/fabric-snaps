/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var txnSnapConfig *viper.Viper
var coreConfig *viper.Viper
var c transactionsnapApi.Config
var channelID = "testChannel"
var mspID = "Org1MSP"
var rawCert = "-----BEGIN CERTIFICATE-----\n" +
	"MIICNjCCAd2gAwIBAgIRAMnf9/dmV9RvCCVw9pZQUfUwCgYIKoZIzj0EAwIwgYEx\n" +
	"CzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4g\n" +
	"RnJhbmNpc2NvMRkwFwYDVQQKExBvcmcxLmV4YW1wbGUuY29tMQwwCgYDVQQLEwND\n" +
	"T1AxHDAaBgNVBAMTE2NhLm9yZzEuZXhhbXBsZS5jb20wHhcNMTcxMTEyMTM0MTEx\n" +
	"WhcNMjcxMTEwMTM0MTExWjBpMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZv\n" +
	"cm5pYTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNjbzEMMAoGA1UECxMDQ09QMR8wHQYD\n" +
	"VQQDExZwZWVyMC5vcmcxLmV4YW1wbGUuY29tMFkwEwYHKoZIzj0CAQYIKoZIzj0D\n" +
	"AQcDQgAEZ8S4V71OBJpyMIVZdwYdFXAckItrpvSrCf0HQg40WW9XSoOOO76I+Umf\n" +
	"EkmTlIJXP7/AyRRSRU38oI8Ivtu4M6NNMEswDgYDVR0PAQH/BAQDAgeAMAwGA1Ud\n" +
	"EwEB/wQCMAAwKwYDVR0jBCQwIoAginORIhnPEFZUhXm6eWBkm7K7Zc8R4/z7LW4H\n" +
	"ossDlCswCgYIKoZIzj0EAwIDRwAwRAIgVikIUZzgfuFsGLQHWJUVJCU7pDaETkaz\n" +
	"PzFgsCiLxUACICgzJYlW7nvZxP7b6tbeu3t8mrhMXQs956mD4+BoKuNI\n" +
	"-----END CERTIFICATE-----"

func TestGetMspID(t *testing.T) {
	value := c.GetMspID()
	if value != coreConfig.GetString("peer.localMspId") {
		t.Fatalf("Expected GetMspID() return value %v but got %v", coreConfig.GetString("peer.localMspId"), value)
	}
}

func TestGetTLSRootCert(t *testing.T) {
	value := c.GetTLSRootCertPath()
	// Test GetTLSRootCertPath()
	if value != c.GetConfigPath(coreConfig.GetString("peer.tls.rootcert.file")) {
		t.Fatalf("Expected GetTLSRootCertPath() return value %v but got %v",
			c.GetConfigPath(coreConfig.GetString("peer.tls.rootcert.file")), value)
	}

	// Test GetTLSRootCert()
	// first prepare for testing GetTLSRootCert() below by creating a dummycert file
	value = "./dummyCert.crt"
	// override config's tls root cert path to test getting a certificate object from real file.
	coreConfig.Set("peer.tls.rootcert.file", value)
	// temporarily create a dummy root cert for testing the cert locally
	// mock a valid cert data for testing..
	err := ioutil.WriteFile(value, []byte(rawCert), os.ModePerm)
	defer os.Remove(value)

	if err != nil {
		t.Fatalf("Failed to create mock root cert file data for testing: %s", err)
	}

	_, err = ioutil.ReadFile(value)

	if err != nil {
		t.Fatalf("Failed to read created mock root cert file for testing: %s", err)
	}

	// now call GetTLSRootCert()
	cert := c.GetTLSRootCert()
	if cert == nil {
		t.Fatalf("Expected to get non nil tls root cert")
	}

	// get the real certificate object from rawCert and compare it with 'cert' variable
	block, _ := pem.Decode([]byte(rawCert))
	pub, err := x509.ParseCertificate(block.Bytes)

	if !cert.Equal(pub) {
		t.Fatalf("Expected to get '%s' for the tls root cert, but got: '%s'", pub.Raw, cert.Raw)
	}
}

func TestGetTLSCert(t *testing.T) {
	value := c.GetTLSCertPath()
	// Test GetTLSCertPath()
	if value != c.GetConfigPath(coreConfig.GetString("peer.tls.cert.file")) {
		t.Fatalf("Expected GetTLSCertPath() return value %v but got %v",
			c.GetConfigPath(coreConfig.GetString("peer.tls.cert.file")), value)
	}

	// Test GetTLSCert()
	// first prepare for testing GetTLSCert() below by creating a dummycert file
	value = "./dummayCert.crt"
	// override config's tls cert path to test getting a certificate object from real file.
	coreConfig.Set("peer.tls.cert.file", value)
	// temporarily create a dummy root cert for testing the cert locally
	// mock a valid cert data for testing..
	err := ioutil.WriteFile(value, []byte(rawCert), os.ModePerm)
	defer os.Remove(value)

	if err != nil {
		t.Fatalf("Failed to create mock root cert file data for testing: %s", err)
	}

	_, err = ioutil.ReadFile(value)

	if err != nil {
		t.Fatalf("Failed to read created mock root cert file for testing: %s", err)
	}

	// now call GetTLSCert()
	cert := c.GetTLSCert()

	if cert == nil {
		t.Fatalf("Expected to get non nil tls cert")
	}

	// get the real certificate object from rawCert and compare it with 'cert' variable
	block, _ := pem.Decode([]byte(rawCert))
	pub, err := x509.ParseCertificate(block.Bytes)

	if !cert.Equal(pub) {
		t.Fatalf("Expected to get '%s' for the tls root cert, but got: '%s'", pub.Raw, cert.Raw)
	}
}

func TestGetTLSKeyPath(t *testing.T) {
	value := c.GetTLSKeyPath()
	if value != c.GetConfigPath(coreConfig.GetString("peer.tls.key.file")) {
		t.Fatalf("Expected GetTLSKeyPath() return value %v but got %v",
			c.GetConfigPath(coreConfig.GetString("peer.tls.key.file")), value)
	}
}

func TestGetLocalPeer(t *testing.T) {
	c.GetPeerConfig().Set("peer.address", "")
	_, err := c.GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != "Peer address not found in config" {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	c.GetPeerConfig().Set("peer.address", "peer:Address")
	c.GetPeerConfig().Set("peer.events.address", "")
	_, err = c.GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != "Peer event address not found in config" {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	c.GetPeerConfig().Set("peer.events.address", "peer:EventAddress")
	_, err = c.GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if !strings.Contains(err.Error(), `parsing "Address": invalid syntax`) {
		t.Fatalf("GetLocalPeer() didn't return expected error msg. got: %s", err.Error())
	}
	c.GetPeerConfig().Set("peer.address", "peer:5050")
	_, err = c.GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if !strings.Contains(err.Error(), `parsing "EventAddress": invalid syntax`) {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	c.GetPeerConfig().Set("peer.events.address", "event:5151")
	c.GetPeerConfig().Set("peer.localMspId", "")
	_, err = c.GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != "Peer localMspId not found in config" {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	c.GetPeerConfig().Set("peer.localMspId", "mspID")
	localPeer, err := c.GetLocalPeer()
	if err != nil {
		t.Fatalf("GetLocalPeer() return error %v", err)
	}
	if localPeer.Host != "peer" {
		t.Fatalf("Expected localPeer.Host value %s but got %s",
			"peer", localPeer.Host)
	}
	if localPeer.Port != 5050 {
		t.Fatalf("Expected localPeer.Port value %d but got %d",
			5050, localPeer.Port)
	}
	if localPeer.EventHost != "peer" {
		t.Fatalf("Expected localPeer.EventHost value %s but got %s",
			"event", localPeer.Host)
	}
	if localPeer.EventPort != 5151 {
		t.Fatalf("Expected localPeer.EventPort value %d but got %d",
			5151, localPeer.EventPort)
	}
	if string(localPeer.MSPid) != "mspID" {
		t.Fatalf("Expected localPeer.MSPid value %s but got %s",
			"mspID", localPeer.MSPid)
	}

}

func TestGetConfigPath(t *testing.T) {
	if c.GetConfigPath("/") != "/" {
		t.Fatalf(`Expected GetConfigPath("/") value %s but got %s`,
			"/", "/")
	}
}

func TestMain(m *testing.M) {
	configData, err := ioutil.ReadFile("../../cmd/sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	configStr := string(configData[:])
	config := &configmanagerApi.ConfigMessage{MspID: mspID,
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "jdoe", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "txnsnap", Version: configmanagerApi.VERSION, Config: string(configStr)}}}}}
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

	c, err = NewConfig("../../cmd/sampleconfig", channelID)
	if err != nil {
		panic(err.Error())
	}

	coreConfig = c.GetPeerConfig()

	txnSnapConfig = viper.New()
	txnSnapConfig.SetConfigFile("../../cmd/sampleconfig/config.yaml")
	txnSnapConfig.ReadInConfig()

	fmt.Printf("%+v\n", txnSnapConfig.AllKeys())

	os.Exit(m.Run())

}
func getMockStub() *mockstub.MockStub {
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.SetMspID("Org1MSP")
	stub.MockTransactionStart("startTxn")
	stub.ChannelID = channelID
	return stub
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(stub *mockstub.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

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

func TestCCErrorRetryableCodes(t *testing.T) {
	os.Setenv("CORE_TXNSNAP_RETRY_CCERRORCODES", "500 501 502  ")
	codes, err := c.CCErrorRetryableCodes()
	assert.NoError(t, err)
	assert.Len(t, codes, 3)
	assert.Equal(t, int32(500), codes[0])
	assert.Equal(t, int32(501), codes[1])
	assert.Equal(t, int32(502), codes[2])

	os.Setenv("CORE_TXNSNAP_RETRY_CCERRORCODES", "500 501 string")
	codes, err = c.CCErrorRetryableCodes()
	assert.Error(t, err)
	assert.Len(t, codes, 0)

	os.Setenv("CORE_TXNSNAP_RETRY_CCERRORCODES", " ")
	codes, err = c.CCErrorRetryableCodes()
	assert.NoError(t, err)
	assert.Len(t, codes, 0)
}
