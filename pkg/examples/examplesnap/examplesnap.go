/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package examplesnap

import (
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//SnapImpl ...
type CCSnapImpl struct {
}

// NewSnap - create new instance of snap
func NewSnap() shim.Chaincode {
	return &CCSnapImpl{}
}

// Invoke snap
func (es *CCSnapImpl) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	response := pb.Response{Payload: args[0]}
	return response
}

// Init snap
func (es *CCSnapImpl) Init(stub shim.ChaincodeStubInterface) pb.Response {
	responsePayload := []byte("Hello from Init")
	response := pb.Response{Payload: responsePayload}
	return response
}
