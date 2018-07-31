/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"container/list"
	"github.com/golang/protobuf/proto"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/op/go-logging"
)

// Logger for the shim package.
var mockLogger = logging.MustGetLogger("mock")

const (
	minUnicodeRuneValue = 0 //U+0000
)

//MspID peers mspid
var MspID string

// MockStub is an implementation of ChaincodeStubInterface for unit testing chaincode.
// Use this instead of ChaincodeStub in your chaincode's unit test calls to Init or Invoke.
type MockStub struct {
	shim.MockStub
	// mocked signedProposal
	signedProposal *pb.SignedProposal
	// arguments the stub was called with
	args [][]byte
	// A pointer back to the chaincode that will invoke this, set by constructor.
	// If a peer calls this stub, the chaincode will be invoked from here.
	cc shim.Chaincode
	// registered list of other MockStub chaincodes that can be called from this MockStub
	Invokables map[string]*MockStub
}

//SetMspID to set mspid
func (stub *MockStub) SetMspID(mspid string) {
	MspID = mspid

}

//GetCreator to get creator bytes
func (stub *MockStub) GetCreator() ([]byte, error) {
	sid := &msp.SerializedIdentity{Mspid: MspID}
	b, err := proto.Marshal(sid)
	if err != nil {
		return nil, err
	}
	return b, nil
}

//NewMockStub Constructor to initialise the internal State map
func NewMockStub(name string, cc shim.Chaincode) *MockStub {
	mockLogger.Debug("MockStub(", name, cc, ")")
	s := new(MockStub)
	s.Name = name
	s.cc = cc
	s.State = make(map[string][]byte)
	s.Invokables = make(map[string]*MockStub)
	s.Keys = list.New()

	return s
}

// MockInit Initialise this chaincode,  also starts and ends a transaction.
func (stub *MockStub) MockInit(uuid string, args [][]byte) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Init(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

//MockInvoke invokes this chaincode, also starts and ends a transaction.
func (stub *MockStub) MockInvoke(uuid string, args [][]byte) pb.Response {
	stub.args = args
	stub.MockTransactionStart(uuid)
	res := stub.cc.Invoke(stub)
	stub.MockTransactionEnd(uuid)
	return res
}

//GetArgs returns args
func (stub *MockStub) GetArgs() [][]byte {
	return stub.args
}
