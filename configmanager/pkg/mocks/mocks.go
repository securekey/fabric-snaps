/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
)

// NewMockStub creates a mock configuration snap stub
func NewMockStub(channelID string) *mockstub.MockStub {
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.SetMspID("Org1MSP")
	stub.MockTransactionStart("startTxn")
	stub.ChannelID = channelID
	return stub
}

// SaveConfig saves the config data to the configuration snap using the given stub
// and caches the data in the configuration service
func SaveConfig(stub *mockstub.MockStub, mspID, peerID, appName, ver string, configData []byte) error {
	config := &api.ConfigMessage{MspID: mspID, Peers: []api.PeerConfig{{PeerID: peerID, App: []api.AppConfig{{AppName: appName,
		Version: ver, Config: string(configData)}}}}}
	configBytes, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "error marshalling config")
	}
	if err := mgmt.NewConfigManager(stub).Save(configBytes); err != nil {
		return errors.Wrap(err, "error saving config")
	}
	service.Initialize(stub, mspID)
	return nil
}

// SaveConfigFromFile reads the config data from the given file and saves it to the configuration snap
// using the given stub and caches the data in the configuration service
func SaveConfigFromFile(stub *mockstub.MockStub, mspID, peerID, appName, ver, configFile string) error {
	configData, err := ioutil.ReadFile(configFile) // nolint: gas
	if err != nil {
		return errors.Wrap(err, "error reading config file")
	}
	return SaveConfig(stub, mspID, peerID, appName, ver, configData)
}
