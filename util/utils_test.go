/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"strings"
	"testing"

	"github.com/hyperledger/fabric/protos/common"
	protos_peer "github.com/hyperledger/fabric/protos/peer"
	protos_utils "github.com/hyperledger/fabric/protos/utils"
)

func TestGetCreatorFromSignedProposal(t *testing.T) {
	// Test ProposalBytes is nil in SignedProposal
	_, err := GetCreatorFromSignedProposal(&protos_peer.SignedProposal{})
	if err == nil {
		t.Fatalf("GetCreatorFromSignedProposal should return error when ProposalBytes is nil")
	}
	if !strings.Contains(err.Error(), "ProposalBytes is nil in SignedProposal") {
		t.Fatalf("GetCreatorFromSignedProposal didn't return the appropriate error message (%s)", err.Error())
	}

	// Test proposal.Header is nil
	proposal := createTransactionProposal(t, "test")
	proposal.Header = nil
	proposalBytes, err := protos_utils.GetBytesProposal(proposal)
	if err != nil {
		t.Fatalf("GetBytesProposal return error %v", err)
	}
	_, err = GetCreatorFromSignedProposal(&protos_peer.SignedProposal{ProposalBytes: proposalBytes})
	if err == nil {
		t.Fatalf("GetCreatorFromSignedProposal should return error when proposal.Header is nil")
	}
	if !strings.Contains(err.Error(), "Header is nil in Proposal") {
		t.Fatalf("GetCreatorFromSignedProposal didn't return the appropriate error message (%s)", err.Error())
	}

	// Test header.SignatureHeader is nil
	proposal = createTransactionProposal(t, "test")
	header, err := protos_utils.GetHeader(proposal.Header)
	if err != nil {
		t.Fatalf("GetHeader return error %v", err)
	}
	header.SignatureHeader = nil
	proposal.Header, err = protos_utils.GetBytesHeader(header)
	if err != nil {
		t.Fatalf("GetBytesHeader return error %v", err)
	}
	proposalBytes, err = protos_utils.GetBytesProposal(proposal)
	if err != nil {
		t.Fatalf("GetBytesProposal return error %v", err)
	}
	_, err = GetCreatorFromSignedProposal(&protos_peer.SignedProposal{ProposalBytes: proposalBytes})
	if err == nil {
		t.Fatalf("GetCreatorFromSignedProposal should return error when proposalHeader.SignatureHeader is nil")
	}
	if !strings.Contains(err.Error(), "signatureHeader is nil in proposalHeader") {
		t.Fatalf("GetCreatorFromSignedProposal didn't return the appropriate error message (%s)", err.Error())
	}

	// Test valid creator
	proposal = createTransactionProposal(t, "test")
	proposalBytes, err = protos_utils.GetBytesProposal(proposal)
	if err != nil {
		t.Fatalf("GetBytesProposal return error %v", err)
	}
	creator, err := GetCreatorFromSignedProposal(&protos_peer.SignedProposal{ProposalBytes: proposalBytes})
	if err != nil {
		t.Fatalf("GetCreatorFromSignedProposal return error %v", err)
	}
	if string(creator) != "creatorValue" {
		t.Fatalf("GetCreatorFromSignedProposal return unexpected creator %s", string(creator))
	}
}
func createTransactionProposal(t *testing.T, chainID string) *protos_peer.Proposal {
	var args [][]byte
	args = append(args, []byte("invoke"))
	ccis := &protos_peer.ChaincodeInvocationSpec{ChaincodeSpec: &protos_peer.ChaincodeSpec{
		Type: protos_peer.ChaincodeSpec_GOLANG, ChaincodeId: &protos_peer.ChaincodeID{Name: "ccID"},
		Input: &protos_peer.ChaincodeInput{Args: args}}}
	transientDataMap := make(map[string][]byte)
	transientDataMap["test"] = []byte("transientData")
	proposal, _, err := protos_utils.CreateChaincodeProposalWithTransient(
		common.HeaderType_ENDORSER_TRANSACTION, chainID, ccis, []byte("creatorValue"), transientDataMap)
	if err != nil {
		t.Fatalf("Could not create chaincode proposal, err %s", err)
	}
	return proposal
}
