/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric/bccsp"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/protos/common"
	protosUtils "github.com/hyperledger/fabric/protos/utils"

	"github.com/hyperledger/fabric/core/chaincode/shim"

	pbsdk "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	evservice "github.com/securekey/fabric-snaps/eventservice/pkg/service"
	"github.com/securekey/fabric-snaps/mocks/event/mockevent"
	"github.com/securekey/fabric-snaps/mocks/event/mockproducer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/config"
	mocks "github.com/securekey/fabric-snaps/transactionsnap/cmd/mocks"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/txsnapservice"
)

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
var mockEndorserServer *fcmocks.MockEndorserServer
var mockBroadcastServer *fcmocks.MockBroadcastServer

var eventProducer *mockproducer.MockProducer
var endorserTestURL = "127.0.0.1:7040"
var broadcastTestURL = "127.0.0.1:7041"
var endorserTestEventHost = "127.0.0.1"
var endorserTestEventPort = 7053
var membership api.MembershipManager
var fcClient api.Client
var channelID = "testChannel"
var mspID = "Org1MSP"

const (
	org1 = "Org1MSP"
	org2 = "Org2MSP"
)

var p1, p2 sdkApi.Peer

func TestTransactionSnapInit(t *testing.T) {
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	var args [][]byte
	response := stub.MockInit("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expecting response status %d but got %d", shim.OK, response.Status)
	}
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
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

//This test will be re-introduced with mock txSnapService
func testGetPeersOfChannel(t *testing.T) {

	// membership = mocks.NewMockMembershipManager(nil).Add("testChannel", p1, p2)
	// err := configureTxService("testChannel")
	// if err != nil {
	// 	t.Fatalf("Error %v", err)
	// }
	// txSnapService.Membership = membership
	// fmt.Printf("%s", txSnapService.Membership)

	// b, err := getPeersOfChannel([]string{"testChannel"})
	// if err != nil {
	// 	t.Fatalf("Error %v", err)
	// }
	// fmt.Printf("Peersof channel %s", string(b))
	// if len(b) < 2 {
	// 	t.Fatalf("Expected that two peers were configured ")
	// }

}

//This test will be re-introduced with mock txSnapService

func testGetPeersOfChannelQueryErrorWarning(t *testing.T) {

	// membership = mocks.NewMockMembershipManager(errors.New("Query Error")).Add("testChannel", p1, p2)
	// err := configureTxService("testChannel")
	// if err != nil {
	// 	t.Fatalf("Error %v", err)
	// }
	// txSnapService.Membership = membership
	// fmt.Printf("%s", txSnapService.Membership)

	// b, err := getPeersOfChannel([]string{"testChannel"})
	// if err != nil {
	// 	t.Fatalf("Error %v", err)
	// }
	// fmt.Printf("Peersof channel%s", string(b))
	// if len(b) < 2 {
	// 	t.Fatalf("Expected that two peers were configured ")
	// }

}

//This test will be re-introduced with mock txSnapService

func testGetPeersOfChannelQueryErrorNoPeers(t *testing.T) {

	// membership = mocks.NewMockMembershipManager(errors.New("Query Error"))
	// err := configureTxService("testChannel")
	// if err != nil {
	// 	t.Fatalf("Error %v", err)
	// }
	// txSnapService.Membership = membership
	// fmt.Printf("%s", txSnapService.Membership)

	// _, err = getPeersOfChannel([]string{"testChannel"})
	// if err == nil {
	// 	t.Fatalf("Expected error 'ould not get peers on channel testChannel: Query Error'")
	// }

}

func TestWrongRegisterTxEventValue(t *testing.T) {
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := make([][]byte, 4)
	args[0] = []byte("commitTransaction")
	args[1] = []byte("testChannel")
	args[2] = nil
	args[3] = []byte("false1")
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := `Cannot ParseBool the fourth arg to registerTxEvent strconv.ParseBool: parsing "false1": invalid syntax`
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
}

