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
	"github.com/securekey/fabric-snaps/util"
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
func (es *TxnSnap) Invoke(stub shim.ChaincodeStubInterface) (resp pb.Response) {

	defer util.HandlePanic(&resp, logger, stub)
	//service will be used to endorse and commit transaction
	function, _ := stub.GetFunctionAndParameters()

	if function == "endorseTransaction" {
		tpResponses, err := es.endorseTransaction(stub.GetArgs())
		if err != nil {
			return util.CreateShimResponseFromError(err, logger, stub)

		}
		payload, e := json.Marshal(tpResponses)
		if e != nil {
			return util.CreateShimResponseFromError(errors.WithMessage(errors.SystemError, e, "Error marshalling endorsment responses"), logger, stub)
		}
		return pb.Response{Payload: payload, Status: shim.OK}
	}
	if function == "commitTransaction" {
		err := es.commitTransaction(stub.GetArgs())
		if err != nil {
			return util.CreateShimResponseFromError(err, logger, stub)
		} //TODO QQQ Check the response code
		return pb.Response{Payload: nil, Status: shim.OK}
	}
	if function == "verifyTransactionProposalSignature" {
		args := stub.GetArgs()
		es.ValidateTransactionProposalLength(args, stub)
		if err := es.verifyTxnProposalSignature(args); err != nil {
			return util.CreateShimResponseFromError(err, logger, stub)
		}
		return pb.Response{Payload: nil, Status: shim.OK}
	}
	if function == "unsafeGetState" {
		args := stub.GetArgs()
		logger.Debugf("Function unsafeGetState invoked with args %v", args)
		resp, err := es.unsafeGetState(args)
		if err != nil {
			return util.CreateShimResponseFromError(err, logger, stub)

		}
		return pb.Response{Payload: resp, Status: shim.OK}
	}
	return util.CreateShimResponseFromError(errors.New(errors.InvalidFunctionError, fmt.Sprintf("Function %s is not supported", function)), logger, stub)

}

// ValidateTransactionProposalLength - To Validate if the Transaction Proposal Length is less than 3
func (es *TxnSnap) ValidateTransactionProposalLength(args [][]byte, stub shim.ChaincodeStubInterface) (resp pb.Response) {
	if len(args) < 3 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Not enough arguments in call to verify transaction proposal signature"), logger, stub)
	}
	return
}
func (es *TxnSnap) verifyTransactionProposalSignature(stub shim.ChaincodeStubInterface) (resp pb.Response) {
	args := stub.GetArgs()
	es.ValidateTransactionProposalLength(args, stub)
	if err := es.verifyTxnProposalSignature(args); err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}
	return pb.Response{Payload: nil, Status: shim.OK}
}
func (es *TxnSnap) transactionCommit(stub shim.ChaincodeStubInterface) (resp pb.Response) {
	err := es.commitTransaction(stub.GetArgs())
	if err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	} //TODO QQQ Check the response code
	return pb.Response{Payload: nil, Status: shim.OK}
}
func (es *TxnSnap) endorseTransaction(args [][]byte) (*channel.Response, errors.Error) {

	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return nil, errors.New(errors.MissingRequiredParameterError, "Not enough arguments in call to endorse transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return nil, err
	}
	if snapTxRequest.ChannelID == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))

	}
	logger.Debugf("Endorser args: %s", ccargs)
	srvc, e := es.getTxService(snapTxRequest.ChannelID)
	if e != nil {
		return nil, errors.WithMessage(errors.GetTxServiceError, e, fmt.Sprintf("Failed to get TxService for channelID %s", snapTxRequest.ChannelID))
	}

	response, err := srvc.EndorseTransaction(snapTxRequest, nil)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (es *TxnSnap) commitTransaction(args [][]byte) errors.Error {

	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return errors.New(errors.MissingRequiredParameterError, "Not enough arguments in call to commit transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return err
	}
	if snapTxRequest.ChannelID == "" {
		return errors.New(errors.MissingRequiredParameterError, "ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))

	}
	logger.Debugf("Endorser args: %s", ccargs)
	srvc, e := es.getTxService(snapTxRequest.ChannelID)
	if e != nil {
		return errors.WithMessage(errors.GetTxServiceError, e, fmt.Sprintf("Failed to get TxService for channelID %s", snapTxRequest.ChannelID))
	}

	_, err = srvc.CommitTransaction(snapTxRequest, nil)
	if err != nil {
		return err
	}

	return nil
}

func (es *TxnSnap) verifyTxnProposalSignature(args [][]byte) errors.Error {
	if len(args) < 1 {
		return errors.New(errors.MissingRequiredParameterError, "Expected arg here containing channelID")
	}
	channelID := string(args[1])

	signedProposal := &pb.SignedProposal{}
	if err := proto.Unmarshal(args[2], signedProposal); err != nil {
		return errors.Wrap(errors.UnmarshalError, err, "Failed Unmarshal signedProposal")
	}
	srvc, e := es.getTxService(channelID)
	if e != nil {
		return errors.WithMessage(errors.GetTxServiceError, e, fmt.Sprintf("Failed to get TxService for channelID %s", channelID))
	}

	err := srvc.VerifyTxnProposalSignature(signedProposal)
	if err != nil {
		return err
	}
	return nil
}

// unsafeGetState allows the caller to read a given key from the stateDB without
// producing a read set.
// Function name: unsafeGetState, Arguments: channelID, ccID, key
func (es *TxnSnap) unsafeGetState(args [][]byte) ([]byte, errors.Error) {
	if len(args) < 4 {
		return nil, errors.New(errors.MissingRequiredParameterError,
			"unsafeGetState requires function and three args: channelID, ccID, key")
	}

	channelID := string(args[1])
	ccNamespace := string(args[2])
	key := string(args[3])

	db, err := dbprovider.GetStateDB(channelID)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to get State DB")
	}

	err = db.Open()
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to open State DB")
	}
	defer db.Close()
	defer logger.Debug("DB handle closed")

	logger.Debug("DB handle opened")

	vv, err := db.GetState(ccNamespace, key)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Failed to get state")
	}

	if vv == nil {
		logger.Debugf("Query returned nil for namespace %s and key %s", ccNamespace, key)
		return nil, nil
	}

	logger.Debugf("Query returned %+v for namespace %s and key %s", vv.Value, ccNamespace, key)

	return vv.Value, nil
}

// getSnapTransactionRequest
func getSnapTransactionRequest(snapTransactionRequestbBytes []byte) (*api.SnapTransactionRequest, errors.Error) {
	var snapTxRequest api.SnapTransactionRequest
	err := json.Unmarshal(snapTransactionRequestbBytes, &snapTxRequest)
	if err != nil {
		return nil, errors.WithMessage(errors.UnmarshalError, err, "Cannot decode parameters from request to Snap Transaction Request")
	}
	return &snapTxRequest, nil
}

func main() {
}
