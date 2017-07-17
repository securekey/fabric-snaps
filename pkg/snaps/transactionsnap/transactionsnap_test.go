/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	clientConfig "github.com/hyperledger/fabric-sdk-go/pkg/config"

	fcMocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	ab "github.com/hyperledger/fabric/protos/orderer"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/client"
	config "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/config"
	mocks "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/mocks"

	"google.golang.org/grpc"
)

var mockEndorserServer *fcMocks.MockEndorserServer
var mockBroadcastServer *fcMocks.MockBroadcastServer
var mockEventServer *mocks.MockEventServer

var endorserTestHost = "127.0.0.1"
var endorserTestPort = 7040
var endorserTestEventPort = 17564
var broadcastTestHost = "127.0.0.1"
var broadcastTestPort = 7041

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
	args := createTransactionSnapRequest("notSupportedFunction", "ccid", "testChannel", false)
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
	var funcs []string
	funcs = append(funcs, "endorseTransaction")
	funcs = append(funcs, "commitTransaction")
	for _, value := range funcs {
		args := createTransactionSnapRequest(value, "ccid", "", false)
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
}

func TestNotSpecifiedChaincodeID(t *testing.T) {

	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "", "testChannel", false)
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
		errorMsg := fmt.Sprintf("Not enough arguments in call to %s transaction", value)
		if response.Message != errorMsg {
			t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
		}
	}
}

func TestSupportedFunctionWithNilRequest(t *testing.T) {

	snap := &TxnSnap{}
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
		errorMsg := "Cannot decode parameters from request to Snap Transaction Request unexpected end of JSON input"
		if response.Message != errorMsg {
			t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
		}
	}
}

func TestTransactionSnapInvokeFuncEndorseTransactionStatusSuccess(t *testing.T) {
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = false
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel", false)
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

func TestTransactionSnapInvokeFuncEndorseTransactionReturnError(t *testing.T) {
	mockEndorserServer.ProposalError = fmt.Errorf("proposalError")
	mockEndorserServer.AddkvWrite = false
	snap := &TxnSnap{}
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
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = false
	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)

	// registerTxEvent is false
	args := createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}

	// registerTxEvent is true
	args = createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", true)
	go func() {
		time.Sleep(time.Second * 2)
		mockBlock, err := mocks.CreateBlockWithCCEvent(&pb.ChaincodeEvent{}, newTxID.ID, "testChannel")
		if err != nil {
			fmt.Printf("Error CreateBlockWithCCEvent %v\n", err)
			return
		}
		mockEventServer.SendMockEvent(&pb.Event{Event: &pb.Event_Block{Block: mockBlock}})
	}()

	//invoke transaction snap
	response = stub.MockInvoke("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
}

func TestTransactionSnapInvokeFuncCommitTransactionReturnError(t *testing.T) {
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = true

	snap := &TxnSnap{}
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg := "broadcast response is not success : INTERNAL_SERVER_ERROR"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}
	// registerTxEvent is true with wrongTxnID
	mockBroadcastServer.BroadcastInternalServerError = false
	registerTxEventTimeout = 5
	args = createTransactionSnapRequest("commitTransaction", "ccid", "testChannel", true)
	go func() {
		time.Sleep(time.Second * 2)
		mockBlock, err := mocks.CreateBlockWithCCEvent(&pb.ChaincodeEvent{}, "wrongTxnID", "testChannel")
		if err != nil {
			fmt.Printf("Error CreateBlockWithCCEvent %v\n", err)
			return
		}
		mockEventServer.SendMockEvent(&pb.Event{Event: &pb.Event_Block{Block: mockBlock}})
	}()

	//invoke transaction snap
	response = stub.MockInvoke("TxID", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg = "SendTransaction Didn't receive tx event for txid"
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
	additionalCCIDs := []string{"additionalccid"}
	snapTxReq := SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:     chaincodeID,
		TransientMap:    transientMap,
		EndorserArgs:    endorserArgs,
		AdditionalCCIDs: additionalCCIDs,
		RegisterTxEvent: registerTxEvent}
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

func startEndorserServer() *fcMocks.MockEndorserServer {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", endorserTestHost, endorserTestPort))
	endorserServer := &fcMocks.MockEndorserServer{}
	pb.RegisterEndorserServer(grpcServer, endorserServer)
	if err != nil {
		panic(fmt.Sprintf("Error starting endorser server: %s", err))
	}
	fmt.Printf("Test endorser server started\n")
	go grpcServer.Serve(lis)
	return endorserServer
}

func startBroadcastServer() *fcMocks.MockBroadcastServer {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", broadcastTestHost, broadcastTestPort))
	broadcastServer := new(fcMocks.MockBroadcastServer)
	ab.RegisterAtomicBroadcastServer(grpcServer, broadcastServer)
	if err != nil {
		panic(fmt.Sprintf("Error starting BroadcastServer %s", err))
	}
	fmt.Printf("Test broadcast server started\n")
	go grpcServer.Serve(lis)

	return broadcastServer
}

func configureClient() client.Client {
	fabricClient, err := client.GetInstance()
	if err != nil {
		panic(fmt.Sprintf("Error initializing fabricClient: %s", err))
	}
	clientConfig.FabricClientViper().Set("client.tls.enabled", false)

	newtworkConfig, _ := fabricClient.GetConfig().NetworkConfig()
	newtworkConfig.Orderers["orderer0"] = apiconfig.OrdererConfig{Host: broadcastTestHost, Port: broadcastTestPort}
	clientConfig.FabricClientViper().Set("client.network", newtworkConfig)

	//create selection service
	peer, _ := sdkFabApi.NewPeer(fmt.Sprintf("%s:%d", endorserTestHost, endorserTestPort), "", "", fabricClient.GetConfig())
	selectionService := mocks.MockSelectionService{TestEndorsers: []sdkApi.Peer{peer},
		TestPeer:       config.PeerConfig{EventHost: endorserTestHost, EventPort: endorserTestEventPort},
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
	mockEndorserServer = startEndorserServer()
	mockBroadcastServer = startBroadcastServer()
	mockEventServer, err = mocks.StartMockEventServer(fmt.Sprintf("%s:%d", endorserTestHost, endorserTestEventPort))
	if err != nil {
		panic(err.Error())
	}
	os.Exit(m.Run())
}
