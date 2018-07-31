/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/policy"
	policymocks "github.com/hyperledger/fabric/core/policy/mocks"
	"github.com/hyperledger/fabric/gossip/api"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"

	"github.com/hyperledger/fabric/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

// newMockStub creates a mock stub for the MSCC.
// - identity is the identity of the signer
// - identityDeserializer is the deserializer that validates and deserializes the identity
// - localMSPID is the ID of the peer's local MSP
// - localPeerAddress is the address (host:port) of the local peer
// - members contains zero or more MSP network members
func newMockStub(identity []byte, identityDeserializer msp.IdentityDeserializer, localMSPID api.OrgIdentityType, localPeerAddress string, bcInfo []*mockbcinfo.ChannelBCInfo, members ...memservice.MspNetworkMembers) *shim.MockStub { //nolint: deadcode
	// Override the MSCC initializer in order to inject our mocks
	initializer = func(mscc *MembershipSnap) error {
		policyChecker := policy.NewPolicyChecker(
			&policymocks.MockChannelPolicyManagerGetter{
				Managers: map[string]policies.Manager{},
			},
			identityDeserializer,
			&policymocks.MockMSPPrincipalGetter{Principal: identity},
		)

		m := make(map[string]string)
		for _, member := range members {
			for _, netMember := range member.NetworkMembers {
				m[string(netMember.PKIid)] = string(member.MspID)
			}
		}

		mscc.policyChecker = policyChecker
		mscc.membershipService = memservice.NewServiceWithMocks(localMSPID, localPeerAddress, bcInfo, members...)

		return nil
	}

	stub := shim.NewMockStub("MembershipSnap", New())
	stub.MockInit("txid", nil)

	return stub
}

func newMockIdentity() []byte { //nolint: deadcode
	return []byte("Some Identity")
}

func newMockSignedProposal(identity []byte) (*pb.SignedProposal, msp.IdentityDeserializer) { //nolint: deadcode
	sProp, _ := utils.MockSignedEndorserProposalOrPanic("", &pb.ChaincodeSpec{}, identity, nil)
	sProp.Signature = sProp.ProposalBytes
	identityDeserializer := &policymocks.MockIdentityDeserializer{
		Identity: identity,
		Msg:      sProp.ProposalBytes,
	}
	return sProp, identityDeserializer
}
