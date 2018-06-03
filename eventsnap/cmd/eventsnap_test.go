/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	configmanagerapi "github.com/securekey/fabric-snaps/configmanager/api"
	configmocks "github.com/securekey/fabric-snaps/configmanager/pkg/mocks"
	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/config"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/mocks"
	"github.com/securekey/fabric-snaps/mocks/mockprovider"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	transactionsnapMocks "github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
)

const (
	TxnSnapAppName    = "txnsnap"
	channelID1        = "testchannel"
	channelID2        = "testChannel2"
	mspID             = "Org1MSP"
	peerID            = "peer1"
	testhost          = "127.0.0.1"
	testport          = 7040
	testBroadcastPort = 7041
)

type sampleConfig struct {
	api.Config
}

func TestEventSnap(t *testing.T) {

	delayStartChannelEventsDuration = 0 * time.Second
	configStub1 := configmocks.NewMockStub(channelID1)
	configStub1.ChannelID = channelID1
	service.Initialize(configStub1, mspID)

	configStub2 := configmocks.NewMockStub(channelID2)
	configStub2.ChannelID = channelID2
	service.Initialize(configStub2, mspID)

	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, config.EventSnapAppName, configmanagerapi.VERSION, "./sampleconfig/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub2, mspID, peerID, config.EventSnapAppName, configmanagerapi.VERSION, "./sampleconfig/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, TxnSnapAppName, configmanagerapi.VERSION, "./sampleconfig/txnsnap/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub2, mspID, peerID, TxnSnapAppName, configmanagerapi.VERSION, "./sampleconfig/txnsnap/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}

	eventsnap := &eventSnap{
		peerConfigPath: "./sampleconfig",
	}

	stub := shim.NewMockStub("eventsnap", eventsnap)

	// Start mock deliver server
	deliverServer, err := mocks.StartMockDeliverServer("127.0.0.1:7040")
	if err != nil {
		t.Fatalf("Failed to start mock event hub: %s", err)
	}
	defer deliverServer.Stop()

	client.ServiceProviderFactory = &mockprovider.Factory{}
	// Happy Path
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid2", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Another channel
	stub.ChannelID = channelID2
	if resp := stub.MockInit("txid4", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}
	time.Sleep(2 * time.Second)

	eventService1 := localservice.Get(channelID1)
	if eventService1 == nil {
		t.Fatalf("Expecting local event service for %s to be registered but got nil", channelID1)
	}
	eventService2 := localservice.Get(channelID2)
	if eventService2 == nil {
		t.Fatalf("Expecting local event service for %s to be registered but got nil", channelID2)
	}

	//TODO :  event service currently doesn't support block event (commenting below tests)

	//reg1, bEventCh1, err := eventService1.RegisterBlockEvent()
	//if err != nil {
	//	t.Fatalf("Error in RegisterBlockEvent on event service channel %s: %s", channelID1, err)
	//}
	//defer eventService1.Unregister(reg1)
	//
	//reg2, bEventCh2, err := eventService2.RegisterBlockEvent()
	//if err != nil {
	//	t.Fatalf("Error in RegisterBlockEvent on event service channel %s: %s", channelID2, err)
	//}
	//defer eventService2.Unregister(reg2)
	//
	//// ehMx.RLock()
	//// mockEventHubs[channelID1].ProduceEvent(mockevent.NewBlockEvent(channelID1))
	//// mockEventHubs[channelID2].ProduceEvent(mockevent.NewBlockEvent(channelID2))
	//// ehMx.RUnlock()
	//
	//numExpected := 2
	//numReceived := 0
	//done := false
	//
	//for !done {
	//	select {
	//	case event, ok := <-bEventCh1:
	//		if !ok {
	//			t.Fatalf("event channel1 disconnected")
	//		}
	//		fmt.Printf("*** Received event on bEventCh1: %v\n", event)
	//		numReceived++
	//	case event, ok := <-bEventCh2:
	//		if !ok {
	//			t.Fatalf("event channel2 disconnected")
	//		}
	//		fmt.Printf("*** Received event on bEventCh2: %s\n", event)
	//		numReceived++
	//	case <-time.After(2 * time.Second):
	//		if numReceived != numExpected {
	//			t.Fatalf("Expecting %d events but received %d", numExpected, numReceived)
	//		} else {
	//			done = true
	//		}
	//	}
	//}
}

func TestMain(m *testing.M) {

	txsnapservice.PeerConfigPath = "./sampleconfig"

	opts := &factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &factory.FileKeystoreOpts{KeyStorePath: "./sampleconfig/msp/keystore"},
		},
	}
	factory.InitFactories(opts)

	os.Exit(m.Run())
}

func getConfigBlockPayload() []byte {
	// create config block builder in order to create valid payload
	builder := &fcmocks.MockConfigBlockBuilder{
		MockConfigGroupBuilder: fcmocks.MockConfigGroupBuilder{
			ModPolicy: "Admins",
			MSPNames: []string{
				"Org1MSP",
			},
			OrdererAddress: fmt.Sprintf("grpc://%s:%d", testhost, testBroadcastPort),
			RootCA:         transactionsnapMocks.RootCA,
		},
		Index:           0,
		LastConfigIndex: 0,
	}

	payload, _ := proto.Marshal(builder.Build())

	return payload
}
