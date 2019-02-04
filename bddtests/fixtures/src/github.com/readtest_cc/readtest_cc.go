/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("ReadTest_cc")

const (
	// TransactionSnap name
	TransactionSnap = "txnsnap"
	// QueryFunc query function
	QueryFunc = "unsafeGetState"
)

// ReadTest demostrates how to perform an unsafe read via chaincode
type ReadTest struct {
}

// Init - nothing to do for now
func (t *ReadTest) Init(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4")
	}

	// Initialize the chaincode
	A := args[0]
	Aval := string(args[1])
	B := args[2]
	Bval := string(args[3])

	logger.Infof("Aval = %d, Bval = %d\n", Aval, Bval)

	// Write the state to the ledger
	err := stub.PutState(A, []byte(Aval))
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(B, []byte(Bval))
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

// Invoke demonstrates unsafeGetState functionality on the transaction snap
// It supports one function: concat
// Required args are: channelID, ccID, key1, key2, key3
// We will perform an unsafe read on key1, key2, concatenate the result and
// store it in key3
// The response will contain a concatenated string of the values that were read (corresponding to key1, key2)
func (t *ReadTest) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	logger.Infof("ReadTest Args=%s", args)

	if function == "concat" {
		if len(args) < 5 {
			return shim.Error("Required args are: concat, channelID, ccID, key1, key2, key3")
		}
		channelID := string(args[0])
		ccID := string(args[1])
		key1 := string(args[2])
		key2 := string(args[3])
		key3 := string(args[4])

		// The unsafeGetState on the transaction snap can be invoked with or without a channel.
		resp1 := stub.InvokeChaincode(TransactionSnap, [][]byte{[]byte(QueryFunc), []byte(channelID), []byte(ccID), []byte(key1)}, channelID)
		resp2 := stub.InvokeChaincode(TransactionSnap, [][]byte{[]byte(QueryFunc), []byte(channelID), []byte(ccID), []byte(key2)}, "")
		resp3 := stub.InvokeChaincode(TransactionSnap, [][]byte{[]byte(QueryFunc), []byte(channelID), []byte(ccID), []byte("invalidKey")}, "")

		if resp1.GetStatus() != 200 {
			return shim.Error("Query on key1 failed: " + resp1.GetMessage())
		}
		if resp2.GetStatus() != 200 {
			return shim.Error("Query on key2 failed: " + resp2.GetMessage())
		}
		if resp3.GetStatus() != 200 {
			return shim.Error("Query on invalid key failed: " + resp2.GetMessage())
		}

		logger.Infof("Response from %s for key1 : %s ", TransactionSnap, string(resp1.Payload))
		logger.Infof("Response from %s for key2 : %s ", TransactionSnap, string(resp2.Payload))

		v3 := strings.Join([]string{string(resp1.Payload), string(resp2.Payload)}, "")
		err := stub.PutState(key3, []byte(v3))
		if err != nil {
			return shim.Error(fmt.Sprintf("PutState failed: key %s, value %s, error: %s", key3, v3, err.Error()))
		}

		return shim.Success([]byte(v3))
	}

	if function == "put" {
		if len(args) < 2 {
			return shim.Error("Required args are: key,value")
		}
		key := string(args[0])
		value := []byte(args[1])
		stub.PutState(key, value)
		return shim.Success([]byte(""))
	}
	if function == "checkGetStateAndUnsafeGetState" {
		if len(args) < 1 {
			return shim.Error("Required args are: channelID, ccID, key")
		}
		channelID := string(args[0])
		ccID := string(args[1])
		key := string(args[2])
		value, err := stub.GetState(key)
		if err != nil {
			return shim.Error(err.Error())
		}
		resp1 := stub.InvokeChaincode(TransactionSnap, [][]byte{[]byte(QueryFunc), []byte(channelID), []byte(ccID), []byte(key)}, channelID)
		if resp1.GetStatus() != 200 {
			return shim.Error(fmt.Sprintf("Query on %s failed: %s", key, resp1.GetMessage()))
		}
		if string(value) != string(resp1.Payload) {
			return shim.Error(fmt.Sprintf("unsafeGetState return different value from GetState %s %s", value, resp1.Payload))
		}
		return shim.Success([]byte(""))
	}

	return shim.Error("Only three functions is supported: concat, put, get")

}

func main() {
	err := shim.Start(new(ReadTest))
	if err != nil {
		fmt.Printf("Error starting HttpSnapTest: %s", err)
	}
}
