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

	"github.com/gogo/protobuf/proto"
	apiconfig "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	"github.com/hyperledger/fabric/bccsp"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	protosUtils "github.com/hyperledger/fabric/protos/utils"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/config"
	mocks "github.com/securekey/fabric-snaps/transactionsnap/cmd/mocks"
)

var channelID = "testChannel"
var mspID = "Org1MSP"
var mockEndorserServer *fcmocks.MockEndorserServer
var mockBroadcastServer *fcmocks.MockBroadcastServer
var mockEventServer *fcmocks.MockEventServer

var endorserTestURL = "127.0.0.1:7040"
var broadcastTestURL = "127.0.0.1:7041"
var endorserTestEventHost = "127.0.0.1"
var endorserTestEventPort = 7053
var membership api.MembershipManager
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

var p1, p2 sdkApi.Peer

type sampleConfig struct {
	api.Config
}

// Override GetMspConfigPath for relative path, just to avoid using new core.yaml for this purpose
func (c *sampleConfig) GetMspConfigPath() string {
	return "../sampleconfig/msp"
}

var txService TxServiceImpl

func TestEndorseTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", "testChannel", false)
	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}
	snapTxReq = createTransactionSnapRequest("endorsetransaction", "ccid", "", false)
	fmt.Printf("%v\n", txService)
	_, err = txService.EndorseTransaction(&snapTxReq, nil)
	if err == nil {
		t.Fatalf("ChannelID is required field")
	}

	snapTxReq = createTransactionSnapRequest("endorsetransaction", "", "testChannel", false)
	fmt.Printf("%v\n", txService)
	_, err = txService.EndorseTransaction(&snapTxReq, nil)
	if err == nil {
		t.Fatalf("ChannelID is required field")
	}

}

func TestCommitTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", "testChannel", false)
	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}
	timeDuration := time.Duration(1) * time.Millisecond
	validationCode, err := txService.CommitTransaction("testChannel", txnProposalResponse, false, timeDuration)
	if err != nil {
		t.Fatalf("Expected to commit tx")
	}
	if validationCode != 0 {
		t.Fatalf("Expected to commit tx")
	}
	_, err = txService.CommitTransaction("", txnProposalResponse, false, timeDuration)
	if err == nil {
		t.Fatalf("ChannelID is required in commit transaction")
	}

	_, err = txService.CommitTransaction("channelID", nil, false, timeDuration)
	if err == nil {
		t.Fatalf("TxProposalResponse is null. Expected error")
	}
	timeDuration = time.Duration(-1) * time.Millisecond
	_, err = txService.CommitTransaction("channelID", nil, false, timeDuration)
	if err == nil {
		t.Fatalf("Expected error:'Error creating transaction: at least one proposal response is necessary'")
	}

}

func TestEndorseCommitTransaction(t *testing.T) {
	snapTxReq := createTransactionSnapRequest("endorsetransaction", "ccid", "testChannel", false)
	fmt.Printf("%v\n", txService)
	txnProposalResponse, err := txService.EndorseTransaction(&snapTxReq, nil)
	if err != nil {
		t.Fatalf("Error endorsing transaction %v", err)
	}
	if txnProposalResponse == nil {
		t.Fatalf("Expected proposal response")
	}
	timeDuration := time.Duration(1) * time.Millisecond
	validationCode, err := txService.EndorseAndCommitTransaction(&snapTxReq, nil, timeDuration)
	if err != nil {
		t.Fatalf("Expected to commit tx")
	}
	if validationCode != 0 {
		t.Fatalf("Expected to commit tx")
	}

}

func TestVerifyProposalSignature(t *testing.T) {
	err := txService.VerifyTxnProposalSignature("", nil)
	if err == nil {
		t.Fatalf("ChannelID is mandatory field")
	}
	err = txService.VerifyTxnProposalSignature("testChannelID", nil)
	if err == nil {
		t.Fatalf("SignedProposal is mandatory field")
	}

}

func TestMain(m *testing.M) {

	path := "../sampleconfig/msp/keystore"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		panic(fmt.Sprintf("Wrong path: %v\n", err))
	}
	opts := &bccspFactory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &bccspFactory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &bccspFactory.FileKeystoreOpts{KeyStorePath: "../sampleconfig/msp/keystore"},
		},
	}
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
	config, err := config.NewConfig("../sampleconfig", channelID)
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}

	fcClient, err = client.GetInstance(channelID, &sampleConfig{config})
	if err != nil {
		panic(fmt.Sprintf("Client GetInstance return error %v", err))
	}
	configureClient(fcClient, &sampleConfig{config}, configData)

	p1 = peer("grpc://peer1:7051", org1)
	p2 = peer("grpc://peer2:7051", org1)
	membership = mocks.NewMockMembershipManager(nil).Add("testChannel", p1, p2)

	txService.Config = config
	txService.FcClient = fcClient
	txService.Membership = membership
	//txService = TxServiceImpl{Config: config, FcClient: fcClient, Membership: membership}
	mockEndorserServer = fcmocks.StartEndorserServer(endorserTestURL)
	mockBroadcastServer = fcmocks.StartMockBroadcastServer(broadcastTestURL)
	mockEventServer, err = fcmocks.StartMockEventServer(fmt.Sprintf("%s:%d", endorserTestEventHost, endorserTestEventPort))
	if err != nil {
		panic(err.Error())
	}
	testChannel, err := fcClient.NewChannel("testChannel")
	if err != nil {
		panic(fmt.Sprintf("NewChannel return error: %v", err))
	}
	builder := &fcmocks.MockConfigUpdateEnvelopeBuilder{
		ChannelID: "testChannel",
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

func peer(url string, mspID string) sdkApi.Peer {

	peer, err := sdkFabApi.NewPeer(url, "", "", fcClient.GetConfig())
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}

	peer.SetName(url)
	peer.SetMSPID(mspID)
	fmt.Printf("\npeer %v\n", peer)
	return peer
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
	peer, _ := sdkFabApi.NewPeer(endorserTestURL, "", "", fabricClient.GetConfig())
	selectionService := mocks.MockSelectionService{TestEndorsers: []sdkApi.Peer{peer},
		TestPeer:       api.PeerConfig{EventHost: endorserTestEventHost, EventPort: endorserTestEventPort},
		InvalidChannel: ""}

	fabricClient.SetSelectionService(&selectionService)
	return fabricClient
}

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, registerTxEvent bool) api.SnapTransactionRequest {

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

	return snapTxReq

}

func getMockStub() *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	return stub
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(stub *shim.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}
