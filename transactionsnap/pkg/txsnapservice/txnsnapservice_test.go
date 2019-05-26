/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txsnapservice

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	sdkconfig "github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite"
	servicemocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/mocks"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/metrics/pkg/util"
	metricsutil "github.com/securekey/fabric-snaps/metrics/pkg/util"
	eventserviceMocks "github.com/securekey/fabric-snaps/mocks/event/mockservice/eventservice"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestEndorseTransactionWithTxID(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil, []byte("nonce"), "")
	txService := newMockTxService(nil)
	mockEndorserServer.GetMockPeer().KVWrite = false

	resp, err := txService.EndorseTransaction(&snapTxReq, nil)
	require.NoError(t, err)
	require.Equal(t, resp.TxValidationCode, pb.TxValidationCode_BAD_PROPOSAL_TXID)

	snapTxReq.Nonce = []byte("")
	snapTxReq.TransactionID = "test"
	resp, err = txService.EndorseTransaction(&snapTxReq, nil)
	require.NoError(t, err)
	require.Equal(t, resp.TxValidationCode, pb.TxValidationCode_BAD_PROPOSAL_TXID)

	// test with wrong txID
	snapTxReq.TransactionID = "test"
	snapTxReq.Nonce = []byte("nonce")

	snapTxReq = createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil, nil, "")
	response, err := txService.EndorseTransaction(&snapTxReq, nil)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotEqual(t, len(response.Responses), 0)
	require.Equal(t, int(response.Responses[0].ProposalResponse.Response.Status), 200)
	require.Equal(t, string(response.Responses[0].ProposalResponse.Response.Payload), "value")

}

func TestCommitTransaction(t *testing.T) {
	// commit with kvwrite false
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil, nil, "")
	txService := newMockTxService(nil)
	mockEndorserServer.GetMockPeer().KVWrite = false

	_, commit, err := txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if commit {
		t.Fatalf("commit value should be false")
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

	_, commit, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %s", err)
	}
	if !commit {
		t.Fatalf("commit value should be true")
	}

}

func TestCommitTransactionWithTxID(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil, []byte("nonce"), "")
	txService := newMockTxService(nil)
	mockEndorserServer.GetMockPeer().KVWrite = false

	resp, _, err := txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_BAD_PROPOSAL_TXID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_BAD_PROPOSAL_TXID)
	}

	snapTxReq.Nonce = []byte("")
	snapTxReq.TransactionID = "test"
	resp, _, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_BAD_PROPOSAL_TXID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_BAD_PROPOSAL_TXID)
	}

	// test with wrong txID
	snapTxReq.TransactionID = "test"
	snapTxReq.Nonce = []byte("nonce")
	resp, _, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_BAD_PROPOSAL_TXID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_BAD_PROPOSAL_TXID)
	}

	sdk, e := fabsdk.New(sdkconfig.FromFile("../../cmd/sampleconfig/config.yaml"))
	require.NoError(t, e)
	defer sdk.Close()

	ctx, e := sdk.Context(fabsdk.WithUser("Txn-Snap-User"), fabsdk.WithOrg("peerorg1"))()
	require.NoError(t, e)

	// test with correct txID
	creator, err1 := ctx.Serialize()
	if err1 != nil {
		t.Fatalf("Error fcClient.GetContext().Serialize() %v", err1)
	}
	ho := cryptosuite.GetSHA256Opts()
	h, err1 := ctx.CryptoSuite().GetHash(ho)
	if err1 != nil {
		t.Fatalf("Error fcClient.GetContext().CryptoSuite().GetHash %v", err1)
	}

	snapTxReq.TransactionID, err1 = computeTxnID(snapTxReq.Nonce, creator, h)
	if err1 != nil {
		t.Fatalf("Error computeTxnID %v", err1)
	}
	fmt.Printf("****** Creator [%s], TxnID: [%s]\n", creator, snapTxReq.TransactionID)

	resp, _, err = txService.CommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error commit transaction %v", err)
	}
	if resp.TxValidationCode != pb.TxValidationCode_VALID {
		t.Fatalf("resp.TxValidationCode not equal to %v", pb.TxValidationCode_VALID)
	}
}

