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
	"testing"
	"time"

	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/mocks/event/mockevent"
	"github.com/securekey/fabric-snaps/mocks/event/mockproducer"
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/mocks/mockchpeer"

	"github.com/gogo/protobuf/proto"
	apiconfig "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	sdkpeer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	"github.com/hyperledger/fabric/bccsp"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	protosUtils "github.com/hyperledger/fabric/protos/utils"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	evservice "github.com/securekey/fabric-snaps/eventservice/pkg/service"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/config"
	mocks "github.com/securekey/fabric-snaps/transactionsnap/cmd/mocks"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/sampleconfig"
)

var channelID = "testChannel"
var mspID = "Org1MSP"
var mockEndorserServer *fcmocks.MockEndorserServer
var mockBroadcastServer *fcmocks.MockBroadcastServer
var mockEventServer *fcmocks.MockEventServer
var eventProducer *mockproducer.MockProducer

var endorserTestURL = "127.0.0.1:7040"
var broadcastTestURL = "127.0.0.1:7041"
var endorserTestEventHost = "127.0.0.1"
var endorserTestEventPort = 7053
var membership api.MembershipManager
var txSnapConfig api.Config
var fcClient api.Client

var validRootCA = `-----BEGIN CERTIFICATE-----
MIICSDCCAe6gAwIBAgIRAPnKpS42wlgtHsddm6q+kYcwCgYIKoZIzj0EAwIwcDEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xGTAXBgNVBAoTEG9yZzEuZXhhbXBsZS5jb20xGTAXBgNVBAMTEG9y
ZzEuZXhhbXBsZS5jb20wHhcNMTcwNDIyMTIwMjU2WhcNMjcwNDIwMTIwMjU2WjBw
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2Fu
IEZyYW5jaXNjbzEZMBcGA1UEChMQb3JnMS5leGFtcGxlLmNvbTEZMBcGA1UEAxMQ
b3JnMS5leGFtcGxlLmNvbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABLi5341r
mriGFHCmVTLdgPGpDFRgwgmHSuLayMsGP0yEmsXh3hKAy24f1mjx/t8WT9G2sAdw
ONsPsfKMSCKpaRqjaTBnMA4GA1UdDwEB/wQEAwIBpjAZBgNVHSUEEjAQBgRVHSUA
BggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdDgQiBCCiLa81ayqrV5Lq
U+NfZvzO8dfxqis6K5Lb+/lqRI6iajAKBggqhkjOPQQDAgNIADBFAiEAr8LYCY2b
q5kNqOUxgHwBa2KTi/zJBR9L3IsTRDjJo8ECICf1xiDgKqZKrAMh0OCebskYwf53
dooG04HBoqBLvB8Q
-----END CERTIFICATE-----
`

const (
	org1 = "Org1MSP"
	org2 = "Org2MSP"
)

var p1, p2 api.ChannelPeer

type sampleConfig struct {
	api.Config
}

// Override GetMspConfigPath for relative path, just to avoid using new core.yaml for this purpose
func (c *sampleConfig) GetMspConfigPath() string {
	return "../sampleconfig/msp"
}

func TestEndorseTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil)
	txService := newMockTxService(nil)
	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}
	snapTxReq = createTransactionSnapRequest("endorsetransaction", "ccid", "", false, nil)
	fmt.Printf("%v\n", txService)
	_, err = txService.EndorseTransaction(&snapTxReq, nil)
	if err == nil {
		t.Fatalf("ChannelID is required field")
	}

	snapTxReq = createTransactionSnapRequest("endorsetransaction", "", channelID, false, nil)
	fmt.Printf("%v\n", txService)
	_, err = txService.EndorseTransaction(&snapTxReq, nil)
	if err == nil {
		t.Fatalf("ChannelID is required field")
	}

}

func TestEndorseTransactionWithPeerFilter(t *testing.T) {
	peerFilterOpts := &api.PeerFilterOpts{
		Type: api.MinBlockHeightPeerFilterType,
		Args: []string{channelID},
	}

	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, peerFilterOpts)
	txService := newMockTxService(nil)
	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}
}

func TestCommitTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, false, nil)
	txService := newMockTxService(nil)
	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}
	validationCode, err := txService.CommitTransaction(channelID, txnProposalResponse, false)
	if err != nil {
		t.Fatalf("Expected to commit tx")
	}
	if validationCode != 0 {
		t.Fatalf("Expected to commit tx")
	}
	_, err = txService.CommitTransaction("", txnProposalResponse, false)
	if err == nil {
		t.Fatalf("ChannelID is required in commit transaction")
	}

	_, err = txService.CommitTransaction("channelID", nil, false)
	if err == nil {
		t.Fatalf("TxProposalResponse is null. Expected error")
	}
	_, err = txService.CommitTransaction("channelID", nil, false)
	if err == nil {
		t.Fatalf("Expected error:'Error creating transaction: at least one proposal response is necessary'")
	}

}

func TestEndorseCommitTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", channelID, true, nil)
	txService := newMockTxService(func(responses []*apitxn.TransactionProposalResponse) error {
		go func() {
			time.Sleep(2 * time.Second)
			eventProducer.ProduceEvent(
				mockevent.NewFilteredBlockEvent(
					channelID,
					mockevent.NewFilteredTx(responses[0].Proposal.TxnID.ID, pb.TxValidationCode_VALID),
				),
			)
		}()
		return nil
	})

	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}

	validationCode, err := txService.EndorseAndCommitTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Expected to commit tx")
	}
	if validationCode != 0 {
		t.Fatalf("Expected to commit tx")
	}

}

func TestVerifyProposalSignature(t *testing.T) {
	txService := newMockTxService(nil)
	err := txService.VerifyTxnProposalSignature("", nil)
	if err == nil {
		t.Fatalf("ChannelID is mandatory field")
	}
	err = txService.VerifyTxnProposalSignature("testChannelID", nil)
	if err == nil {
		t.Fatalf("SignedProposal is mandatory field")
	}

}

func newMockTxService(callback EndorsedCallback) *TxServiceImpl {
	return &TxServiceImpl{
		Config:     txSnapConfig,
		FcClient:   fcClient,
		Membership: membership,
		Callback:   callback,
	}
}

func TestMain(m *testing.M) {

	//Setup bccsp factory
	// note: use of 'pkcs11' tag in the unit test will load the PCKS11 version of the factory opts.
	// otherwise default SW version will be used.
	opts := sampleconfig.GetSampleBCCSPFactoryOpts("./sampleconfig")
	bccspFactory.InitFactories(opts)

	bccspFactory.InitFactories(opts)
	configData, err := ioutil.ReadFile("../sampleconfig/config.yaml")
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

	PeerConfigPath = "../sampleconfig"
	txSnapConfig, err = config.NewConfig("../sampleconfig", channelID)
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}

	fcClient, err = client.GetInstance(channelID, &sampleConfig{txSnapConfig})
	if err != nil {
		panic(fmt.Sprintf("Client GetInstance return error %v", err))
	}
	configureClient(fcClient, &sampleConfig{txSnapConfig}, configData)

	p1 = peer("grpc://peer1:7051", org1)
	p2 = peer("grpc://peer2:7051", org1)
	membership = mocks.NewMockMembershipManager(nil).Add(channelID, p1, p2)
	mockEndorserServer = fcmocks.StartEndorserServer(endorserTestURL)
	mockBroadcastServer = fcmocks.StartMockBroadcastServer(broadcastTestURL)
	if eventProducer == nil {
		eventService, producer, err := evservice.NewServiceWithMockProducer(channelID, []evservice.EventType{evservice.FILTEREDBLOCKEVENT}, evservice.DefaultOpts())
		localservice.Register(channelID, eventService)
		if err != nil {
			panic(err.Error())
		}
		eventProducer = producer
	}
	testChannel, err := fcClient.NewChannel(channelID)
	if err != nil {
		panic(fmt.Sprintf("NewChannel return error: %v", err))
	}
	builder := &fcmocks.MockConfigUpdateEnvelopeBuilder{
		ChannelID: channelID,
		MockConfigGroupBuilder: fcmocks.MockConfigGroupBuilder{
			ModPolicy:      "Admins",
			MSPNames:       []string{"Org1MSP"},
			OrdererAddress: "localhost:8085",
			RootCA:         validRootCA,
		},
	}
	err = testChannel.Initialize(builder.BuildConfigUpdateBytes())
	if err != nil {
		panic(fmt.Sprintf("channel Initialize failed : %v", err))
	}
	DoIntializeChannel = false
	os.Exit(m.Run())

}

