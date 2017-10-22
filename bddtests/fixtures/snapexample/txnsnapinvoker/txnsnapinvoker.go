/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"

	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/internal/github.com/hyperledger/fabric/core/chaincode/shim"
)

var logger = shim.NewLogger("TxSnapInvoker")

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

	// Construct Snap arguments
	var ccArgs [][]byte
	ccArgs = args[1:]
	if snapFunc == "endorseAndCommitTransaction" || snapFunc == "endorseTransaction" {
		ccArgs = createTransactionSnapRequest(string(args[1]), string(args[3]), string(args[2]), args[4:], true)
	}

	logger.Infof("Invoking chaincode %s with ccArgs=%s", snapName, ccArgs)

	// Leave channel (last argument) empty since we are calling chaincode(s) on the same channel
	response := stub.InvokeChaincode(snapName, ccArgs, "")
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to invoke chaincode %s. Error: %s", snapName, string(response.Message))
		logger.Warning(errStr)
		return shim.Error(errStr)
	}

	logger.Infof("Response from %s: %s ", snapName, string(response.Payload))

	return shim.Success(response.Payload)
}

func createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, clientArgs [][]byte, registerTxEvent bool) [][]byte {

	snapTxReq := SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        nil,
		EndorserArgs:        clientArgs,
		CCIDsForEndorsement: nil,
		RegisterTxEvent:     registerTxEvent}
	snapTxReqB, _ := json.Marshal(snapTxReq)

	var args [][]byte
	args = append(args, []byte(functionName))
	args = append(args, snapTxReqB)
	return args
}

func main() {
	err := shim.Start(new(TxnSnapInvoker))
	if err != nil {
		fmt.Printf("Error starting TxnSnapInvoker: %s", err)
	}
}