func TestCommitOnlyTransaction(t *testing.T) {
	nonce := []byte("nonce")
	txId := createTxId(t, nonce)

	// first endorse transaction
	mockEndorserServer.GetMockPeer().KVWrite = true
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil, nonce, txId)
	txService := newMockTxService(nil)

	response, err := txService.EndorseTransaction(&snapTxReq, nil)

	assert.Nil(t, err, fmt.Sprintf("Error endorsing transaction %s", err))
	assert.NotNil(t, response, "Expected proposal response")
	assert.True(t, len(response.Responses) > 0, "Received an empty transaction proposal response")
	assert.Equal(t, int32(200), response.Responses[0].ProposalResponse.Response.Status,
		"Expected proposal response status: SUCCESS")
	assert.Equal(t, "value", string(response.Responses[0].ProposalResponse.Response.Payload),
		fmt.Sprintf("Expected proposal response payload: value but got %v",
			string(response.Responses[0].ProposalResponse.Response.Payload)))

	// commit - send the endorse response in this request
	snapTxReq = createTransactionSnapRequest("committransaction", "ccid", channelID, true, nil, nonce, txId)
	txService = newMockTxService(func(response invoke.Response) error {
		go func() {
			time.Sleep(2 * time.Second)
			eventProducer.Ledger().NewFilteredBlock(
				channelID,
				servicemocks.NewFilteredTx(txId, pb.TxValidationCode_VALID),
			)
		}()
		return nil
	})

	invokeResponse := &invoke.Response{
		TransactionID:    fab.TransactionID(snapTxReq.TransactionID),
		Responses:        response.Responses,
		Payload:          response.Payload,
		ChaincodeStatus:  response.ChaincodeStatus,
		Proposal:         response.Proposal,
		TxValidationCode: response.TxValidationCode,
	}
	_, commit, err := txService.CommitOnlyTransaction(&snapTxReq, invokeResponse, nil)
	assert.Nil(t, err, fmt.Sprintf("Error commit transaction %v", err))
	assert.True(t, commit, "commit value should be true")
}

func TestCommitOnlyTransactionForNoWriteSet(t *testing.T) {
	nonce := []byte("nonce")
	txId := createTxId(t, nonce)

	// first endorse transaction
	mockEndorserServer.GetMockPeer().KVWrite = false
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil, nonce, txId)
	txService := newMockTxService(nil)

	response, err := txService.EndorseTransaction(&snapTxReq, nil)

	assert.Nil(t, err, fmt.Sprintf("Error endorsing transaction %s", err))
	assert.NotNil(t, response, "Expected proposal response")
	assert.True(t, len(response.Responses) > 0, "Received an empty transaction proposal response")
	assert.Equal(t, int32(200), response.Responses[0].ProposalResponse.Response.Status,
		"Expected proposal response status: SUCCESS")
	assert.Equal(t, "value", string(response.Responses[0].ProposalResponse.Response.Payload),
		fmt.Sprintf("Expected proposal response payload: value but got %v",
			string(response.Responses[0].ProposalResponse.Response.Payload)))

	// commit
	snapTxReq = createTransactionSnapRequest("committransaction", "ccid", channelID, true, nil, nonce, txId)
	txService = newMockTxService(func(response invoke.Response) error {
		go func() {
			time.Sleep(2 * time.Second)
			eventProducer.Ledger().NewFilteredBlock(
				channelID,
				servicemocks.NewFilteredTx(txId, pb.TxValidationCode_VALID),
			)
		}()
		return nil
	})

	invokeResponse := &invoke.Response{
		TransactionID:    fab.TransactionID(snapTxReq.TransactionID),
		Responses:        response.Responses,
		Payload:          response.Payload,
		ChaincodeStatus:  response.ChaincodeStatus,
		Proposal:         response.Proposal,
		TxValidationCode: response.TxValidationCode,
	}
	_, commit, err := txService.CommitOnlyTransaction(&snapTxReq, invokeResponse, nil)
	assert.Nil(t, err, fmt.Sprintf("Error commit transaction %v", err))
	assert.False(t, commit, "commit value should be false")
}