func TestNotSpecifiedChannel(t *testing.T) {
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	var funcs []string
	funcs = append(funcs, "endorseTransaction")
	funcs = append(funcs, "commitTransaction")
	funcs = append(funcs, "endorseAndCommitTransaction")
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
			t.Fatalf("Expecting error due to an misconfigured endorsers args")
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
	if response.Message != errorMsg {
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
			t.Fatalf("Expecting error 'ChaincodeID is mandatory field of the SnapTransactionRequest'")
		}
	}

	var args [][]byte
	args = append(args, []byte("endorseAndCommitTransaction"))
	response := stub.MockInvoke("TxID1", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg := "Not enough arguments in call to endorse and commit transaction"
	if response.Message != errorMsg {
		t.Fatalf("Expecting error message(%s) but got %s", errorMsg, response.Message)
	}
	args = args[:0]
	args = append(args, []byte("verifyTransactionProposalSignature"))
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	if response.Message == "" {
		t.Fatalf("Expecting 'Expected args  containing channelID'")
	}

	args = args[:0]
	args = append(args, []byte("getPeersOfChannel"))
	response = stub.MockInvoke("TxID1", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	if response.Message == "" {
		t.Fatalf("Expecting 'Expected args  containing channelID'")
	}

}

func TestSupportedFunctionWithNilRequest(t *testing.T) {

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	var funcs []string
	funcs = append(funcs, "endorseTransaction")
	funcs = append(funcs, "endorseAndCommitTransaction")
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
	snap := New()
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
	snap := New()
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

func TestTransactionSnapInvokeFuncEndorseAndCommitTransactionSuccess(t *testing.T) {
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = false
	snap := &TxnSnap{getTxService: func(channelID string) (*txsnapservice.TxServiceImpl, error) {
		txService, err := txsnapservice.Get(channelID)
		if err != nil {
			return nil, err
		}
		txService.Callback = func(responses []*apitxn.TransactionProposalResponse) error {
			go func() {
				time.Sleep(3 * time.Second)
				eventProducer.ProduceEvent(
					mockevent.NewFilteredBlockEvent(
						channelID,
						mockevent.NewFilteredTx(responses[0].Proposal.TxnID.ID, pb.TxValidationCode_VALID),
					),
				)
			}()
			return nil
		}
		return txService, nil
	}}
	stub := shim.NewMockStub("transactionsnap", snap)

	// registerTxEvent is false
	args := createTransactionSnapRequest("endorseAndCommitTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID1", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}

	//invoke transaction snap
	args = createTransactionSnapRequest("endorseAndCommitTransaction", "ccid", "testChannel", true)
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d (%s)", shim.OK, response.Status, response.Message)
	}
}

func TestTransactionSnapInvokeFuncEndorseAndCommitTransactionReturnError(t *testing.T) {
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = true

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseAndCommitTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID1", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg := "broadcast response is not success INTERNAL_SERVER_ERROR"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}
	// registerTxEvent is true with wrongTxnID
	mockBroadcastServer.BroadcastInternalServerError = false

	//invoke transaction snap
	args = createTransactionSnapRequest("endorseAndCommitTransaction", "ccid", "testChannel", true)
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg = "SendTransaction Didn't receive tx event for txid"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}
}

func TestTransactionSnapInvokeFuncCommitTransactionSuccess(t *testing.T) {
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID1", args)
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

	txID := tpResponse[0].Proposal.TxnID.ID
	go func() {
		time.Sleep(2 * time.Second)
		eventProducer.ProduceEvent(
			mockevent.NewFilteredBlockEvent(
				channelID,
				mockevent.NewFilteredTx(txID, pb.TxValidationCode_VALID),
			),
		)
	}()

	// Call commit transaction with registerTxEvent is false
	args = make([][]byte, 4)
	args[0] = []byte("commitTransaction")
	args[1] = []byte("testChannel")
	args[2] = response.GetPayload()
	args[3] = []byte("false")
	//invoke transaction snap
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d (%s)", shim.OK, response.Status, response.Message)
	}

	// Call commit transaction with registerTxEvent is true
	args[3] = []byte("true")
	response = stub.MockInvoke("TxID3", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %v", shim.OK, response)
	}

}

