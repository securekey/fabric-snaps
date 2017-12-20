/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	eventrelay "github.com/securekey/fabric-snaps/eventrelay/pkg/relay"
	localservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/config"
	"github.com/securekey/fabric-snaps/mocks/event/mockevent"
	"github.com/securekey/fabric-snaps/mocks/event/mockeventhub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type mockConfigProvider struct {
	configs map[string]*config.EventSnapConfig
	mutex   sync.RWMutex
}

func newMockConfigProvider() *mockConfigProvider {
	return &mockConfigProvider{configs: make(map[string]*config.EventSnapConfig)}
}

func (cfgprovider *mockConfigProvider) setConfig(channelID string, cfg *config.EventSnapConfig) {
	cfgprovider.mutex.Lock()
	defer cfgprovider.mutex.Unlock()
	cfgprovider.configs[channelID] = cfg
}

func (cfgprovider *mockConfigProvider) GetConfig(channelID string) (*config.EventSnapConfig, error) {
	cfgprovider.mutex.RLock()
	defer cfgprovider.mutex.RUnlock()
	return cfgprovider.configs[channelID], nil
}

func TestEventSnap(t *testing.T) {
	ehMx := &sync.RWMutex{}
	channelID1 := "ch1"
	channelID2 := "ch2"

	config0, err := newMockConfig("", "./sampleconfig", "emptyAddress")
	if err != nil {
		fmt.Printf("Error %v", err)
	}
	fmt.Printf("Config 0 %v", config0)
	config1, err := newMockConfig(channelID1, "./sampleconfig", "")
	if err != nil {
		fmt.Printf("Error %v", err)
	}
	fmt.Printf("Config 1 %v\n", config1)
	config2, err := newMockConfig(channelID2, "./sampleconfig", "ch2")
	if err != nil {
		fmt.Printf("Error %v", err)
	}

	configProvider := newMockConfigProvider()
	configProvider.setConfig("ch1", config0)
	eventsnap := &eventSnap{
		pserver:        grpc.NewServer(),
		configProvider: configProvider,
	}

	stub := shim.NewMockStub("eventsnap", eventsnap)
	// Invalid options
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid2", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in init since no event hub address was specified but got OK")
	}

	mockEventHubs := make(map[string]*mockeventhub.MockEventHub)
	configProvider.setConfig("", config0)
	eventsnap = &eventSnap{
		pserver: grpc.NewServer(),
		eropts: eventrelay.MockOpts(func(channelID string, address string, regTimeout time.Duration, adapter eventrelay.EventAdapter, tlsCredentials credentials.TransportCredentials) (eventrelay.EventHub, error) {
			fmt.Printf("Creating mock event hub for channel %s\n", channelID)
			mockeh := mockeventhub.New(adapter)
			ehMx.Lock()
			mockEventHubs[channelID] = mockeh
			ehMx.Unlock()
			return mockeh, nil
		}),
		configProvider: configProvider,
	}

	stub = shim.NewMockStub("eventsnap", eventsnap)

	// Initialize with no channel
	if resp := stub.MockInit("txid1", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Initialize with channel
	stub.ChannelID = channelID1
	configProvider.setConfig("ch1", config1)
	if resp := stub.MockInit("txid3", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Another channel
	stub.ChannelID = channelID2
	configProvider.setConfig("ch2", config2)
	if resp := stub.MockInit("txid4", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Delay adding the configuration
	time.Sleep(8 * time.Second)

	configProvider.setConfig(channelID1, config1)
	configProvider.setConfig(channelID2, config1)

	// Wait for the event snap to pick up the configuration
	time.Sleep(6 * time.Second)

	// Init again on same channelemptyAddress
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

	ehMx.RLock()
	mockEventHubs[channelID1].ProduceEvent(mockevent.NewBlockEvent(channelID1))
	mockEventHubs[channelID2].ProduceEvent(mockevent.NewBlockEvent(channelID2))
	ehMx.RUnlock()

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

func newMockConfig(channelID string, configPath string, option string) (*config.EventSnapConfig, error) {

	peerCfg, err := config.New("", configPath)
	if err != nil {
		return nil, err
	}

	switch option {
	case "emptyAddress":
		peerCfg.EventHubAddress = ""
		peerCfg.ChannelConfigLoaded = true
	case "ch2":
		peerCfg.ChannelConfigLoaded = true
		peerCfg.EventHubRegTimeout = time.Duration(2 * time.Second)
		peerCfg.EventHubRegTimeout = time.Duration(2 * time.Second)
		peerCfg.EventRelayTimeout = time.Duration(2 * time.Second)
		peerCfg.EventDispatcherBufferSize = uint(200)
		peerCfg.EventConsumerBufferSize = uint(200)
		peerCfg.EventConsumerTimeout = time.Duration(20 * time.Millisecond)
	default:
		if peerCfg != nil && channelID != "" {
			peerCfg.ChannelConfigLoaded = true
			peerCfg.EventHubRegTimeout = time.Duration(1 * time.Second)
			peerCfg.EventHubRegTimeout = time.Duration(1 * time.Second)
			peerCfg.EventRelayTimeout = time.Duration(1 * time.Second)
			peerCfg.EventDispatcherBufferSize = uint(100)
			peerCfg.EventConsumerBufferSize = uint(100)
			peerCfg.EventConsumerTimeout = time.Duration(10 * time.Millisecond)
		}
	}

	return peerCfg, nil
}
