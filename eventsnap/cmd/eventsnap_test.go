/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
	localservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/config"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	configmocks "github.com/securekey/fabric-snaps/configmanager/pkg/mocks"
)

func TestEventSnap(t *testing.T) {
	// ehMx := &sync.RWMutex{}
	channelID1 := "ch1"
	channelID2 := "ch2"

	// config0, err := config.New("", "./sampleconfig", newMockConfigServiceProvider()())
	// if err != nil {
	// 	t.Fatalf("Error getting config for channel [%s]: %s", "", err)
	// }

	peerID := "peer1"
	mspID := "Org1MSP"
	configStub1 := configmocks.NewMockStub(channelID1)
	configStub1.ChannelID = channelID1
	service.Initialize(configStub1, mspID)

	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, config.EventSnapAppName, "./sampleconfig/configch1.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}

	eventsnap := &eventSnap{
		peerConfigPath: "./sampleconfig",
	}

	stub := shim.NewMockStub("eventsnap", eventsnap)
	// Invalid options
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid2", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in init since no event hub address was specified but got OK")
	}

	// // mockEventHubs := make(map[string]*mockeventhub.MockEventHub)
	// // configProvider.setConfig("", config0)
	// eventsnap = &eventSnap{
	// 	// eropts: eventrelay.MockOpts(func(channelID string, address string, regTimeout time.Duration, adapter eventrelay.EventAdapter, tlsConfig *tls.Config) (eventrelay.EventHub, error) {
	// 	// 	fmt.Printf("Creating mock event hub for channel %s\n", channelID)
	// 	// 	mockeh := mockeventhub.New(adapter)
	// 	// 	ehMx.Lock()
	// 	// 	mockEventHubs[channelID] = mockeh
	// 	// 	ehMx.Unlock()
	// 	// 	return mockeh, nil
	// 	// }),
	// 	configProvider: configProvider,
	// }

	// stub = shim.NewMockStub("eventsnap", eventsnap)

	// Initialize
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid3", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Another channel
	stub.ChannelID = channelID2
	// configProvider.setConfig("ch2", config2)
	if resp := stub.MockInit("txid4", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Delay adding the configuration
	time.Sleep(8 * time.Second)

	// configProvider.setConfig(channelID1, config1)
	// configProvider.setConfig(channelID2, config1)

	// Wait for the event snap to pick up the configuration
	time.Sleep(6 * time.Second)

	// Init again on same channel
	if resp := stub.MockInit("txid5", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in init since init was already called for the same channel but got OK")
	}

	// Invoke should return error
	if resp := stub.MockInvoke("txid6", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in invoke since invoke is not supported but got OK")
	}

	eventService1 := localservice.Get(channelID1)
	if eventService1 == nil {
		t.Fatalf("Expecting local event service for %s to be registered but got nil", channelID1)
	}
	eventService2 := localservice.Get(channelID2)
	if eventService2 == nil {
		t.Fatalf("Expecting local event service for %s to be registered but got nil", channelID2)
	}
	reg1, bEventCh1, err := eventService1.RegisterBlockEvent()
	if err != nil {
		t.Fatalf("Error in RegisterBlockEvent on event service channel %s: %s", channelID1, err)
	}
	defer eventService1.Unregister(reg1)

	reg2, bEventCh2, err := eventService2.RegisterBlockEvent()
	if err != nil {
		t.Fatalf("Error in RegisterBlockEvent on event service channel %s: %s", channelID2, err)
	}
	defer eventService2.Unregister(reg2)

	// ehMx.RLock()
	// mockEventHubs[channelID1].ProduceEvent(mockevent.NewBlockEvent(channelID1))
	// mockEventHubs[channelID2].ProduceEvent(mockevent.NewBlockEvent(channelID2))
	// ehMx.RUnlock()

	numExpected := 2
	numReceived := 0
	done := false

	for !done {
		select {
		case event, ok := <-bEventCh1:
			if !ok {
				t.Fatalf("event channel1 disconnected")
			}
			fmt.Printf("*** Received event on bEventCh1: %v\n", event)
			numReceived++
		case event, ok := <-bEventCh2:
			if !ok {
				t.Fatalf("event channel2 disconnected")
			}
			fmt.Printf("*** Received event on bEventCh2: %s\n", event)
			numReceived++
		case <-time.After(2 * time.Second):
			if numReceived != numExpected {
				t.Fatalf("Expecting %d events but received %d", numExpected, numReceived)
			} else {
				done = true
			}
		}
	}
}

// func newMockConfig(channelID string, configPath string, option string) (*config.EventSnapConfig, error) {

// 	esconfig := &config.EventSnapConfig{
// 		MSPID:           "Org1MSP",
// 		EventHubAddress: "localhost:9053",
// 	}

// 	fileName := configPath + "/config" + channelID + ".yaml"
// 	sdkConfigBytes, err := ioutil.ReadFile(fileName)
// 	if err != nil {
// 		panic(fmt.Sprintf("Got error reading config file [%s]: %s", fileName, err))
// 	}
// 	esconfig.Bytes = sdkConfigBytes

// 	switch option {
// 	case "emptyAddress":
// 		esconfig.EventHubAddress = ""
// 		esconfig.ChannelConfigLoaded = true

// 	default:
// 		if esconfig != nil && channelID != "" {
// 			esconfig.ChannelConfigLoaded = true
// 			esconfig.EventHubRegTimeout = time.Duration(1 * time.Second)
// 			esconfig.EventHubRegTimeout = time.Duration(1 * time.Second)
// 			esconfig.EventDispatcherBufferSize = uint(100)
// 			esconfig.EventConsumerBufferSize = uint(100)
// 			esconfig.EventConsumerTimeout = time.Duration(10 * time.Millisecond)
// 		}
// 	}

// 	return esconfig, nil
// }
