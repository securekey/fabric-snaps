/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = logging.NewLogger("eventsnap")

// eventSnap no longer supported.
// TODO: Remove all code related to event snap
type eventSnap struct {
}

// New returns a new Event Snap
func New() shim.Chaincode {
	return &eventSnap{}
}

// Init initializes the Event Snap.
func (s *eventSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Errorf("******** Event Snap no longer support")
	return shim.Error("event snap no longer supported")
}

// Invoke isn't implemented for this snap.
func (s *eventSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("not implemented")
}

func main() {
}
