/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transactionsnap

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	clientConfig "github.com/hyperledger/fabric-sdk-go/pkg/config"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/client"
	config "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/config"
	mocks "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/mocks"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

var mockEndorserServer *MockEndorserServer

var testHost = "127.0.0.1"
var testPort = 7050
var testEventPort = 17564
var proposalReturnStatus int32
var proposalError error

// MockEndorserServer mock endoreser server to process endorsement proposals
type MockEndorserServer struct {
	ProposalError error
}

// ProcessProposal mock implementation
func (m *MockEndorserServer) ProcessProposal(context context.Context,
	proposal *pb.SignedProposal) (*pb.ProposalResponse, error) {
	return &pb.ProposalResponse{Response: &pb.Response{
		Status:  proposalReturnStatus,
		Payload: []byte("testsomething"),
	}}, proposalError
}

func TestTransactionSnapInit(t *testing.T) {
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	var args [][]byte
	response := stub.MockInit("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expecting response status %d but got %d", shim.OK, response.Status)
	}
}

func TestNotSupportedFunction(t *testing.T) {
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("notSupportedFunction", "ccid", "testChannel")
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}

	errorMsg := "Function notSupportedFunction is not supported"
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestNotSpecifiedChannel(t *testing.T) {
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "")
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "Cannot create channel Error creating new channel: failed to create Channel. Missing required 'name' parameter"
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestNotSecifiedChaincodeID(t *testing.T) {

	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "", "testChannel")
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "ChaincodeID is mandatory field of the SnapTransactionRequest"
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestSupportedFunctionWithoutRequest(t *testing.T) {

	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	var args [][]byte
	args = append(args, []byte("endorseTransaction"))
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "Not enough arguments in call to endorse transaction"
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestSupportedFunctionWithNilRequest(t *testing.T) {

	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	var args [][]byte
	args = append(args, []byte("endorseTransaction"))
	args = append(args, nil)
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "Cannot decode parameters from request to endorse transaction unexpected end of JSON input"
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestTransactionSnapInvokeFuncEndorseTransactionStatusSuccess(t *testing.T) {
	proposalReturnStatus = 200
	proposalError = nil
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel")
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	if len(response.GetPayload()) == 0 {
		t.Fatalf("Received an empty payload")
	}
	var tpResponse []*apitxn.TransactionProposalResponse
	err := json.Unmarshal(response.GetPayload(), &tpResponse)
	if err != nil {
		t.Fatalf("Cannot unmarshal transaction proposal response %v", err)
	}
	if len(tpResponse) == 0 {
		t.Fatalf("Received an empty transaction proposal response")
	}
	if tpResponse[0].ProposalResponse.Response.Status != 200 {
		t.Fatalf("Expected proposal response status: SUCCESS")
	}

}

func TestTransactionSnapInvokeFuncEndorseTransactionStatusFailed(t *testing.T) {
	proposalReturnStatus = 500
	proposalError = nil
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel")
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	if len(response.GetPayload()) == 0 {
		t.Fatalf("Received an empty payload")
	}
	var tpResponse []*apitxn.TransactionProposalResponse
	err := json.Unmarshal(response.GetPayload(), &tpResponse)
	if err != nil {
		t.Fatalf("Cannot unmarshal transaction proposal response %v", err)
	}
	if len(tpResponse) == 0 {
		t.Fatalf("Received an empty transaction proposal response")
	}
	if tpResponse[0].ProposalResponse.Response.Status != 500 {
		t.Fatalf("Expected proposal response status: FAILED")
	}

}

func TestTransactionSnapInvokeFuncEndorseTransactionReturnError(t *testing.T) {
	proposalError = fmt.Errorf("proposalError")
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel")
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

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string) [][]byte {

	transientMap := make(map[string][]byte)
	transientMap["key"] = []byte("transientvalue")
	endorserArgs := make([][]byte, 5)
	endorserArgs[0] = []byte("invoke")
	endorserArgs[1] = []byte("move")
	endorserArgs[2] = []byte("a")
	endorserArgs[3] = []byte("b")
	endorserArgs[4] = []byte("1")
	additionalCCIDs := []string{"additionalccid"}
	snapTxReq := SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:     chaincodeID,
		TransientMap:    transientMap,
		EndorserArgs:    endorserArgs,
		AdditionalCCIDs: additionalCCIDs}
	snapTxReqB, err := json.Marshal(snapTxReq)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return nil
	}

	var args [][]byte
	args = append(args, []byte(functionName))
	args = append(args, snapTxReqB)
	return args
}

func startEndorserServer() *MockEndorserServer {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", testHost, testPort))
	endorserServer := &MockEndorserServer{}
	pb.RegisterEndorserServer(grpcServer, endorserServer)
	if err != nil {
		panic(fmt.Sprintf("Error starting endorser server: %s", err))
	}
	fmt.Printf("Test endorser server started\n")
	go grpcServer.Serve(lis)
	return endorserServer
}

func configureClient() client.Client {
	fabricClient, err := client.GetInstance()
	if err != nil {
		panic(fmt.Sprintf("Error initializing fabricClient: %s", err))
	}
	clientConfig.FabricClientViper().Set("client.tls.enabled", false)

	//create selection service
	peer, _ := sdkFabApi.NewPeer(fmt.Sprintf("%s:%d", testHost, testPort), "", "", fabricClient.GetConfig())
	selectionService := mocks.MockSelectionService{TestEndorsers: []sdkApi.Peer{peer},
		TestPeer:       config.PeerConfig{EventHost: testHost, EventPort: testEventPort},
		InvalidChannel: ""}

	fabricClient.SetSelectionService(&selectionService)
	return fabricClient
}

func TestMain(m *testing.M) {
	err := config.Init("")
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}
	configureClient()
	startEndorserServer()

	os.Exit(m.Run())
}
