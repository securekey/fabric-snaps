/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/golang/protobuf/proto"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
)

// ProposalBuilder builds a mock signed proposal
type ProposalBuilder struct {
	channelID    string
	chaincodeID  string
	mspID        string
	args         [][]byte
	transientMap map[string][]byte
}

// NewProposalBuilder returns a mock proposal builder
func NewProposalBuilder() *ProposalBuilder {
	return &ProposalBuilder{
		transientMap: make(map[string][]byte),
	}
}

// ChannelID sets the channel ID for the proposal
func (b *ProposalBuilder) ChannelID(value string) *ProposalBuilder {
	b.channelID = value
	return b
}

// ChaincodeID sets the chaincode ID for the proposal
func (b *ProposalBuilder) ChaincodeID(value string) *ProposalBuilder {
	b.chaincodeID = value
	return b
}

// MSPID sets the MSP ID of the creator
func (b *ProposalBuilder) MSPID(value string) *ProposalBuilder {
	b.mspID = value
	return b
}

// Args adds chaincode arguments
func (b *ProposalBuilder) Args(args ...[]byte) *ProposalBuilder {
	b.args = args
	return b
}

// TransientArg adds a transient key-value
func (b *ProposalBuilder) TransientArg(key string, value []byte) *ProposalBuilder {
	b.transientMap[key] = value
	return b
}

// Build returns the signed proposal
func (b *ProposalBuilder) Build() *pb.SignedProposal {
	// create invocation spec to target a chaincode with arguments
	ccis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{
		Type: pb.ChaincodeSpec_GOLANG, ChaincodeId: &pb.ChaincodeID{Name: b.chaincodeID},
		Input: &pb.ChaincodeInput{Args: b.args}}}

	sID := &msp.SerializedIdentity{Mspid: b.mspID, IdBytes: []byte(CertPem)}
	creator, err := proto.Marshal(sID)
	if err != nil {
		panic(err)
	}

	proposal, _, err := putils.CreateChaincodeProposalWithTransient(
		cb.HeaderType_ENDORSER_TRANSACTION, b.channelID, ccis, creator, b.transientMap)
	if err != nil {
		panic(fmt.Sprintf("Could not create chaincode proposal, err %s", err))
	}

	proposalBytes, err := proto.Marshal(proposal)
	if err != nil {
		panic(fmt.Sprintf("Error marshalling proposal: %s", err))
	}

	// sign proposal bytes
	block, _ := pem.Decode(KeyPem)
	lowLevelKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		panic(err)
	}

	signature, err := SignECDSA(lowLevelKey, proposalBytes)
	if err != nil {
		panic(err)
	}

	return &pb.SignedProposal{ProposalBytes: proposalBytes, Signature: signature}
}
