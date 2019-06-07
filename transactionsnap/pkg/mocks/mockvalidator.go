/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	pb "github.com/hyperledger/fabric/protos/peer"
)

// MockValidator is a mock proposal response validator
type MockValidator struct {
	Code pb.TxValidationCode
	Err  error
}

// NewMockValidator returns a mock validator
func NewMockValidator() *MockValidator {
	return &MockValidator{}
}

// ValidateProposalResponses returns a mock response for validation
func (m *MockValidator) ValidateProposalResponses(proposal *pb.SignedProposal, proposalResponses []*pb.ProposalResponse) (pb.TxValidationCode, error) {
	return m.Code, m.Err
}
