/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	logging "github.com/op/go-logging"

	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	client "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/client"
)

// The newTxID is added so the unit test can access the new transaction id generated in transactionsnap
var newTxID apitxn.TransactionID

var registerTxEventTimeout time.Duration = 30

// TxnSnap implements endorse transaction and commit transaction
type TxnSnap struct {
}

//SnapTransactionRequest type will be passed as argument to a transaction snap
//ChannelID and ChaincodeID are mandatory fields
type SnapTransactionRequest struct {
	ChannelID           string            // required channel ID
	ChaincodeID         string            // required chaincode ID
	TransientMap        map[string][]byte // optional transient Map
	EndorserArgs        [][]byte          // optional args for endorsement
	CCIDsForEndorsement []string          // optional ccIDs For endorsement selection
	RegisterTxEvent     bool              // optional args for register Tx event (default is false)
}

var logger = logging.MustGetLogger("transaction-snap")
var fcClient client.Client

// Init snap
func (es *TxnSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	//initialize fabric client
	err := getInstanceOfFabricClient()
	response := pb.Response{Status: shim.OK}
	if err != nil {
		response = pb.Response{Status: shim.ERROR, Message: fmt.Sprintf("getInstanceOfFabricClient return error %s", err.Error())}
	}
	return response
}

//Invoke transaction snap
//required args are function name and SnapTransactionRequest
func (es *TxnSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()

	switch function {
	case "endorseTransaction":

		tpResponses, err := endorseTransaction(stub)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		payload, err := json.Marshal(tpResponses)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}

		return pb.Response{Payload: payload, Status: shim.OK}
	case "commitTransaction":
		err := commitTransaction(stub)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		return pb.Response{Payload: nil, Status: shim.OK}
	case "endorseAndCommitTransaction":
		err := endorseAndCommitTransaction(stub)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		return pb.Response{Payload: nil, Status: shim.OK}
	case "verifyTransactionProposalSignature":
		err := verifyTxnProposalSignature(stub)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		return pb.Response{Payload: nil, Status: shim.OK}
	default:
		return pb.Response{Payload: nil, Status: shim.ERROR, Message: fmt.Sprintf("Function %s is not supported", function)}
	}

}

//endorseTransaction returns []*sdkApi.TransactionProposalResponse
func endorseTransaction(stub shim.ChaincodeStubInterface) ([]*apitxn.TransactionProposalResponse, error) {

	args := stub.GetArgs()
	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return nil, errors.New("Not enough arguments in call to endorse transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return nil, err
	}
	if snapTxRequest.ChaincodeID == "" {
		return nil, fmt.Errorf("ChaincodeID is mandatory field of the SnapTransactionRequest")
	}
	channel, err := fcClient.NewChannel(snapTxRequest.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("Cannot create channel %v", err)
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))
	}
	logger.Debug("Endorser args:", ccargs)

	tpxResponse, err := fcClient.EndorseTransaction(channel, snapTxRequest.ChaincodeID,
		ccargs, snapTxRequest.TransientMap, nil, snapTxRequest.CCIDsForEndorsement)
	if err != nil {
		return nil, err
	}

	return tpxResponse, nil
}

//commitTransaction returns error
func commitTransaction(stub shim.ChaincodeStubInterface) error {
	args := stub.GetArgs()
	//first arg is function name; the second one is channel name; the third one is tpResponses;
	//the fourth one is registerTxEvent
	if len(args) < 4 {
		return errors.New("Not enough arguments in call to commit transaction")
	}

	channel, err := fcClient.NewChannel(string(args[1]))
	if err != nil {
		return fmt.Errorf("Cannot create channel %v", err)
	}
	var tpResponses []*apitxn.TransactionProposalResponse
	json.Unmarshal(args[2], &tpResponses)
	registerTxEvent, err := strconv.ParseBool(string(args[3]))
	if err != nil {
		return fmt.Errorf("Cannot ParseBool the fourth arg to registerTxEvent %v", err)
	}
	err = fcClient.CommitTransaction(channel, tpResponses, registerTxEvent, registerTxEventTimeout)

	if err != nil {
		return fmt.Errorf("CommitTransaction returned error: %v", err)
	}
	return nil
}

//endorseAndCommitTransaction returns error
func endorseAndCommitTransaction(stub shim.ChaincodeStubInterface) error {
	args := stub.GetArgs()
	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return errors.New("Not enough arguments in call to endorse and commit transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return err
	}

	tpxResponse, err := endorseTransaction(stub)
	if err != nil {
		return err
	}
	newTxID = tpxResponse[0].Proposal.TxnID

	// Channel already checked in endorseTransaction
	channel, _ := fcClient.NewChannel(snapTxRequest.ChannelID)
	err = fcClient.CommitTransaction(channel, tpxResponse, snapTxRequest.RegisterTxEvent, registerTxEventTimeout)

	if err != nil {
		return fmt.Errorf("CommitTransaction returned error: %v", err)
	}
	return nil
}

//verifyTxnProposalSignature returns error
func verifyTxnProposalSignature(stub shim.ChaincodeStubInterface) error {
	args := stub.GetArgs()
	//first arg is function name; the second one is channel name; the third one is TxnProposalBytes
	if len(args) < 3 {
		return errors.New("Not enough arguments in call to verify transaction proposal signature")
	}
	channel, err := fcClient.NewChannel(string(args[1]))
	if err != nil {
		return fmt.Errorf("Cannot create channel %v", err)
	}
	err = fcClient.VerifyTxnProposalSignature(channel, args[2])
	if err != nil {
		return fmt.Errorf("VerifyTxnProposalSignature returned error: %v", err)
	}
	return nil
}

// getInstanceOfFabricClient
func getInstanceOfFabricClient() error {
	var err error
	fcClient, err = client.GetInstance()
	if err != nil {
		return fmt.Errorf("Cannot initialize client %v", err)
	}
	return nil
}

// getSnapTransactionRequest
func getSnapTransactionRequest(snapTransactionRequestbBytes []byte) (*SnapTransactionRequest, error) {
	var snapTxRequest SnapTransactionRequest
	err := json.Unmarshal(snapTransactionRequestbBytes, &snapTxRequest)
	if err != nil {
		return nil, fmt.Errorf("Cannot decode parameters from request to Snap Transaction Request %v", err)
	}
	return &snapTxRequest, nil
}

func main() {
	err := shim.Start(new(TxnSnap))
	if err != nil {
		fmt.Printf("Error starting Txn snap: %s", err)
	}
}
