/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	servicemocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/mocks"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	pbsdk "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	protosUtils "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/utils"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	cb "github.com/hyperledger/fabric/protos/common"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	memApi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	eventserviceMocks "github.com/securekey/fabric-snaps/mocks/event/mockservice/eventservice"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
	"github.com/securekey/fabric-snaps/mocks/mockmembership"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/sampleconfig"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/initbcinfo"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/mocks"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mockEndorserServer *mocks.MockEndorserServer
var mockBroadcastServer *fcmocks.MockBroadcastServer
var eventProducer *servicemocks.MockProducer

var txSnapConfig api.Config

var testhost = "127.0.0.1"
var testport = 7040
var testBroadcastPort = 7041
var fcClient api.Client
var channelID = "testChannel"
var mspID = "Org1MSP"

type sampleConfig struct {
	api.Config
}

func TestTransactionSnapInit(t *testing.T) {
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	var args [][]byte
	response := stub.MockInit("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expecting response status %d but got %d", shim.OK, response.Status)
	}

	initialBlockHeight := uint64(1001)

	ledgerBCInfoProvider = mockbcinfo.NewProvider(
		mockbcinfo.NewChannelBCInfo(channelID, mockbcinfo.BCInfo(initialBlockHeight)),
	)

	stub.ChannelID = channelID
	response = stub.MockInit("TxID", args)
	require.Equalf(t, int32(shim.OK), response.Status, "Expecting response status %d but got %d", shim.OK, response.Status)

	var bcInfo *cb.BlockchainInfo
	for i := 0; i < 10; i++ {
		ok := false
		bcInfo, ok = initbcinfo.Get(channelID)
		if ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NotNil(t, bcInfo)
	assert.Equal(t, initialBlockHeight, bcInfo.Height)
}

func TestNotSupportedFunction(t *testing.T) {
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("notSupportedFunction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}

	errorMsg := "Function notSupportedFunction is not supported"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}

}

func TestNotSpecifiedChannel(t *testing.T) {
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	var funcs []string
	funcs = append(funcs, "endorseTransaction")
	funcs = append(funcs, "commitTransaction")
	funcs = append(funcs, "verifyTransactionProposalSignature")
	for _, value := range funcs {
		var args [][]byte
		if value == "verifyTransactionProposalSignature" {
			args = make([][]byte, 3)
			args[0] = []byte(value)
			args[1] = []byte("")
			args[2] = nil
		} else if value == "commitTransaction" {
			args = make([][]byte, 4)
			args[0] = []byte(value)
			args[1] = []byte("")
			args[2] = nil
			args[3] = nil
		} else {
			args = createTransactionSnapRequest(value, "ccid", "", false)
		}
		//invoke transaction snap
		response := stub.MockInvoke("TxID", args)

		if response.Status != shim.ERROR {
			t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
		}
		if response.Message == "" {
			t.Fatal("Expecting error due to an misconfigured endorsers args")
		}
	}
}

