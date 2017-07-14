/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package pgresolver

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric/protos/msp"
)

// NewPrincipal creates a new MSPPrincipal
func NewPrincipal(name string, classification mb.MSPPrincipal_Classification) (*mb.MSPPrincipal, error) {
	member1Role, err := proto.Marshal(&mb.MSPRole{Role: mb.MSPRole_MEMBER, MspIdentifier: name})
	if err != nil {
		return nil, fmt.Errorf("Error marshal MSPRole: %s", err)
	}
	return &mb.MSPPrincipal{
		PrincipalClassification: classification,
		Principal:               member1Role}, nil
}

// NewSignedByPolicy creates a SignaturePolicy at the given index
func NewSignedByPolicy(index int32) *common.SignaturePolicy {
	return &common.SignaturePolicy{
		Type: &common.SignaturePolicy_SignedBy{
			SignedBy: index,
		}}
}

// NewNOutOfPolicy creates an NOutOf signature policy
func NewNOutOfPolicy(n int32, signedBy ...*common.SignaturePolicy) *common.SignaturePolicy {
	return &common.SignaturePolicy{
		Type: &common.SignaturePolicy_NOutOf_{
			NOutOf: &common.SignaturePolicy_NOutOf{
				N:     n,
				Rules: signedBy,
			}}}
}

// GetPolicies creates a set of 'signed by' signature policies and corresponding identities for the given set of MSP IDs
func GetPolicies(mspIDs ...string) (signedBy []*common.SignaturePolicy, identities []*mb.MSPPrincipal, err error) {
	for i, mspID := range mspIDs {
		signedBy = append(signedBy, NewSignedByPolicy(int32(i)))
		principal, err := NewPrincipal(mspID, mb.MSPPrincipal_ROLE)
		if err != nil {
			return nil, nil, err
		}
		identities = append(identities, principal)
	}
	return
}
