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
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
)

func TestInvalidConfig(t *testing.T) {
	_, err := New("", "./invalid")
	if err == nil {
		t.Fatalf("Expecting error for invalid config but received none")
	}
}

func TestConfigNoChannel(t *testing.T) {
	config, err := New("", "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}

	checkString(t, "PeerMspID", config.PeerMspID, "Org1MSP")
	checkString(t, "PeerID", config.PeerID, "peer1")
}

func checkString(t *testing.T, field string, value string, expectedValue string) {
	if value != expectedValue {
		t.Fatalf("Expecting [%s] for [%s] but got [%s]", expectedValue, field, value)
	}
}

func TestMain(m *testing.M) {

	configData, err := ioutil.ReadFile("../sampleconfig/config.yaml")
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
func TestFindPKCSLib(t *testing.T) {

	lib := FindPKCS11Lib("lib1,lib2,lib3")
	if lib != "" {
		t.Fatalf("Expected empty lib")
	}
}

func TestCSROptions(t *testing.T) {

	csrCfg, err := GetCSRConfigOptions("testChannel", "../sampleconfig")
	if err != nil {
		t.Fatalf("Got error while getting csr options %v", err)
	}

	if csrCfg.CommonName == "" {
		t.Fatalf("Common name is mandatory")
	}
	//country
	if csrCfg.Country == "" {
		t.Fatalf("Country name is an empty string ")
	}
	//street
	if csrCfg.StateProvince == "" {
		t.Fatalf("StateProvince name is mandatory")
	}
	if csrCfg.Locality == "" {
		t.Fatalf("Locality name is an empty string ")
	}
	//organization
	if csrCfg.Org == "" {
		t.Fatalf("Org name is an empty string ")
	}

	//organizational unit
	if len(csrCfg.OrgUnit) == 0 {
		t.Fatalf("OrgUnit name is mandatory")
	}

}

func TestBCCSPOptions(t *testing.T) {

	configKey := configmanagerApi.ConfigKey{MspID: "Org1MSP", PeerID: "peer1", AppName: "configurationsnap"}
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)

	csconfig, err := instance.GetViper("testChannel", configKey, configmanagerApi.YAML)
	if err != nil {
		t.Fatalf("Got error while getting vipers %v", err)
	}
	//test PKCS11 options
	opts, err := getPKCSOptions(csconfig)
	if err != nil {
		t.Fatalf("Got error while getting BCCSP opts %v", err)
	}
	if opts.ProviderName != "PKCS11" {
		t.Fatalf("Expected PKCS11 provider")
	}
	//test PLUGIN options
	opts, err = getPluginOptions(csconfig)
	if err != nil {
		t.Fatalf("Got error while getting BCCSP opts %v", err)
	}
	if opts.PluginOpts == nil {
		t.Fatalf("Expected PluginOpts to be set")
	}
	if opts.PluginOpts.Library == "" {
		t.Fatalf("Expected PluginOpts - Library to be set")
	}
	if opts.PluginOpts.Config == nil {
		t.Fatalf("Expected PluginOpts - Config to be set")
	}
	//Config map is requred
	pluginCfgOptsMap := opts.PluginOpts.Config
	val, _ := pluginCfgOptsMap["key"]
	if val == nil {
		t.Fatalf("Expected value for key")
	}

}

func getMockStub() *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = "testChannel"
	return stub
}

func uplaodConfigToHL(stub *shim.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}
