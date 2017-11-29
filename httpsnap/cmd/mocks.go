/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func newMockStub(channelID string) *shim.MockStub {
	snap := new(HTTPSnap)
	stub := shim.NewMockStub("httpsnap", snap)
	stub.ChannelID = channelID
	return stub
}

func newConfigMockStub(channelID string) *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	return stub
}