func TestVerifyEndorsements(t *testing.T) {
	var tpr []*fab.TransactionProposalResponse
	mockEndorserServer.GetMockPeer().KVWrite = true
	txService := newMockTxService(nil)
	for i := 1; i <= 2; i++ {
		nonce := []byte("nonce")
		txId := createTxId(t, nonce)

		// first endorse transaction
		snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil, nonce, txId)

		response, endorseErr := txService.EndorseTransaction(&snapTxReq, nil)
		require.NoError(t, endorseErr)
		require.NotNil(t, response, "Expected proposal response")
		require.Equal(t, len(response.Responses), 1)
		tpr = append(tpr, response.Responses[0])
	}

	t.Run("test success", func(t *testing.T) {
		var proposalResponses []*pb.ProposalResponse
		proposalResponses = append(proposalResponses, tpr[0].ProposalResponse)

		endorsements, err := json.Marshal(proposalResponses)
		require.NoError(t, err)
		err = txService.VerifyEndorsements(endorsements)
		require.NoError(t, err)
	})

	t.Run("test wrong endorsements signature", func(t *testing.T) {
		var proposalResponses []*pb.ProposalResponse
		proposalResponses = append(proposalResponses, tpr[0].ProposalResponse)

		proposalResponses[0].Endorsement.Signature = []byte("wrongSignature")
		endorsements, err := json.Marshal(proposalResponses)
		require.NoError(t, err)

		err = txService.VerifyEndorsements(endorsements)
		require.Error(t, err)
		require.Contains(t, err.Error(), "signature validation failed")
	})

	t.Run("test endorsements with different response", func(t *testing.T) {
		var proposalResponses []*pb.ProposalResponse
		proposalResponses = append(proposalResponses, tpr[0].ProposalResponse)
		tpr[1].ProposalResponse.Response.Payload = []byte("wrongPayload")
		proposalResponses = append(proposalResponses, tpr[1].ProposalResponse)

		endorsements, err := json.Marshal(proposalResponses)
		require.NoError(t, err)

		err = txService.VerifyEndorsements(endorsements)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ProposalResponsePayloads do not match")

	})

	t.Run("test endorsements with 500 status", func(t *testing.T) {
		var proposalResponses []*pb.ProposalResponse
		proposalResponses = append(proposalResponses, tpr[0].ProposalResponse)
		tpr[1].ProposalResponse.Response.Status = 500
		proposalResponses = append(proposalResponses, tpr[1].ProposalResponse)

		endorsements, err := json.Marshal(proposalResponses)
		require.NoError(t, err)

		err = txService.VerifyEndorsements(endorsements)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Endorser Server Status Code: (500) INTERNAL_SERVER_ERROR")

	})

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
	if err := util.InitializeMetricsProvider("../../cmd/sampleconfig"); err != nil {
		panic(err)
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
			panic(fmt.Sprintf("error creating channel eventservice client: %s", err))
		}
	}

	client.ServiceProviderFactory = &mocks.MockProviderFactory{EventService: eventService}
	client.CfgProvider = func(channelID string) (api.Config, error) { return &sampleConfig{txSnapConfig}, nil }
	fcClient, err = client.GetInstance(channelID, client.NewMetrics(metricsutil.GetMetricsInstance()))
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

func createTxId(t *testing.T, nonce []byte) string {
	sdk, e := fabsdk.New(sdkconfig.FromFile("../../cmd/sampleconfig/config.yaml"))
	require.NoError(t, e)
	defer sdk.Close()

	ctx, e := sdk.Context(fabsdk.WithUser("Txn-Snap-User"), fabsdk.WithOrg("peerorg1"))()
	require.NoError(t, e)

	// test with correct txID
	creator, err1 := ctx.Serialize()
	assert.Nil(t, err1, fmt.Sprintf("Error fcClient.GetContext().Serialize(): %v", err1))

	ho := cryptosuite.GetSHA256Opts()
	h, err1 := ctx.CryptoSuite().GetHash(ho)
	assert.Nil(t, err1, fmt.Sprintf("Error while computing hash: %v", err1))

	txId, err := computeTxnID(nonce, creator, h)
	assert.Nil(t, err, fmt.Sprintf("Error while computing a tranxId: %v", err))

	return txId
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