func TestTransactionSnapInvokeFuncCommitTransactionReturnError(t *testing.T) {
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = true
	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := createTransactionSnapRequest("endorseTransaction", "ccid", "testChannel", false)
	//invoke transaction snap
	response := stub.MockInvoke("TxID1", args)
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
	// Call commit transaction with registerTxEvent is false
	args = make([][]byte, 4)
	args[0] = []byte("commitTransaction")
	args[1] = []byte("testChannel")
	args[2] = response.GetPayload()
	args[3] = []byte("false")
	//invoke transaction snap
	response = stub.MockInvoke("TxID2", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
	errorMsg := "broadcast response is not success INTERNAL_SERVER_ERROR"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}

	// registerTxEvent is true
	mockBroadcastServer.BroadcastInternalServerError = false
	args[3] = []byte("true")
	//invoke transaction snap
	response = stub.MockInvoke("TxID3", args)
	if response.Status != shim.ERROR {
		t.Fatalf("Expected response status %d but got %d", shim.ERROR, response.Status)
	}
	errorMsg = "SendTransaction Didn't receive tx event for txid"
	if !strings.Contains(response.Message, errorMsg) {
		t.Fatalf("Expecting error message contain(%s) but got %s", errorMsg, response.Message)
	}

}

func TestTransactionSnapInvokeFuncVerifyTxnProposalSignatureSuccess(t *testing.T) {
	//Replace client with mock client wrapper, which assumes channel is already initialized
	fcClient = mocks.GetNewClientWrapper(fcClient)
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = false
	req := apitxn.ChaincodeInvokeRequest{
		ChaincodeID: "ccID",
		Args:        nil,
		Fcn:         "fcn",
	}
	txnProposal, err := newTransactionProposal("testChannel", req, fcClient.GetUser())
	if err != nil {
		t.Fatalf("Error creating transaction proposal: %s", err)
	}

	signedProposalBytes, err := proto.Marshal(txnProposal.SignedProposal)
	if err != nil {
		t.Fatalf("Error Marshal signedProposal: %v", err)
	}

	snap := New()
	stub := shim.NewMockStub("transactionsnap", snap)
	args := make([][]byte, 3)
	args[0] = []byte("verifyTransactionProposalSignature")
	args[1] = []byte("testChannel")
	args[2] = signedProposalBytes
	//invoke transaction snap
	response := stub.MockInvoke("TxID", args)
	if response.Status != shim.OK {
		t.Fatalf("Expected response status %d but got %d", shim.OK, response.Status)
	}
}

func TestTransactionSnapInvokeFuncVerifyTxnProposalSignatureReturnError(t *testing.T) {
	//Replace client with mock client wrapper, which assumes channel is already initialized

	fcClient = mocks.GetNewClientWrapper(fcClient)
	mockEndorserServer.ProposalError = nil
	mockEndorserServer.AddkvWrite = true
	mockBroadcastServer.BroadcastInternalServerError = false
	req := apitxn.ChaincodeInvokeRequest{
		ChaincodeID: "ccID",
		Args:        nil,
		Fcn:         "fcn",
	}
	txnProposal, err := newTransactionProposal("testChannel", req, fcClient.GetUser())
	if err != nil {
		t.Fatalf("Error creating transaction proposal: %s", err)
	}
	txnProposal.SignedProposal.Signature = []byte("wrongSignature")

	signedProposalBytes, err := proto.Marshal(txnProposal.SignedProposal)
	if err != nil {
		t.Fatalf("Error Marshal signedProposal: %v", err)
	}

	snap := New()
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
		fmt.Printf("err: %v\n", err)
		return nil
	}

	var args [][]byte
	args = append(args, []byte(functionName))
	args = append(args, snapTxReqB)
	return args
}

