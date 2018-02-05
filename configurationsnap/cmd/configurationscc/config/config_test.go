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
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
)

func TestInvalidConfig(t *testing.T) {
	_, err := New("testChannel", "./invalid")
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

func TestInitializeLogging(t *testing.T) {
	config, err := New("testChannel", "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}
	err = config.initializeLogging()
	if err != nil {
		t.Fatalf("Error initializing logging: %s", err)
	}
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
	stub := getMockStub("testChannel")
	configMsg := &configmanagerApi.ConfigMessage{MspID: "Org1MSP",
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "peer1", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "configurationsnap", Config: string(configData)}}}}}

	configBytes, err := json.Marshal(configMsg)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	//upload valid message to HL
	err = uplaodConfigToHL(m, stub, configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub, "Org1MSP")

	os.Exit(m.Run())
}
func TestFindPKCSLib(t *testing.T) {

	lib := FindPKCS11Lib("lib1")
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

func getMockStub(channelID string) *mockstub.MockStub {
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	stub.SetMspID("Org1MSP")
	return stub
}

func uplaodConfigToHL(t *testing.M, stub *mockstub.MockStub, message []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	err := configManager.Save(message)
	return err

}
