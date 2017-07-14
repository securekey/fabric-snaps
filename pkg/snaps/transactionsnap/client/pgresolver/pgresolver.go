/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package pgresolver

import (
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	common "github.com/hyperledger/fabric/protos/common"
)

// SignaturePolicyFunc is a function that evaluates a signature policy and returns a peer group hierarchy
type SignaturePolicyFunc func() (GroupOfGroups, error)

// SignaturePolicyCompiler compiles a signature policy envelope and returns a peer group hierarchy
type SignaturePolicyCompiler interface {
	Compile(sigPolicyEnv *common.SignaturePolicyEnvelope) (GroupOfGroups, error)
}

// PeerRetriever is a function that retuens a set of peers for the given MSP ID
type PeerRetriever func(mspID string) []sdkApi.Peer

// PeerGroupResolver resolves a group of peers that would (exactly) satisfy
// a chaincode's endorsement policy.
type PeerGroupResolver interface {
	// Resolve returns a PeerGroup ensuring that all of the peers in the group are
	// in the given set of available peers
	// This method should never return nil but may return a PeerGroup that contains no peers.
	Resolve() PeerGroup
}

// LoadBalancePolicy is used to pick a peer group from a given set of peer groups
type LoadBalancePolicy interface {
	// Choose returns one of the peer groups from the given set of peer groups.
	// This method should never return nil but may return a PeerGroup that contains no peers.
	Choose(peerGroups []PeerGroup) PeerGroup
}
