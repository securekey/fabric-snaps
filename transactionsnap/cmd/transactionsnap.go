/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"encoding/json"

	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice/dbprovider"
	"github.com/securekey/fabric-snaps/util/errors"
)

// txServiceProvider is used by unit tests
type txServiceProvider func(channelID string) (*txsnapservice.TxServiceImpl, error)

//TxnSnap implements endorse transaction and commit transaction
type TxnSnap struct {
	// getTxService is used by unit tests
	getTxService txServiceProvider
}

var logger = logging.NewLogger("txnsnap")

// New chaincode implementation
func New() shim.Chaincode {
	return &TxnSnap{getTxService: func(channelID string) (*txsnapservice.TxServiceImpl, error) {
		return txsnapservice.Get(channelID)
	}}
}

// Init snap
func (es *TxnSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {

	return shim.Success(nil)
}

//Invoke transaction snap
//required args are function name and SnapTransactionRequest
func (es *TxnSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	//service will be used to endorse and commit transaction
	function, _ := stub.GetFunctionAndParameters()

	switch function {
	case "endorseTransaction":

		tpResponses, err := es.endorseTransaction(stub.GetArgs())
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		payload, err := json.Marshal(tpResponses)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		return pb.Response{Payload: payload, Status: shim.OK}
	case "commitTransaction":

		err := es.commitTransaction(stub.GetArgs())
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		//TODO QQQ Check the response code
		return pb.Response{Payload: nil, Status: shim.OK}

	case "verifyTransactionProposalSignature":

		args := stub.GetArgs()
		if len(args) < 3 {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: "Not enough arguments in call to verify transaction proposal signature"}
		}

		if err := es.verifyTxnProposalSignature(args); err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		return pb.Response{Payload: nil, Status: shim.OK}

	case "unsafeGetState":
		args := stub.GetArgs()
		logger.Debugf("Function unsafeGetState invoked with args %v", args)
		resp, err := es.unsafeGetState(args)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		return pb.Response{Payload: resp, Status: shim.OK}

	default:
		return pb.Response{Payload: nil, Status: shim.ERROR, Message: fmt.Sprintf("Function %s is not supported", function)}
	}

}

func (es *TxnSnap) endorseTransaction(args [][]byte) (*channel.Response, error) {

	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return nil, errors.New(errors.GeneralError, "Not enough arguments in call to endorse transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return nil, err
	}
	if snapTxRequest.ChannelID == "" {
		return nil, errors.New(errors.GeneralError, "ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))

	}
	logger.Debugf("Endorser args: %s", ccargs)
	srvc, err := es.getTxService(snapTxRequest.ChannelID)
	if err != nil {
		return nil, err
	}

	response, err := srvc.EndorseTransaction(snapTxRequest, nil)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (es *TxnSnap) commitTransaction(args [][]byte) error {

	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return errors.New(errors.GeneralError, "Not enough arguments in call to commit transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return err
	}
	if snapTxRequest.ChannelID == "" {
		return errors.New(errors.GeneralError, "ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))

	}
	logger.Debugf("Endorser args: %s", ccargs)
	srvc, err := es.getTxService(snapTxRequest.ChannelID)
	if err != nil {
		return err
	}

	_, err = srvc.CommitTransaction(snapTxRequest, nil)
	if err != nil {
		return err
	}

	return nil
}

func (es *TxnSnap) verifyTxnProposalSignature(args [][]byte) error {
	if len(args) < 1 {
		return errors.New(errors.GeneralError, "Expected arg here containing channelID")
	}
	channelID := string(args[1])

	signedProposal := &pb.SignedProposal{}
	if err := proto.Unmarshal(args[2], signedProposal); err != nil {
		return errors.Wrap(errors.GeneralError, err, "Failed Unmarshal signedProposal")
	}
	srvc, err := es.getTxService(channelID)
	if err != nil {
		return err
	}
	err = srvc.VerifyTxnProposalSignature(signedProposal)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "VerifyTxnProposalSignature returned error")
	}
	return nil
}

// unsafeGetState allows the caller to read a given key from the stateDB without
// producing a read set.
// Function name: unsafeGetState, Arguments: channelID, ccID, key
func (es *TxnSnap) unsafeGetState(args [][]byte) ([]byte, error) {
	if len(args) < 4 {
		return nil, errors.New(errors.GeneralError,
			"unsafeGetState requires function and three args: channelID, ccID, key")
	}

	channelID := string(args[1])
	ccNamespace := string(args[2])
	key := string(args[3])

	db, err := dbprovider.GetStateDB(channelID)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed to get State DB")
	}

	err = db.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	defer logger.Debug("DB handle closed")

	logger.Debug("DB handle opened")

	vv, err := db.GetState(ccNamespace, key)
	if err != nil {
		return nil, err
	}

	if vv == nil {
		logger.Debugf("Query returned nil for namespace %s and key %s", ccNamespace, key)
		return nil, nil
	}

	logger.Debugf("Query returned %+v for namespace %s and key %s", vv.Value, ccNamespace, key)

	return vv.Value, nil
}

// getSnapTransactionRequest
func getSnapTransactionRequest(snapTransactionRequestbBytes []byte) (*api.SnapTransactionRequest, error) {
	var snapTxRequest api.SnapTransactionRequest
	err := json.Unmarshal(snapTransactionRequestbBytes, &snapTxRequest)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Cannot decode parameters from request to Snap Transaction Request")
	}
	return &snapTxRequest, nil
}

func main() {
}