func configureClient(config api.Config) api.Client {
	fabricClient, err := client.GetInstance("testchannel", config)
	if err != nil {
		panic(fmt.Sprintf("Error initializing fabricClient: %s", err))
	}

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

// newTransactionProposal creates a proposal for transaction. This involves assembling the proposal
// with the data (chaincodeName, function to call, arguments, transient data, etc.) and signing it using the private key corresponding to the
// ECert to sign.
func newTransactionProposal(channelID string, request apitxn.ChaincodeInvokeRequest, user sdkApi.User) (*apitxn.TransactionProposal, error) {

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
	signature, err := signObjectWithKey(proposalBytes, user.PrivateKey(),
		&bccsp.SHAOpts{}, nil, cryptoSuite)
	if err != nil {
		return nil, err
	}

	// construct the transaction proposal
	signedProposal := pbsdk.SignedProposal{ProposalBytes: proposalBytes, Signature: signature}
	sdkProposal := pbsdk.Proposal{Header: proposal.Header, Payload: proposal.Payload, Extension: proposal.Extension}
	tp := apitxn.TransactionProposal{
		TxnID:          request.TxnID,
		SignedProposal: &signedProposal,
		Proposal:       &sdkProposal,
	}

	return &tp, nil
}

// SignObjectWithKey will sign the given object with the given key,
// hashOpts and signerOpts
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

func TestMain(m *testing.M) {

	opts := &bccspFactory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &bccspFactory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &bccspFactory.FileKeystoreOpts{KeyStorePath: "./sampleconfig/msp/keystore"},
		},
	}
	bccspFactory.InitFactories(opts)

	configData, err := ioutil.ReadFile("./sampleconfig/config.yaml")
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

	_, err = config.NewConfig("./sampleconfig", channelID)
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}
	txsnapservice.PeerConfigPath = "./sampleconfig"

	txService, err := txsnapservice.Get("testChannel")
	if err != nil {
		panic(fmt.Sprintf("Error getting service: %s", err))
	}

	fcClient = txService.FcClient

	newtworkConfig, _ := txService.FcClient.GetConfig().NetworkConfig()
	newtworkConfig.Orderers["orderer.example.com"] = apiconfig.OrdererConfig{URL: broadcastTestURL}

	//create selection service
	peer, _ := sdkFabApi.NewPeer(endorserTestURL, "", "", txService.FcClient.GetConfig())
	selectionService := mocks.MockSelectionService{TestEndorsers: []sdkApi.Peer{peer},
		TestPeer:       api.PeerConfig{EventHost: endorserTestEventHost, EventPort: endorserTestEventPort},
		InvalidChannel: ""}

	txService.FcClient.SetSelectionService(&selectionService)

	mockEndorserServer = fcmocks.StartEndorserServer(endorserTestURL)
	mockBroadcastServer = fcmocks.StartMockBroadcastServer(broadcastTestURL)

	if eventProducer == nil {
		eventService, producer, err := evservice.NewServiceWithMockProducer("testChannel", []evservice.EventType{evservice.FILTEREDBLOCKEVENT}, evservice.DefaultOpts())
		localservice.Register("testChannel", eventService)
		if err != nil {
			panic(err.Error())
		}
		eventProducer = producer
	}

	clientService = newClientServiceMock()

	testChannel, err := txService.FcClient.NewChannel("testChannel")
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

	p1 = Peer("grpc://peer1:7051", org1)
	p2 = Peer("grpc://peer2:7051", org1)
	txsnapservice.DoIntializeChannel = false
	os.Exit(m.Run())
}

func Peer(url string, mspID string) sdkApi.Peer {

	peer, err := sdkFabApi.NewPeer(url, "", "", fcClient.GetConfig())
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}

	peer.SetName(url)
	peer.SetMSPID(mspID)
	fmt.Printf("CreatedPeer %s\n", url)
	return peer
}

// clientServiceMock implements client service mock
type clientServiceMock struct {
}

func newClientServiceMock() api.ClientService {
	return &clientServiceMock{}
}

// GetFabricClient return fabric client
func (cs *clientServiceMock) GetFabricClient(channelID string, config api.Config) (api.Client, error) {
	return fcClient, nil
}

// GetClientMembership return client membership
func (cs *clientServiceMock) GetClientMembership(config api.Config) api.MembershipManager {
	return membership
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
