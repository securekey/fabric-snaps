/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"

	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	protos_utils "github.com/securekey/fabric-snaps/internal/github.com/hyperledger/fabric/protos/utils"
)

// GetCreatorFromSignedProposal ...
func GetCreatorFromSignedProposal(signedProposal *pb.SignedProposal) ([]byte, error) {

	// check ProposalBytes if nil
	if signedProposal.ProposalBytes == nil {
		return nil, fmt.Errorf("ProposalBytes is nil in SignedProposal")
	}

	proposal, err := protos_utils.GetProposal(signedProposal.ProposalBytes)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal ProposalBytes error %v", err)
	}
	// check proposal.Header if nil
	if proposal.Header == nil {
		return nil, fmt.Errorf("Header is nil in Proposal")
	}
	proposalHeader, err := protos_utils.GetHeader(proposal.Header)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal HeaderBytes error %v", err)
	}
	// check proposalHeader.SignatureHeader if nil
	if proposalHeader.SignatureHeader == nil {
		return nil, fmt.Errorf("signatureHeader is nil in proposalHeader")
	}
	signatureHeader, err := protos_utils.GetSignatureHeader(proposalHeader.SignatureHeader)
	if err != nil {
		return nil, fmt.Errorf("Unmarshal SignatureHeader error %v", err)
	}

	return signatureHeader.Creator, nil
}

//GetByteArgs utility which converts string args array to byte args array
func GetByteArgs(argsArray []string) [][]byte {
	txArgs := make([][]byte, len(argsArray))
	for i, val := range argsArray {
		txArgs[i] = []byte(val)
	}
	return txArgs
}
