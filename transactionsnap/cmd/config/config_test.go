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
	"strings"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/spf13/viper"
)

var txnSnapConfig *viper.Viper
var coreConfig *viper.Viper
var c transactionsnapApi.Config
var channelID = "testChannel"
var mspID = "Org1MSP"

func TestGetMspID(t *testing.T) {
	value := c.GetMspID()
	if value != coreConfig.GetString("peer.localMspId") {
		t.Fatalf("Expected GetMspID() return value %v but got %v", coreConfig.GetString("peer.localMspId"), value)
	}
}

func TestGetTLSRootCertPath(t *testing.T) {
	value := c.GetTLSRootCertPath()
	if value != c.GetConfigPath(coreConfig.GetString("peer.tls.rootcert.file")) {
		t.Fatalf("Expected GetTLSRootCertPath() return value %v but got %v",
			c.GetConfigPath(coreConfig.GetString("peer.tls.rootcert.file")), value)
	}
}

func TestGetTLSCertPath(t *testing.T) {
	value := c.GetTLSCertPath()
	if value != c.GetConfigPath(coreConfig.GetString("peer.tls.cert.file")) {
		t.Fatalf("Expected GetTLSCertPath() return value %v but got %v",
			c.GetConfigPath(coreConfig.GetString("peer.tls.cert.file")), value)
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
	if localPeer.Host != "grpc://peer" {
		t.Fatalf("Expected localPeer.Host value %s but got %s",
			"peer", localPeer.Host)
	}
	if localPeer.Port != 5050 {
		t.Fatalf("Expected localPeer.Port value %d but got %d",
			5050, localPeer.Port)
	}
	if localPeer.EventHost != "grpc://peer" {
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

func TestGetGRPCProtocol(t *testing.T) {
	value := c.GetGRPCProtocol()
	if (value == "grpcs://") != txnSnapConfig.GetBool("txnsnap.grpc.tls.enabled") {
		t.Fatalf("Expected GetGRPCProtocol() return value 'grpc://' but got %v", value)
	}
}

func TestMain(m *testing.M) {
	configData, err := ioutil.ReadFile("../sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	configStr := string(configData[:])
	config := &configmanagerApi.ConfigMessage{MspID: mspID,
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "jdoe", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "txnsnap", Config: configStr}}}}}
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

	coreConfig = c.GetPeerConfig()

	txnSnapConfig = viper.New()
	txnSnapConfig.SetConfigFile("../sampleconfig/config.yaml")
	txnSnapConfig.ReadInConfig()

	fmt.Printf("%+v\n", txnSnapConfig.AllKeys())

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