func TestNotSpecifiedChaincodeID(t *testing.T) {

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "ChaincodeID is mandatory field of the SnapTransactionRequest"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestSupportedFunctionWithoutRequest(t *testing.T) {

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)

	var funcs []string
	funcs = append(funcs, "endorse")
	funcs = append(funcs, "commit")
	for _, value := range funcs {
		var args [][]byte
		args = append(args, []byte(value+"Transaction"))
		response := stub.MockInvoke("TxID", args)
		if response.Status != shim.ERROR {
			t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
		}
		if response.Message == "" {
			t.Fatal("Expecting error 'ChaincodeID is mandatory field of the SnapTransactionRequest'")
		}
	}

	var args [][]byte
	args = args[:0]
	args = append(args, []byte("verifyTransactionProposalSignature"))
	response := stub.MockInvoke("TxID2", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	if response.Message == "" {
		t.Fatal("Expecting 'Expected args containing channelID'")
	}

}

func TestSupportedFunctionWithNilRequest(t *testing.T) {

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	var funcs []string
	funcs = append(funcs, "endorseTransaction")
	funcs = append(funcs, "commitTransaction")
	for _, value := range funcs {
		var args [][]byte
		args = append(args, []byte(value))
		args = append(args, nil)
		response := stub.MockInvoke("TxID", args)
		if response.Status != shim.ERROR {
			t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
		}
		errorMsg := "Cannot decode parameters from request to Snap Transaction Request: unexpected end of JSON input"
		if !strings.Contains(response.Message, errorMsg) {
			t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
		}
	}
}

func TestTransactionSnapInvokeFuncEndorseTransactionStatusSuccess(t *testing.T) {
	snap := newMockTxnSnap(nil)
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	if len(response.GetPayload()) == 0 {
		t.Fatal("Received an empty payload")
	}
	var chResponse *channel.Response
	err := json.Unmarshal(response.GetPayload(), &chResponse)
	if err != nil {
		t.Fatalf("Cannot unmarshal transaction proposal response %s", err)
	}
	if len(chResponse.Responses) == 0 {
		t.Fatalf("Received an empty transaction proposal response")
	}
	if chResponse.Responses[0].ProposalResponse.Response.Status != 200 {
		t.Fatal("Expected proposal response status: SUCCESS")
	}
	if string(chResponse.Responses[0].ProposalResponse.Response.Payload) != "value" {
		t.Fatalf("Expected proposal response payload: value but got %v", string(chResponse.Responses[0].ProposalResponse.Response.Payload))
	}

}

func resetMockPeer() {
	mockEndorserServer.GetMockPeer().Status = 200
	mockEndorserServer.GetMockPeer().Error = nil
}

func TestTransactionSnapInvokeFuncEndorseTransactionReturnError(t *testing.T) {
	snap := newMockTxnSnap(nil)
	mockEndorserServer.GetMockPeer().Status = 500
	mockEndorserServer.GetMockPeer().Error = fmt.Errorf("proposalError")
	defer resetMockPeer()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg := "proposalError"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestTransactionSnapInvokeFuncCommitTransactionSuccess(t *testing.T) {
	mockEndorserServer.GetMockPeer().KVWrite = true

	mockBroadcastServer := &fcmocks.MockBroadcastServer{}
	mockBroadcastServer.Start(fmt.Sprintf("%s:%d", testhost, testBroadcastPort))
	mockBroadcastServer.BroadcastInternalServerError = false
	defer mockBroadcastServer.Stop()

	snap := newMockTxnSnap(nil)

	stub := shim.NewMockStub("transactionsnap", snap)
	//invoke transaction snap with registerTxEvent  false
	args := createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", false)
	response := stub.MockInvoke("TxID2", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d (%s)", shim.OK, response.Status, response.Message)
	}

	snap = newMockTxnSnap(func(response invoke.Response) error {
		go func() {
			time.Sleep(2 * time.Second)
			eventProducer.Ledger().NewFilteredBlock(
				channelID,
				servicemocks.NewFilteredTx(string(response.TransactionID), pb.TxValidationCode_VALID),
			)
		}()
		return nil
	})
	stub = shim.NewMockStub("transactionsnap", snap)
	//invoke transaction snap with registerTxEvent  true
	args = createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", true)
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d (%s)", shim.OK, response.Status, response.Message)
	}
}

func TestTransactionSnapInvokeFuncEndorseAndCommitTransactionReturnError(t *testing.T) {
	mockEndorserServer.GetMockPeer().KVWrite = true
	mockBroadcastServer := &fcmocks.MockBroadcastServer{}
	mockBroadcastServer.Start(fmt.Sprintf("%s:%d", testhost, testBroadcastPort))
	mockBroadcastServer.BroadcastInternalServerError = true
	defer mockBroadcastServer.Stop()

	snap := newMockTxnSnap(nil)
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID1", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg := "INTERNAL_SERVER_ERROR"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}
	// registerTxEvent is true with wrongTxnID
	mockBroadcastServer.BroadcastInternalServerError = false

	//invoke transaction snap
	args = createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", true)
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg = "InvokeHandler execute failed: Client Status Code: (5) TIMEOUT"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}
}

