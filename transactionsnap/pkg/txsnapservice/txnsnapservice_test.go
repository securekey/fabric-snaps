/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txsnapservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	txnmocks "github.com/hyperledger/fabric-sdk-go/pkg/client/common/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	servicemocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/mocks"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/errors/status"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	eventserviceMocks "github.com/securekey/fabric-snaps/eventservice/pkg/mocks"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
	"google.golang.org/grpc"
)

var channelID = "testChannel"
var mspID = "Org1MSP"
var mockEndorserServer *mocks.MockEndorserServer
var mockBroadcastServer *fcmocks.MockBroadcastServer
var eventProducer *servicemocks.MockProducer

var testhost = "127.0.0.1"
var testport = 7040
var testBroadcastPort = 7041
var txSnapConfig api.Config
var fcClient api.Client

const (
	org1 = "Org1MSP"
	org2 = "Org2MSP"
)

type sampleConfig struct {
	api.Config
}

type MockProviderFactory struct {
	defsvc.ProviderFactory
}

func (m *MockProviderFactory) CreateDiscoveryProvider(config coreApi.Config, fabPvdr fabApi.InfraProvider) (fabApi.DiscoveryProvider, error) {
	peer, _ := peer.New(fcmocks.NewMockConfig(), peer.WithURL("grpc://"+testhost+":"+strconv.Itoa(testport)))
	mdp, _ := txnmocks.NewMockDiscoveryProvider(nil, []fabApi.Peer{peer})
	return mdp, nil
}

func TestEndorseTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil)
	txService := newMockTxService(nil)
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil,
		MockMSP: "Org1MSP", Status: 200, Payload: []byte("value")})

	response, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if response == nil {
		t.Fatalf("Expected proposal response")
	}

	if len(response.Responses) == 0 {
		t.Fatalf("Received an empty transaction proposal response")
	}
	if response.Responses[0].ProposalResponse.Response.Status != 200 {
		t.Fatalf("Expected proposal response status: SUCCESS")
	}
	if string(response.Responses[0].ProposalResponse.Response.Payload) != "value" {
		t.Fatalf("Expected proposal response payload: value but got %v", string(response.Responses[0].ProposalResponse.Response.Payload))
	}

}

func TestEndorseTransactionWithPeerFilter(t *testing.T) {
	peerFilterOpts := &api.PeerFilterOpts{
		Type: api.MinBlockHeightPeerFilterType,
		Args: []string{channelID},
	}

	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, peerFilterOpts)
	txService := newMockTxService(nil)
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil,
		MockMSP: "Org1MSP", Status: 200, Payload: []byte("value")})
	_, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err == nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if !strings.Contains(err.Error(), status.NoPeersFound.String()) {
		t.Fatalf("Wrong error message")
	}

}

func TestCommitTransaction(t *testing.T) {
	// commit with kvwrite false
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil)
	txService := newMockTxService(nil)
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil,
		MockMSP: "Org1MSP", Status: 200, Payload: []byte("value"), KVWrite: false})

	_, err := txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}

	// commit with kvwrite true
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil,
		MockMSP: "Org1MSP", Status: 200, Payload: []byte("value"), KVWrite: true})

	txService = newMockTxService(func(response invoke.Response) error {
		go func() {
			time.Sleep(2 * time.Second)
			eventProducer.Ledger().NewFilteredBlock(
				channelID,
				servicemocks.NewFilteredTx(string(response.TransactionID), pb.TxValidationCode_VALID),
			)
		}()
		return nil
	})

	_, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}

}

func TestVerifyProposalSignature(t *testing.T) {
	txService := newMockTxService(nil)
	err := txService.VerifyTxnProposalSignature(nil)
	if err == nil {
		t.Fatalf("SignedProposal is mandatory field")
	}
}

func newMockTxService(callback api.EndorsedCallback) *TxServiceImpl {
	return &TxServiceImpl{
		Config:   txSnapConfig,
		FcClient: fcClient,
		Callback: callback,
	}
}

func TestMain(m *testing.M) {

	//Setup bccsp factory
	// note: use of 'pkcs11' tag in the unit test will load the PCKS11 version of the factory opts.
	// otherwise default SW version will be used.
	//opts := sampleconfig.GetSampleBCCSPFactoryOpts("../sampleconfig")
	// TODO: remove code between the TODOs and uncomment above line and investigate
	// why s390 build is failing at the call `client.GetInstance(channelID, &sampleConfig{txSnapConfig})`
	// at line 281 below
	path := "../../cmd/sampleconfig/msp/keystore"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(fmt.Sprintf("Wrong path: %v\n", err))
	}
	opts := &bccspFactory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &bccspFactory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &bccspFactory.FileKeystoreOpts{KeyStorePath: "../../cmd/sampleconfig/msp/keystore"},
		},
	}
	// TDOD
	bccspFactory.InitFactories(opts)

	os.Setenv("CORE_PEER_ADDRESS", testhost+":"+strconv.Itoa(testport))
	defer os.Unsetenv("CORE_PEER_ADDRESS")

	configData, err := ioutil.ReadFile("../../cmd/sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	configMsg := &configmanagerApi.ConfigMessage{MspID: mspID,
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "jdoe", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "txnsnap", Config: string(configData)}}}}}
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
	configmgmtService.Initialize(stub, mspID)

	PeerConfigPath = "../../cmd/sampleconfig"
	txSnapConfig, err = config.NewConfig("../../cmd/sampleconfig", channelID)
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}
	mockEndorserServer = mocks.StartEndorserServer(testhost + ":" + strconv.Itoa(testport))
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil, MockMSP: "Org1MSP", Status: 200,
		Payload: getConfigBlockPayload()})

	fcClient, err = client.GetInstance(channelID, &sampleConfig{txSnapConfig}, &MockProviderFactory{})
	if err != nil {
		panic(fmt.Sprintf("Client GetInstance return error %v", err))
	}

	mockBroadcastServer, _ = fcmocks.StartMockBroadcastServer(fmt.Sprintf("%s:%d", testhost, testBroadcastPort), grpc.NewServer())

	if eventProducer == nil {
		eventService, producer, err := eventserviceMocks.NewServiceWithMockProducer([]options.Opt{}, eventserviceMocks.WithFilteredBlockLedger())
		if err != nil {
			panic(fmt.Sprintf("error creating channel event client: %s", err))
		}
		localservice.Register(channelID, eventService)eventProducer = producer

	}

	txService := newMockTxService(nil)
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil)
	txService.EndorseTransaction(&snapTxReq, nil)
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

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, registerTxEvent bool, peerFilter *api.PeerFilterOpts) api.SnapTransactionRequest {

	transientMap := make(map[string][]byte)
	transientMap["key"] = []byte("transientvalue")
	endorserArgs := make([][]byte, 5)
	endorserArgs[0] = []byte("invoke")
	endorserArgs[1] = []byte("move")
	endorserArgs[2] = []byte("a")
	endorserArgs[3] = []byte("b")
	endorserArgs[4] = []byte("1")
	ccIDsForEndorsement := []string{chaincodeID, "additionalccid"}
	snapTxReq := api.SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        transientMap,
		EndorserArgs:        endorserArgs,
		CCIDsForEndorsement: ccIDsForEndorsement,
		RegisterTxEvent:     registerTxEvent,
		PeerFilter:          peerFilter,
	}

	return snapTxReq

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
