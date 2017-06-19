/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package interfaces

import (
	pb "github.com/hyperledger/fabric/protos/peer"
)

//Snap interface
type Snap interface {
	//Init snap
	Init(stub SnapStubInterface) pb.Response
	//Invoke snap
	Invoke(stub SnapStubInterface) pb.Response
}

//SnapStubInterface ...
type SnapStubInterface interface {
	//TODO - add required methods
	// Get the arguments to the stub call as a 2D byte array
	GetArgs() [][]byte

	// Get the arguments to the stub call as a string array
	GetStringArgs() []string

	// Get the function which is the first argument and the rest of the arguments
	// as parameters
	GetFunctionAndParameters() (string, []string)
}
