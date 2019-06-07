/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator

import (
	"sync"

	"github.com/hyperledger/fabric/core/committer/txvalidator"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/msp/mgmt"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// Validator validates endorsement proposals
type Validator interface {
	ValidateProposalResponses(proposal *pb.SignedProposal, proposalResponses []*pb.ProposalResponse) (pb.TxValidationCode, error)
}

// Manager manages validators
type Manager interface {
	ValidatorForChannel(channelID string) Validator
}

// Get returns the Validator Manager
func Get() Manager {
	return instance
}

type validatorMgr struct {
	validators map[string]*validator
	mutex      sync.RWMutex
}

var instance = newMgr()

func newMgr() *validatorMgr {
	return &validatorMgr{
		validators: make(map[string]*validator),
	}
}

// ValidatorForChannel returns the validator for the given channel
func (m *validatorMgr) ValidatorForChannel(channelID string) Validator {
	m.mutex.RLock()
	v, ok := m.validators[channelID]
	m.mutex.RUnlock()

	if ok {
		return v
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	v, ok = m.validators[channelID]
	if !ok {
		mspMgr := mgmt.GetManagerForChain(channelID)
		pe := &txvalidator.PolicyEvaluator{
			IdentityDeserializer: mspMgr,
		}
		v = newValidator(channelID, peer.GetLedger(channelID), pe, mspMgr)
		m.validators[channelID] = v
	}

	return v
}
