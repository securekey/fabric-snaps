/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	acl "github.com/hyperledger/fabric/core/aclmgmt"
)

var logger = logging.NewLogger("acltestsnap")

type AclTestSnap struct {
}

func (aclTestSnap *AclTestSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (aclTestSnap *AclTestSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("ACL TEST INVOKED")

	signedProposal, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error("Failed to get signed proposal")
	}

	// Request should fail the org1 admin policy
	err = acl.GetACLProvider().CheckACL("sktestadmin", "mychannel", signedProposal)
	if err == nil {
		return shim.Error("Succesful ACL CHECK but should have failed for Org1 Admin policy")
	}

	// Request should pass the org1 member policy
	err = acl.GetACLProvider().CheckACL("sktestmember", "mychannel", signedProposal)
	if err != nil {
		return shim.Error("Failed ACL CHECK, org1 member policy: " + err.Error())
	}

	bytes := []byte("done")
	return shim.Success(bytes)
}

// New chaincode implementation
func New() shim.Chaincode {
	return &AclTestSnap{}
}

func main() {
}