func peer(url string, mspID string) api.ChannelPeer {
	return mockchpeer.New(url, mspID, "", 0)
}

// newTransactionProposal creates a proposal for transaction. This involves assembling the proposal
// with the data (chaincodeName, function to call, arguments, transient data, etc.) and signing it using the private key corresponding to the
// ECert to sign.
func newTransactionProposalB(channelID string, request apitxn.ChaincodeInvokeRequest, user sdkApi.User) ([]byte, error) {

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

	creator, err := user.Identity()
	if err != nil {
		return nil, fmt.Errorf("Error getting creator: %v", err)
	}

	proposal, _, err := protosUtils.CreateChaincodeProposalWithTxIDNonceAndTransient(request.TxnID.ID, common.HeaderType_ENDORSER_TRANSACTION, channelID, ccis, request.TxnID.Nonce, creator, request.TransientMap)
	if err != nil {
		return nil, fmt.Errorf("Could not create chaincode proposal, err %s", err)
	}

	// sign proposal bytes
	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling proposal: %v", err)
	}
	return proposalBytes, nil

}

func newTransactionProposal(channelID string, request apitxn.ChaincodeInvokeRequest, user sdkApi.User) (*pb.SignedProposal, error) {

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

	creator, err := user.Identity()
	if err != nil {
		return nil, fmt.Errorf("Error getting creator: %v", err)
	}

	proposal, _, err := protosUtils.CreateChaincodeProposalWithTxIDNonceAndTransient(request.TxnID.ID, common.HeaderType_ENDORSER_TRANSACTION, channelID, ccis, request.TxnID.Nonce, creator, request.TransientMap)
	if err != nil {
		return nil, fmt.Errorf("Could not create chaincode proposal, err %s", err)
	}

	// sign proposal bytes
	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		return nil, fmt.Errorf("Error marshalling proposal: %v", err)
	}

	if user == nil {
		return nil, fmt.Errorf("Error getting user context: %s", err)
	}

	cryptoSuite := factories.GetSuite(bccspFactory.GetDefault())
	var key apicryptosuite.Key
	key = user.PrivateKey()
	signature, err := signObjectWithKey(proposalBytes, key, &bccsp.SHAOpts{}, nil, cryptoSuite)
	if err != nil {
		return nil, err
	}

	// construct the transaction proposal
	signedProposal := pb.SignedProposal{ProposalBytes: proposalBytes, Signature: signature}

	return &signedProposal, nil
}

func signObjectWithKey(object []byte, key apicryptosuite.Key,
	hashOpts apicryptosuite.HashOpts, signerOpts apicryptosuite.SignerOpts, cryptoSuite apicryptosuite.CryptoSuite) ([]byte, error) {
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

func configureClient(fabricClient api.Client, config api.Config, sdkConfig []byte) api.Client {

	newtworkConfig, _ := fabricClient.GetConfig().NetworkConfig()
	newtworkConfig.Orderers["orderer.example.com"] = apiconfig.OrdererConfig{URL: broadcastTestURL}

	//create selection service
	peer, _ := sdkpeer.New(fabricClient.GetConfig(), sdkpeer.WithURL(endorserTestURL))
	selectionService := mocks.MockSelectionService{TestEndorsers: []sdkApi.Peer{peer},
		TestPeer:       api.PeerConfig{EventHost: endorserTestEventHost, EventPort: endorserTestEventPort},
		InvalidChannel: ""}

	fabricClient.SetSelectionService(&selectionService)
	return fabricClient
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
