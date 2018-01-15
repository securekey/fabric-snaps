/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"encoding/json"

	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"

	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	client "github.com/securekey/fabric-snaps/transactionsnap/cmd/client"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/txsnapservice"
)

// txServiceProvider is used by unit tests
type txServiceProvider func(channelID string) (*txsnapservice.TxServiceImpl, error)

//TxnSnap implements endorse transaction and commit transaction
type TxnSnap struct {
	// getTxService is used by unit tests
	getTxService txServiceProvider
}

// clientServiceImpl implements client service
type clientServiceImpl struct {
}

var clientService = newClientService()

var logger = logging.NewLogger("transaction-snap")

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
	function, args := stub.GetFunctionAndParameters()

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

		_, err := es.commitTransaction(stub.GetArgs())
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		//TODO QQQ Check the response code
		return pb.Response{Payload: nil, Status: shim.OK}

	case "endorseAndCommitTransaction":

		err := es.endorseAndCommitTransaction(stub.GetArgs())
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
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
	case "getPeersOfChannel":
		payload, err := es.getPeersOfChannel(args)
		if err != nil {
			logger.Errorf("getPeersOfChannel error: %s", err.Error())
			return shim.Error(err.Error())
		}
		logger.Debugf("getPeersOfChannel payload: %s", string(payload))
		return shim.Success(payload)
	default:
		return pb.Response{Payload: nil, Status: shim.ERROR, Message: fmt.Sprintf("Function %s is not supported", function)}
	}

}

// getPeersOfChannel returns peers that are available for that channel
func (es *TxnSnap) getPeersOfChannel(args []string) ([]byte, error) {

	if len(args) < 1 || args[0] == "" {

		return nil, errors.Errorf("Channel name must be provided")
	}

	// First argument is channel
	channel := args[0]
	logger.Debugf("Retrieving peers on channel: %s", channel)
	srvc, err := es.getTxService(channel)
	if err != nil {
		return nil, err
	}
	channelMembership := srvc.Membership.GetPeersOfChannel(channel)
	if channelMembership.QueryError != nil && channelMembership.Peers == nil {
		return nil, errors.Errorf("Could not get peers on channel %s: %s", channel, channelMembership.QueryError)
	}
	if channelMembership.QueryError != nil && channelMembership.Peers != nil {
		logger.Warnf(
			"Error polling peers on channel %s, using last known configuration. Error: %s",
			channelMembership.QueryError)
	}

	logger.Debugf("Peers on channel(%s): %s", channel, channelMembership.Peers)

	// Construct list of endpoints
	endpoints := make([]protosPeer.PeerEndpoint, 0, len(channelMembership.Peers))
	for _, peer := range channelMembership.Peers {
		endpoints = append(endpoints, protosPeer.PeerEndpoint{Endpoint: peer.URL(), MSPid: []byte(peer.MSPID())})
	}

	peerBytes, err := json.Marshal(endpoints)
	if err != nil {
		return nil, err
	}

	return peerBytes, nil
}

func (es *TxnSnap) endorseTransaction(args [][]byte) ([]*apitxn.TransactionProposalResponse, error) {

	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return nil, errors.New("Not enough arguments in call to endorse transaction")
	}
	//second argument is SnapTransactionRequest
	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return nil, err
	}
	if snapTxRequest.ChannelID == "" {
		return nil, errors.Errorf("ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))

	}
	logger.Debug("Endorser args:", ccargs)
	srvc, err := es.getTxService(snapTxRequest.ChannelID)
	if err != nil {
		return nil, err
	}

	tpxResponse, err := srvc.EndorseTransaction(snapTxRequest, nil)
	if err != nil {
		return nil, err
	}

	return tpxResponse, nil
}

func (es *TxnSnap) commitTransaction(args [][]byte) (pb.TxValidationCode, error) {
	if len(args) < 4 {
		return pb.TxValidationCode(-1), errors.New("Not enough arguments in call to commit transaction")
	}

	channelID := string(args[1])
	if channelID == "" {
		return pb.TxValidationCode(-1), errors.Errorf("Cannot create channel Error creating new channel: name is required")

	}

	registerTxEvent, err := strconv.ParseBool(string(args[3]))
	if err != nil {
		return pb.TxValidationCode(-1), errors.Errorf("Cannot ParseBool the fourth arg to registerTxEvent %v", err)
	}

	var tpResponses []*apitxn.TransactionProposalResponse
	if err := json.Unmarshal(args[2], &tpResponses); err != nil {
		return pb.TxValidationCode(-1), errors.Errorf("Cannot unmarshal responses")
	}
	srvc, err := es.getTxService(channelID)
	if err != nil {
		return pb.TxValidationCode(-1), err
	}
	validationCode, err := srvc.CommitTransaction(channelID, tpResponses, registerTxEvent)
	if err != nil {
		return validationCode, errors.Errorf("CommitTransaction returned error: %v", err)
	}
	return validationCode, nil
}

//endorseAndCommitTransaction returns error

func (es *TxnSnap) endorseAndCommitTransaction(args [][]byte) error {
	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return errors.New("Not enough arguments in call to endorse and commit transaction")
	}

	snapTxRequest, err := getSnapTransactionRequest(args[1])
	if err != nil {
		return err
	}

	srvc, err := es.getTxService(snapTxRequest.ChannelID)
	if err != nil {
		return err
	}

	if _, err := srvc.EndorseAndCommitTransaction(snapTxRequest, nil); err != nil {
		return errors.Errorf("CommitTransaction returned error: %v", err)
	}
	return nil
}

func (es *TxnSnap) verifyTxnProposalSignature(args [][]byte) error {
	if len(args) < 1 {
		return errors.New("Expected arg here containing channelID")
	}
	channelID := string(args[1])

	signedProposal := &pb.SignedProposal{}
	if err := proto.Unmarshal(args[2], signedProposal); err != nil {
		return err
	}
	srvc, err := es.getTxService(channelID)
	if err != nil {
		return err
	}
	err = srvc.VerifyTxnProposalSignature(channelID, signedProposal)
	if err != nil {
		return errors.Errorf("VerifyTxnProposalSignature returned error: %s", err)
	}
	return nil
}

// getSnapTransactionRequest
func getSnapTransactionRequest(snapTransactionRequestbBytes []byte) (*api.SnapTransactionRequest, error) {
	var snapTxRequest api.SnapTransactionRequest
	err := json.Unmarshal(snapTransactionRequestbBytes, &snapTxRequest)
	if err != nil {
		return nil, errors.Errorf("Cannot decode parameters from request to Snap Transaction Request %v", err)
	}
	return &snapTxRequest, nil
}

func newClientService() api.ClientService {
	return &clientServiceImpl{}
}

// GetFabricClient return fabric client
func (cs *clientServiceImpl) GetFabricClient(channelID string, config api.Config) (api.Client, error) {
	fcClient, err := client.GetInstance(channelID, config)
	if err != nil {
		return nil, errors.Errorf("Cannot initialize client %v", err)
	}
	return fcClient, nil
}

// GetClientMembership return client membership
func (cs *clientServiceImpl) GetClientMembership(config api.Config) api.MembershipManager {
	// membership mananger
	membership := client.GetMembershipInstance(config)

	return membership
}

func main() {
}
