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
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/initbcinfo"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice/dbprovider"
	"github.com/securekey/fabric-snaps/util"
	"github.com/securekey/fabric-snaps/util/bcinfo"
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

type bcInfoProvider interface {
	GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error)
}

// bcInfoProvider can be modified by unit test
var ledgerBCInfoProvider bcInfoProvider = bcinfo.NewProvider()

// New chaincode implementation
func New() shim.Chaincode {
	return &TxnSnap{getTxService: func(channelID string) (*txsnapservice.TxServiceImpl, error) {
		return txsnapservice.Get(channelID)
	}}
}

// Init snap
func (es *TxnSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	channelID := stub.GetChannelID()
	if channelID != "" {
		logger.Debugf("Getting local blockchain info for channel [%s]", channelID)
		bcInfo, err := ledgerBCInfoProvider.GetBlockchainInfo(channelID)
		if err != nil {
			panic("unable to get blockchain info: " + err.Error())
		}

		logger.Infof("Setting initial blockchain info for channel [%s]: %#v", channelID, bcInfo)
		err = initbcinfo.Set(channelID, bcInfo)
		if err != nil {
			panic("unable to set initial blockchain info: " + err.Error())
		}
	}
	return shim.Success(nil)
}

//Invoke transaction snap
//required args are function name and SnapTransactionRequest
func (es *TxnSnap) Invoke(stub shim.ChaincodeStubInterface) (resp pb.Response) {

	defer util.HandlePanic(&resp, logger, stub)
	//service will be used to endorse and commit transaction
	function, _ := stub.GetFunctionAndParameters()

	switch function {
	case "endorseTransaction":
		return es.invokeEndorseTransaction(stub)
	case "commitTransaction":
		return es.invokeCommitTransaction(stub)
	case "verifyTransactionProposalSignature":
		return es.invokeVerifyTransactionProposalSignature(stub)
	case "unsafeGetState":
		return es.invokeUnsafeGetState(stub)
	default:
		return util.CreateShimResponseFromError(errors.New(errors.InvalidFunctionError, fmt.Sprintf("Function %s is not supported", function)), logger, stub)
	}
}

func (es *TxnSnap) invokeEndorseTransaction(stub shim.ChaincodeStubInterface) pb.Response {
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
func (es *TxnSnap) invokeCommitTransaction(stub shim.ChaincodeStubInterface) pb.Response {
	err := es.commitTransaction(stub.GetArgs())
	if err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	} //TODO QQQ Check the response code
	return pb.Response{Payload: nil, Status: shim.OK}
}

func (es *TxnSnap) invokeVerifyTransactionProposalSignature(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 3 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Not enough arguments in call to verify transaction proposal signature"), logger, stub)
	}
	if err := es.verifyTxnProposalSignature(args); err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}
	return pb.Response{Payload: nil, Status: shim.OK}
}
func (es *TxnSnap) invokeUnsafeGetState(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	logger.Debugf("Function unsafeGetState invoked with args %v", args)
	resp, err := es.unsafeGetState(args)
	if err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}
	return pb.Response{Payload: resp, Status: shim.OK}
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

	_, _, err = srvc.CommitTransaction(snapTxRequest, nil)
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