func TestTransactionSnapInvokeFuncVerifyTxnProposalSignatureSuccess(t *testing.T) {
	//Replace client with mock client wrapper, which assumes channel is already initialized
	req := fabApi.ChaincodeInvokeRequest{
		ChaincodeID: "ccID",
		Args:        nil,
		Fcn:         "fcn",
	}
	signedProposal, err := newSignedProposal("testChannel", req)
	if err != nil {
		t.Fatalf("Error creating signed proposal: %s", err)
	}

	signedProposalBytes, err := proto.Marshal(signedProposal)
	if err != nil {
		t.Fatalf("Error Marshal signedProposal: %v", err)
	}

	snap := newMockTxnSnap(nil)
	stub := shim.NewMockStub("transactionsnap", snap)
	args := make([][]byte, 3)
	args[0] = []byte("verifyTransactionProposalSignature")
	args[1] = []byte("testChannel")
	args[2] = signedProposalBytes
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d error %v", shim.OK, response.Status, response.Message)
	}
}

func TestTransactionSnapInvokeFuncVerifyTxnProposalSignatureReturnError(t *testing.T) {
	req := fabApi.ChaincodeInvokeRequest{
		ChaincodeID: "ccID",
		Args:        nil,
		Fcn:         "fcn",
	}
	signedProposal, err := newSignedProposal("testChannel", req)
	if err != nil {
		t.Fatalf("Error creating signed proposal: %s", err)
	}
	signedProposal.Signature = []byte("wrongSignature")

	signedProposalBytes, err := proto.Marshal(signedProposal)
	if err != nil {
		t.Fatalf("Error Marshal signedProposal: %s", err)
	}

	snap := newMockTxnSnap(nil)
	stub := shim.NewMockStub("transactionsnap", snap)
	args := make([][]byte, 3)
	args[0] = []byte("verifyTransactionProposalSignature")
	args[1] = []byte("testChannel")
	args[2] = signedProposalBytes
	//invoke transaction snap

	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "The creator's signature over the proposal is not valid"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}
}

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, registerTxEvent bool) [][]byte {

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
		RegisterTxEvent:     registerTxEvent}
	snapTxReqB, err := json.Marshal(snapTxReq)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		return nil
	}

	var args [][]byte
	args = append(args, []byte(functionName))
	args = append(args, snapTxReqB)
	return args
}

// newSignedProposal creates a proposal for transaction. This involves assembling the proposal
// with the data (chaincodeName, function to call, arguments, transient data, etc.) and signing it using the private key corresponding to the
// ECert to sign.
func newSignedProposal(channelID string, request fabApi.ChaincodeInvokeRequest) (*pbsdk.SignedProposal, error) {

	txnID := "value"
	nonce := []byte("value")

	// Add function name to arguments
	argsArray := make([][]byte, len(request.Args)+1)
	argsArray[0] = []byte(request.Fcn)
	for i, arg := range request.Args {
		argsArray[i+1] = []byte(arg)
	}

	// create invocation spec to target a chaincode with arguments
	ccis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{
		Type: pb.ChaincodeSpec_GOLANG, ChaincodeId: &pb.ChaincodeID{Name: request.ChaincodeID},
		Input: &pb.ChaincodeInput{Args: argsArray}}}

	sID := &msp.SerializedIdentity{Mspid: mspID, IdBytes: []byte(mocks.CertPem)}
	creator, err := proto.Marshal(sID)
	if err != nil {
		return nil, err
	}

	proposal, _, err := protosUtils.CreateChaincodeProposalWithTxIDNonceAndTransient(txnID, common.HeaderType_ENDORSER_TRANSACTION, channelID, ccis, nonce, creator, request.TransientMap)
	if err != nil {
		return nil, fmt.Errorf("Could not create chaincode proposal, err %s", err)
	}

	// sign proposal bytes
	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling proposal: %s", err)
	}

	block, _ := pem.Decode(mocks.KeyPem)
	lowLevelKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	signature, err := mocks.SignECDSA(lowLevelKey, proposalBytes)
	if err != nil {
		return nil, err
	}

	// construct the transaction proposal
	signedProposal := &pbsdk.SignedProposal{ProposalBytes: proposalBytes, Signature: signature}

	return signedProposal, nil
}

