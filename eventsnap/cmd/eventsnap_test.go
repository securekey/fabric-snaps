/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	configmocks "github.com/securekey/fabric-snaps/configmanager/pkg/mocks"
	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/config"
	discoveryService "github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	txnConfig "github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
)

const (
	TxnSnapAppName       = "txnsnap"
	channelID1           = "testchannel"
	channelID2           = "testChannel2"
	mspID                = "Org1MSP"
	peerID               = "peer1"
	eventSvcExistsErrMsg = "Event service already initialized for channel"
	testhost             = "127.0.0.1"
	testport             = 7040
	testBroadcastPort    = 7041
)

type sampleConfig struct {
	api.Config
}

type MockProviderFactory struct {
	defsvc.ProviderFactory
}

func (m *MockProviderFactory) CreateDiscoveryProvider(config coreApi.Config, fabPvdr fabApi.InfraProvider) (fabApi.DiscoveryProvider, error) {
	return &impl{clientConfig: config}, nil
}

type impl struct {
	clientConfig coreApi.Config
}

// CreateDiscoveryService return impl of DiscoveryService
func (p *impl) CreateDiscoveryService(channelID string) (fabApi.DiscoveryService, error) {
	memService := membership.NewServiceWithMocks([]byte("Org1MSP"), "internalhost1:1000", mockbcinfo.ChannelBCInfos(mockbcinfo.NewChannelBCInfo(channelID, mockbcinfo.BCInfo(uint64(1000)))))
	return discoveryService.New(channelID, p.clientConfig, memService), nil
}

func TestEventSnap(t *testing.T) {

	configStub1 := configmocks.NewMockStub(channelID1)
	configStub1.ChannelID = channelID1
	service.Initialize(configStub1, mspID)

	configStub2 := configmocks.NewMockStub(channelID2)
	configStub2.ChannelID = channelID2
	service.Initialize(configStub2, mspID)

	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, config.EventSnapAppName, "./sampleconfig/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub2, mspID, peerID, config.EventSnapAppName, "./sampleconfig/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, TxnSnapAppName, "./sampleconfig/txnsnap/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub2, mspID, peerID, TxnSnapAppName, "./sampleconfig/txnsnap/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}

	eventsnap := &eventSnap{
		peerConfigPath: "./sampleconfig",
	}

	stub := shim.NewMockStub("eventsnap", eventsnap)

	// Start mock event hub
	eventServer, err := fcmocks.StartMockEventServer("127.0.0.1:7051")
	if err != nil {
		t.Fatalf("Failed to start mock event hub: %v", err)
	}
	defer eventServer.Stop()

	mockEndorserServer := mocks.StartEndorserServer(testhost + ":" + strconv.Itoa(testport))
	payloadMap := make(map[string][]byte, 2)
	payloadMap["GetConfigBlock"] = getConfigBlockPayload()
	payloadMap["default"] = []byte("value")
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil, MockMSP: "Org1MSP", Status: 200,
		Payload: payloadMap})

	channels := []string{channelID1, channelID2}
	for _, channel := range channels {

		txSnapConfig, err := txnConfig.NewConfig("./sampleconfig/txnsnap/", channel)
		if err != nil {
			panic(fmt.Sprintf("Error initializing config: %s", err))
		}
		_, err = client.GetInstance(channel, &sampleConfig{txSnapConfig}, &MockProviderFactory{})
		if err != nil {
			panic(fmt.Sprintf("Client GetInstance return error %v", err))
		}
	}

	// Happy Path
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid2", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Initialize again with same channel
	if resp := stub.MockInit("txid3", nil); resp.Status == shim.OK || !strings.Contains(resp.GetMessage(), eventSvcExistsErrMsg) {
		t.Fatalf("Expected '%s', but got '%s'", eventSvcExistsErrMsg, resp.GetMessage())
	}

	// Another channel
	stub.ChannelID = channelID2
	if resp := stub.MockInit("txid4", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Init again on same channel
	if resp := stub.MockInit("txid5", nil); resp.Status == shim.OK || !strings.Contains(resp.GetMessage(), eventSvcExistsErrMsg) {
		t.Fatalf("Expected '%s', but got '%s'", eventSvcExistsErrMsg, resp.GetMessage())
	}

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
			RootCA:         mocks.RootCA,
		},
		Index:           0,
		LastConfigIndex: 0,
	}

	payload, _ := proto.Marshal(builder.Build())

	return payload
}
