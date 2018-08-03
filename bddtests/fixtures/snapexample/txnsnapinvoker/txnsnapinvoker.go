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
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = shim.NewLogger("TxSnapInvoker")

// New chaincode implementation
func New() shim.Chaincode {
	return &TxnSnapInvoker{}
}

// TxnSnapInvoker demostrates how to invoke tx snap via chaincode
type TxnSnapInvoker struct {
}

// Init - nothing to do for now
func (t *TxnSnapInvoker) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke httpsnap
func (t *TxnSnapInvoker) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	args := stub.GetArgs()

	logger.Infof("TxnSnapInvoker Args=%s", args)

	if len(args) < 2 {
		return shim.Error("Missing snap name and/or snap func")
	}

	// snap name is mandatory
	snapName := string(args[0])
	if snapName == "" {
		return shim.Error("Snap name is required")
	}

	// snap func is mandatory
	snapFunc := string(args[1])
	if snapFunc == "" {
		return shim.Error("Snap func is required")
	}

	function := string(args[1])
	channelID := string(args[2])
	ccID := string(args[3])

	// Construct Snap arguments
	var ccArgs [][]byte
	ccArgs = args[1:]
	if snapFunc == "commitTransaction" || snapFunc == "endorseTransaction" {
		peerFilter := &api.PeerFilterOpts{
			Type: api.MinBlockHeightPeerFilterType,
			Args: []string{channelID, fmt.Sprintf("%d", 3)},
		}
		ccArgs = createTransactionSnapRequest(function, ccID, channelID, args[4:], true, peerFilter)
	}

	if snapFunc == "verifyTransactionProposalSignature" {
		signedProposal, err := stub.GetSignedProposal()
		if err != nil {
			return shim.Error(fmt.Sprintf("Get SignedProposal return error: %v", err))
		}
		signedProposalBytes, err := proto.Marshal(signedProposal)
		if err != nil {
			return shim.Error(fmt.Sprintf("Marshal SignedProposal return error: %v", err))
		}
		ccArgs[2] = signedProposalBytes
	}

	logger.Infof("Invoking chaincode %s with ccArgs=%s", snapName, ccArgs)

	// Leave channel (last argument) empty since we are calling chaincode(s) on the same channel
	response := stub.InvokeChaincode(snapName, ccArgs, "")
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to invoke chaincode %s. Error: %s", snapName, string(response.Message))
		logger.Warning(errStr)
		return shim.Error(errStr)
	}

	if snapFunc == "endorseTransaction" {
		var trxResponse *channel.Response
		err := json.Unmarshal(response.Payload, &trxResponse)
		if err != nil {
			return shim.Error(fmt.Sprintf("Unmarshal(%s) to TransactionProposalResponse return error: %v", response.Payload, err))
		}
		return shim.Success(trxResponse.Responses[0].ProposalResponse.GetResponse().Payload)

	}

	logger.Infof("Response from %s: %s ", snapName, string(response.Payload))

	return shim.Success(response.Payload)
}

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, clientArgs [][]byte, registerTxEvent bool, peerFilter *api.PeerFilterOpts) [][]byte {

	snapTxReq := api.SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        nil,
		EndorserArgs:        clientArgs,
		CCIDsForEndorsement: nil,
		RegisterTxEvent:     registerTxEvent,
		PeerFilter:          peerFilter}
	snapTxReqB, _ := json.Marshal(snapTxReq)

	var args [][]byte
	args = append(args, []byte(functionName))
	args = append(args, snapTxReqB)
	return args
}

func main() {
}