// SignObjectWithKey will sign the given object with the given key,
// hashOpts and signerOpts
func signObjectWithKey(object []byte, key coreApi.Key,
	hashOpts coreApi.HashOpts, signerOpts coreApi.SignerOpts, cryptoSuite coreApi.CryptoSuite) ([]byte, error) {
	digest, err := cryptoSuite.Hash(object, hashOpts)
	if err != nil {
		return nil, err
	}
	signature, err := cryptoSuite.Sign(key, digest, signerOpts)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func TestTxnSnapUnsafeGetState(t *testing.T) {
	var args [][]byte

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)

	args = append(args, []byte("unsafeGetState"))
	response := stub.MockInvoke("TxID", args)
	assert.NotNil(t, response)
	assert.NotEqual(t, int32(200), response.GetStatus())
	assert.Contains(t, response.GetMessage(), "requires function and three args")

	args = append(args, []byte("channel"))
	args = append(args, []byte("cc"))
	args = append(args, []byte("key"))
	response = stub.MockInvoke("TxID", args)
	assert.NotNil(t, response)
	assert.NotEqual(t, int32(200), response.GetStatus())
	assert.Contains(t, response.GetMessage(), "Failed to open ledger")
}

func TestMain(m *testing.M) {
	main()
	//Setup bccsp factory
	// note: use of 'pkcs11' tag in the unit test will load the PCKS11 version of the factory opts.
	// otherwise default SW version will be used.
	//opts := sampleconfig.GetSampleBCCSPFactoryOpts("../sampleconfig")
	// TODO: remove code between the TODOs and uncomment above line and investigate
	// why s390 build is failing at the call `client.GetInstance(channelID, &sampleConfig{txSnapConfig})`
	// at line 281 below
	path := "./sampleconfig/msp/keystore"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(fmt.Sprintf("Wrong path: %v\n", err))
	}
	opts := &bccspFactory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &bccspFactory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &bccspFactory.FileKeystoreOpts{KeyStorePath: "./sampleconfig/msp/keystore"},
		},
	}
	// TDOD
	bccspFactory.InitFactories(opts)

	os.Setenv("CORE_PEER_ADDRESS", "peer1:5100")
	defer os.Unsetenv("CORE_PEER_ADDRESS")

	configData, err := ioutil.ReadFile("./sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %s\n", err))
	}
	configMsg := &configmanagerApi.ConfigMessage{MspID: mspID,
		Peers: []configmanagerApi.PeerConfig{{
			PeerID: "jdoe", App: []configmanagerApi.AppConfig{
				{AppName: "txnsnap", Version: configmanagerApi.VERSION, Config: string(configData)}}}}}
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

	_, err = config.NewConfig("./sampleconfig", channelID)
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}

	txsnapservice.PeerConfigPath = sampleconfig.ResolvPeerConfig("./sampleconfig")

	txSnapConfig, err = config.NewConfig("./sampleconfig", channelID)
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
	fcClient, err = client.GetInstance("testChannel")
	if err != nil {
		panic(fmt.Sprintf("Client GetInstance return error %s", err))
	}

	mockMembership := &mockmembership.Service{
		PeersOfChannel: map[string][]*memApi.PeerEndpoint{
			channelID: {
				&memApi.PeerEndpoint{
					Endpoint:     "grpc://127.0.0.1:7040",
					MSPid:        []byte("org1MSP"),
					LedgerHeight: 1000,
					Roles:        []string{memservice.EndorserRole, memservice.CommitterRole},
				},
			},
		},
	}

	client.MemServiceProvider = func() (memApi.Service, error) {
		return mockMembership, nil
	}

	os.Exit(m.Run())

}

func newMockTxService(callback api.EndorsedCallback) *txsnapservice.TxServiceImpl {
	return &txsnapservice.TxServiceImpl{
		FcClient: fcClient,
		Callback: callback,
	}
}

func newMockTxnSnap(callback api.EndorsedCallback) *TxnSnap {
	return &TxnSnap{getTxService: func(channelID string) (*txsnapservice.TxServiceImpl, error) {
		return newMockTxService(callback), nil
	}}
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
