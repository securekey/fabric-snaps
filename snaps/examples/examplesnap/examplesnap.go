/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package examplesnap

import (
	pb "github.com/hyperledger/fabric/protos/peer"
	snap_interfaces "github.com/securekey/fabric-snaps/snaps/interfaces"
)

//SnapImpl ...
type SnapImpl struct {
}

// NewSnap - create new instance of snap
func NewSnap() snap_interfaces.Snap {
	return &SnapImpl{}
}

// Invoke snap
func (es *SnapImpl) Invoke(stub snap_interfaces.SnapStubInterface) pb.Response {
	responsePayload := []byte("Hello from invoke")
	response := pb.Response{Payload: responsePayload}
	return response
}

// Init snap
func (es *SnapImpl) Init(stub snap_interfaces.SnapStubInterface) pb.Response {
	responsePayload := []byte("Hello from Init")
	response := pb.Response{Payload: responsePayload}
	return response
}
