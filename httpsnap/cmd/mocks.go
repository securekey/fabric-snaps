/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	mockstub "github.com/securekey/fabric-snaps/mocks/mockstub"
)

func newMockStub(channelID string, MspID string) *mockstub.MockStub { //nolint:deadcode
	snap := new(HTTPSnap)
	stub := mockstub.NewMockStub("httpsnap", snap)
	stub.ChannelID = channelID
	stub.SetMspID(MspID)
	return stub
}

func newConfigMockStub(channelID string, MspID string) *mockstub.MockStub { //nolint:deadcode
	stub := mockstub.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	stub.SetMspID(MspID)
	return stub
}
