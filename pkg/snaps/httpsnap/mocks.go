/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnap

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func newMockStub() *shim.MockStub {
	snap := new(CCSnapImpl)
	return shim.NewMockStub("httpsnap", snap)
}
