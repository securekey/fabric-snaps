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
	"testing"
	"time"

	"encoding/hex"
	"hash"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite"
	servicemocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/mocks"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	eventserviceMocks "github.com/securekey/fabric-snaps/eventservice/pkg/mocks"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
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
)

type sampleConfig struct {
	api.Config
}

func TestEndorseTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil, nil, "")
	txService := newMockTxService(nil)

	response, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %s", err)
	}
	if response == nil {
		t.Fatal("Expected proposal response")
	}

	if len(response.Responses) == 0 {
		t.Fatal("Received an empty transaction proposal response")
	}
	if response.Responses[0].ProposalResponse.Response.Status != 200 {
		t.Fatal("Expected proposal response status: SUCCESS")
	}
	if string(response.Responses[0].ProposalResponse.Response.Payload) != "value" {
		t.Fatalf("Expected proposal response payload: value but got %v", string(response.Responses[0].ProposalResponse.Response.Payload))
	}

}

func TestCommitTransaction(t *testing.T) {
	// commit with kvwrite false
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil, nil, "")
	txService := newMockTxService(nil)
	mockEndorserServer.GetMockPeer().KVWrite = false

	_, err := txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}

	// commit with kvwrite true
	mockEndorserServer.GetMockPeer().KVWrite = true

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
		t.Fatalf("Error commit transaction %s", err)
	}

}

func TestCommitTransactionWithTxID(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil, []byte("nonce"), "")
	txService := newMockTxService(nil)
	mockEndorserServer.GetMockPeer().KVWrite = false

	resp, err := txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_BAD_PROPOSAL_TXID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_BAD_PROPOSAL_TXID)
	}

	snapTxReq.Nonce = []byte("")
	snapTxReq.TransactionID = "test"
	resp, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_BAD_PROPOSAL_TXID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_BAD_PROPOSAL_TXID)
	}

	// test with wrong txID
	snapTxReq.TransactionID = "test"
	snapTxReq.Nonce = []byte("nonce")
	resp, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_BAD_PROPOSAL_TXID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_BAD_PROPOSAL_TXID)
	}

	// test with correct txID
	creator, err1 := fcClient.GetContext().Serialize()
	if err1 != nil {
		t.Fatalf("Error fcClient.GetContext().Serialize() %v", err1)
	}
	ho := cryptosuite.GetSHA256Opts()
	h, err1 := fcClient.GetContext().CryptoSuite().GetHash(ho)
	if err1 != nil {
		t.Fatalf("Error fcClient.GetContext().CryptoSuite().GetHash %v", err1)
	}

	snapTxReq.TransactionID, err1 = computeTxnID(snapTxReq.Nonce, creator, h)
	if err1 != nil {
		t.Fatalf("Error computeTxnID %v", err1)
	}
	resp, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_VALID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_VALID)
	}
}

func computeTxnID(nonce, creator []byte, h hash.Hash) (string, error) {
	logger.Debugf("computeTxnID nonce %s creator %s", nonce, creator)
	b := append(nonce, creator...)

	_, err := h.Write(b)
	if err != nil {
		return "", err
	}
	digest := h.Sum(nil)
	id := hex.EncodeToString(digest)

	return id, nil
}

func TestVerifyProposalSignature(t *testing.T) {
	txService := newMockTxService(nil)
	err := txService.VerifyTxnProposalSignature(nil)
	if err == nil {
		t.Fatal("SignedProposal is mandatory field")
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
	// TODO
	bccspFactory.InitFactories(opts)

	os.Setenv("CORE_PEER_ADDRESS", "peer1:5100")
	defer os.Unsetenv("CORE_PEER_ADDRESS")

	os.Setenv("CORE_TXNSNAP_RETRY_ATTEMPTS", "1")
	defer os.Unsetenv("CORE_TXNSNAP_RETRY_ATTEMPTS")

	configData, err := ioutil.ReadFile("../../cmd/sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	configMsg := &configmanagerApi.ConfigMessage{MspID: mspID,
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "jdoe", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "txnsnap", Version: configmanagerApi.VERSION, Config: string(configData)}}}}}
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
	payloadMap := make(map[string][]byte, 2)
	payloadMap["GetConfigBlock"] = getConfigBlockPayload()
	payloadMap["default"] = []byte("value")
	mockEndorserServer.SetMockPeer(&mocks.MockPeer{MockName: "Peer1", MockURL: "http://peer1.com", MockRoles: []string{}, MockCert: nil, MockMSP: "Org1MSP", Status: 200,
		Payload: payloadMap})

	var eventService fab.EventService
	if eventProducer == nil {
		var err error
		eventService, eventProducer, err = eventserviceMocks.NewServiceWithMockProducer([]options.Opt{}, eventserviceMocks.WithFilteredBlockLedger())
		if err != nil {
			panic(fmt.Sprintf("error creating channel event client: %s", err))
		}
	}

	client.ServiceProviderFactory = &mocks.MockProviderFactory{EventService: eventService}
	fcClient, err = client.GetInstance(channelID, &sampleConfig{txSnapConfig})
	if err != nil {
		panic(fmt.Sprintf("Client GetInstance return error %s", err))
	}

	mockBroadcastServer := &fcmocks.MockBroadcastServer{}
	mockBroadcastServer.Start(fmt.Sprintf("%s:%d", testhost, testBroadcastPort))
	defer mockBroadcastServer.Stop()

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

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, registerTxEvent bool, peerFilter *api.PeerFilterOpts, nonce []byte, txID string) api.SnapTransactionRequest {

	transientMap := make(map[string][]byte)
	transientMap["key"] = []byte("transientvalue")
	endorserArgs := make([][]byte, 5)
	endorserArgs[0] = []byte("invoke")
	endorserArgs[1] = []byte("move")
	endorserArgs[2] = []byte("a")
	endorserArgs[3] = []byte("b")
	endorserArgs[4] = []byte("1")
	snapTxReq := api.SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        transientMap,
		EndorserArgs:        endorserArgs,
		CCIDsForEndorsement: nil,
		RegisterTxEvent:     registerTxEvent,
		PeerFilter:          peerFilter,
		Nonce:               nonce,
		TransactionID:       txID,
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
