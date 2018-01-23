/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("HttpSnapTest_cc")

// HTTPSnapTest demostrates how to invoke http snap via chaincode
type HTTPSnapTest struct {
}

//HTTPSnapRequest is used to invoke http snap
type HTTPSnapRequest struct {
	URL         string            // required
	Headers     map[string]string // required
	Body        string            // required
	NamedClient string            // optional
	PinSet      []string          // optional
}

// Init - nothing to do for now
func (t *HTTPSnapTest) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke httpsnap
func (t *HTTPSnapTest) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	args := stub.GetArgs()

	logger.Infof("HttpSnapTest Args=%s", args)

	if len(args) < 2 {
		return shim.Error("Missing snap name and/or url")
	}

	// snap name is mandatory
	snapName := string(args[0])
	if snapName == "" {
		return shim.Error("Snap name is required")
	}

	// url is mandatory
	url := string(args[1])
	if url == "" {
		return shim.Error("Url is required")
	}

	headers := map[string]string{
		"content-type":  "application/json",
		"authorization": "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
	}

	req := HTTPSnapRequest{URL: url, Headers: headers, Body: `{"id":"123", "name": "Test Name"}`}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return shim.Error(err.Error())
	}

	// Construct Snap arguments
	var ccArgs [][]byte
	ccArgs = append(ccArgs, []byte("invoke")) // function
	ccArgs = append(ccArgs, reqBytes)         // http snap request

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

func main() {
	err := shim.Start(new(HTTPSnapTest))
	if err != nil {
		fmt.Printf("Error starting HttpSnapTest: %s", err)
	}
}
