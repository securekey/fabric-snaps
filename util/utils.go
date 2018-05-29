/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	pb "github.com/hyperledger/fabric/protos/peer"
	protos_utils "github.com/hyperledger/fabric/protos/utils"
	"github.com/securekey/fabric-snaps/util/errors"
)

// GetCreatorFromSignedProposal ...
func GetCreatorFromSignedProposal(signedProposal *pb.SignedProposal) ([]byte, error) {

	// check ProposalBytes if nil
	if signedProposal.ProposalBytes == nil {
		return nil, errors.New(errors.GeneralError, "ProposalBytes is nil in SignedProposal")
	}

	proposal, err := protos_utils.GetProposal(signedProposal.ProposalBytes)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Unmarshal ProposalBytes error")
	}
	// check proposal.Header if nil
	if proposal.Header == nil {
		return nil, errors.New(errors.GeneralError, "Header is nil in Proposal")
	}
	proposalHeader, err := protos_utils.GetHeader(proposal.Header)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Unmarshal HeaderBytes error")
	}
	// check proposalHeader.SignatureHeader if nil
	if proposalHeader.SignatureHeader == nil {
		return nil, errors.New(errors.GeneralError, "signatureHeader is nil in proposalHeader")
	}
	signatureHeader, err := protos_utils.GetSignatureHeader(proposalHeader.SignatureHeader)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Unmarshal SignatureHeader error")
	}

	return signatureHeader.Creator, nil
}
