/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package transactionsnap

import (
	"encoding/json"
	"errors"
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api"
	logging "github.com/op/go-logging"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	client "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/client"
)

// TxSnapImpl implements endorse transaction and commit transaction
type TxSnapImpl struct {
}

//SnapTransactionRequest type will be passed as argument to a transaction snap
//ChannelID and ChaincodeID are mandatory fields
type SnapTransactionRequest struct {
	ChannelID       string
	ChaincodeID     string
	TransientMap    map[string][]byte
	EndorserArgs    [][]byte
	AdditionalCCIDs []string
}

var logger = logging.MustGetLogger("transaction-snap")
var fcClient client.Client

// NewSnap - create new instance of snap
func NewSnap() shim.Chaincode {
	return &TxSnapImpl{}
}

//Invoke transaction snap
//required args are function name and SnapTransactionRequest
func (es *TxSnapImpl) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, _ := stub.GetFunctionAndParameters()

	switch function {
	case "endorseTransaction":

		tpResponse, err := endorseTransaction(stub)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		payload, err := json.Marshal(tpResponse)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}

		return pb.Response{Payload: payload, Status: shim.OK}
	default:
		return pb.Response{Payload: nil, Status: shim.ERROR, Message: fmt.Sprintf("Function %s is not supported", function)}
	}

}

//endorseTransaction returns []*sdkApi.TransactionProposalResponse
func endorseTransaction(stub shim.ChaincodeStubInterface) ([]*sdkApi.TransactionProposalResponse, error) {

	args := stub.GetArgs()
	//first arg is function name; the second one is SnapTransactionRequest
	if len(args) < 2 {
		return nil, errors.New("Not enough arguments in call to endorse transaction")
	}
	//second argument is SnapTransactionRequest
	var snapTxRequest SnapTransactionRequest
	err := json.Unmarshal(args[1], &snapTxRequest)
	if err != nil {
		return nil, fmt.Errorf("Cannot decode parameters from request to endorse transaction %v", err)
	}
	if snapTxRequest.ChaincodeID == "" {
		return nil, fmt.Errorf("ChaincodeID is mandatory field of the SnapTransactionRequest")
	}

	if fcClient == nil {
		fcClient, err = GetInstanceOfFabricClient()
		if err != nil {
			return nil, fmt.Errorf("Cannot initialize client %v", err)
		}
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
		ccargs, snapTxRequest.TransientMap, nil)
	if err != nil {
		return nil, err
	}

	return tpxResponse, nil
}

// Init snap
func (es *TxSnapImpl) Init(stub shim.ChaincodeStubInterface) pb.Response {
	//initialize fabric client
	GetInstanceOfFabricClient()
	response := pb.Response{Status: shim.OK}
	return response
}

//
func GetInstanceOfFabricClient() (client.Client, error) {
	fabricClient, err := client.GetInstance()
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize client %v", err)
	}
	fcClient = fabricClient
	return fcClient, nil
}
