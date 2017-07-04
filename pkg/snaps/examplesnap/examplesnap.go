/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package examplesnap

import (
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	logging "github.com/op/go-logging"
)

var logger = logging.MustGetLogger("example-snap")

// ExampleSnap is a sample Snap that simply echos the supplied argument in the response
type ExampleSnap struct {
	name string
}

// NewSnap - create new instance of snap
func NewSnap() shim.Chaincode {
	return &ExampleSnap{}
}

// Invoke snap
// arg[0] - Some message (optional)
func (es *ExampleSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("Snap[%s] - Invoked", es.name)

	args := stub.GetArgs()
	if len(args) > 0 {
		return shim.Success([]byte(args[0]))
	}
	return shim.Success([]byte("Example snap invoked!"))
}

// Init initializes the snap
// arg[0] - Snap name
func (es *ExampleSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetStringArgs()
	if len(args) == 0 {
		return shim.Error("Expecting snap name as first arg")
	}

	es.name = args[0]

	logger.Infof("Snap[%s] - Initialized", es.name)

	return shim.Success(nil)
}
