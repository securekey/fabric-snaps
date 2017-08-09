/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"

	pb "github.com/hyperledger/fabric/protos/peer"
	protos_utils "github.com/hyperledger/fabric/protos/utils"
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
