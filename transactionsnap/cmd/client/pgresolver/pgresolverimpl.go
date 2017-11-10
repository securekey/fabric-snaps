/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pgresolver

import (
	"fmt"
	"reflect"

	"github.com/golang/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	common "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	mb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("pg-resolver")

type peerGroupResolver struct {
	mspGroups []api.Group
	lbp       api.LoadBalancePolicy
}

// NewRoundRobinPeerGroupResolver returns a PeerGroupResolver that chooses peers in a round-robin fashion
func NewRoundRobinPeerGroupResolver(sigPolicyEnv *common.SignaturePolicyEnvelope, peerRetriever api.PeerRetriever) (api.PeerGroupResolver, error) {
	compiler := NewSignaturePolicyCompiler(peerRetriever)
	groupHierarchy, err := compiler.Compile(sigPolicyEnv)
	if err != nil {
		return nil, fmt.Errorf("error evaluating signature policy: %s", err)
	}
	return NewPeerGroupResolver(groupHierarchy, NewRoundRobinLBP())
}

// NewRandomPeerGroupResolver returns a PeerGroupResolver that chooses peers in a round-robin fashion
func NewRandomPeerGroupResolver(sigPolicyEnv *common.SignaturePolicyEnvelope, peerRetriever api.PeerRetriever) (api.PeerGroupResolver, error) {
	compiler := NewSignaturePolicyCompiler(peerRetriever)
	groupHierarchy, err := compiler.Compile(sigPolicyEnv)
	if err != nil {
		return nil, fmt.Errorf("error evaluating signature policy: %s", err)
	}
	return NewPeerGroupResolver(groupHierarchy, NewRandomLBP())
}

// NewPeerGroupResolver returns a new PeerGroupResolver
func NewPeerGroupResolver(groupHierarchy api.GroupOfGroups, lbp api.LoadBalancePolicy) (api.PeerGroupResolver, error) {
	if logging.IsEnabledForLogger(logging.DEBUG, logger) {
		logger.Debugf("\n***** Policy: %s\n", groupHierarchy)
	}

	mspGroups := groupHierarchy.Reduce()

	if logging.IsEnabledForLogger(logging.DEBUG, logger) {
		s := "\n***** Org Groups:\n"
		for i, g := range mspGroups {
			s += fmt.Sprintf("%s", g)
			if i+1 < len(mspGroups) {
				s += fmt.Sprintf("  OR\n")
			}
		}
		s += fmt.Sprintf("\n")
		logger.Debugf(s)
	}

	return &peerGroupResolver{
		mspGroups: mspGroups,
		lbp:       lbp,
	}, nil
}

func (c *peerGroupResolver) Resolve() api.PeerGroup {
	peerGroups := c.getPeerGroups()

	if logging.IsEnabledForLogger(logging.DEBUG, logger) {
		s := ""
		if len(peerGroups) == 0 {
			s = "\n\n***** No Available Peer Groups\n"
		} else {
			s = "\n\n***** Available Peer Groups:\n"
			for i, grp := range peerGroups {
				s += fmt.Sprintf("%d - %s", i, grp)
				if i+1 < len(peerGroups) {
					s += fmt.Sprintf(" OR\n")
				}
			}
			s += fmt.Sprintf("\n")
		}
		logger.Debugf(s)
	}

	return c.lbp.Choose(peerGroups)
}

func (c *peerGroupResolver) getPeerGroups() []api.PeerGroup {
	var allPeerGroups []api.PeerGroup
	for _, g := range c.mspGroups {
		for _, pg := range mustGetPeerGroups(g) {
			allPeerGroups = append(allPeerGroups, pg)
		}
	}
	return allPeerGroups
}

func mustGetPeerGroups(group api.Group) []api.PeerGroup {
	items := group.Items()
	if len(items) == 0 {
		return nil
	}

	if _, ok := items[0].(api.Group); !ok {
		group = NewGroup([]api.Item{group})
	}

	groups := make([]api.Group, len(group.Items()))
	for i, item := range group.Items() {
		if grp, ok := item.(api.PeerGroup); ok {
			groups[i] = grp
		} else {
			panic(fmt.Sprintf("unexpected: expecting item to be a PeerGroup but found: %s", reflect.TypeOf(item)))
		}
	}

	andedGroups := and(groups)
	peerGroups := make([]api.PeerGroup, len(andedGroups))
	for i, g := range andedGroups {
		peerGroups[i] = mustGetPeerGroup(g)
	}

	return peerGroups
}

func mustGetPeerGroup(g api.Group) api.PeerGroup {
	if pg, ok := g.(api.PeerGroup); ok {
		return pg
	}

	var peers []sdkApi.Peer
	for _, item := range g.Items() {
		if pg, ok := item.(sdkApi.Peer); ok {
			peers = append(peers, pg)
		} else {
			panic(fmt.Sprintf("expecting item to be a Peer but found: %s", reflect.TypeOf(item)))
		}
	}
	return NewPeerGroup(peers...)
}

// NewSignaturePolicyCompiler returns a new PolicyCompiler
func NewSignaturePolicyCompiler(peerRetriever api.PeerRetriever) api.SignaturePolicyCompiler {
	return &signaturePolicyCompiler{
		peerRetriever: peerRetriever,
	}
}

type signaturePolicyCompiler struct {
	peerRetriever api.PeerRetriever
}

func (c *signaturePolicyCompiler) Compile(sigPolicyEnv *common.SignaturePolicyEnvelope) (api.GroupOfGroups, error) {
	policFunc, err := c.compile(sigPolicyEnv.Rule, sigPolicyEnv.Identities)
	if err != nil {
		return nil, fmt.Errorf("error compiling chaincode signature policy: %s", err)
	}
	return policFunc()
}

func (c *signaturePolicyCompiler) compile(sigPolicy *common.SignaturePolicy, identities []*mb.MSPPrincipal) (api.SignaturePolicyFunc, error) {
	if sigPolicy == nil {
		return nil, fmt.Errorf("nil signature policy")
	}

	switch t := sigPolicy.Type.(type) {
	case *common.SignaturePolicy_SignedBy:
		return func() (api.GroupOfGroups, error) {
			mspID, err := mspPrincipalToString(identities[t.SignedBy])
			if err != nil {
				return nil, fmt.Errorf("error getting MSP ID from MSP principal: %s", err)
			}
			return NewGroupOfGroups([]api.Group{NewMSPPeerGroup(mspID, c.peerRetriever)}), nil
		}, nil

	case *common.SignaturePolicy_NOutOf_:
		nOutOfPolicy := t.NOutOf
		var pfuncs []api.SignaturePolicyFunc
		for _, policy := range nOutOfPolicy.Rules {
			f, err := c.compile(policy, identities)
			if err != nil {
				return nil, err
			}
			pfuncs = append(pfuncs, f)
		}
		return func() (api.GroupOfGroups, error) {
			var groups []api.Group
			for _, f := range pfuncs {
				grps, err := f()
				if err != nil {
					return nil, err
				}
				groups = append(groups, grps)
			}

			itemGroups, err := NewGroupOfGroups(groups).Nof(nOutOfPolicy.N)
			if err != nil {
				return nil, err
			}

			return itemGroups, nil
		}, nil

	default:
		return nil, fmt.Errorf("unsupported signature policy type: %v", t)
	}
}

func mspPrincipalToString(principal *mb.MSPPrincipal) (string, error) {
	switch principal.PrincipalClassification {
	case mb.MSPPrincipal_ROLE:
		// Principal contains the msp role
		mspRole := &mb.MSPRole{}
		proto.Unmarshal(principal.Principal, mspRole)
		return mspRole.MspIdentifier, nil

	case mb.MSPPrincipal_ORGANIZATION_UNIT:
		// Principal contains the OrganizationUnit
		unit := &mb.OrganizationUnit{}
		proto.Unmarshal(principal.Principal, unit)
		return unit.MspIdentifier, nil

	case mb.MSPPrincipal_IDENTITY:
		// TODO: Do we need to support this?
		return "", fmt.Errorf("unsupported PrincipalClassification type: %s", reflect.TypeOf(principal.PrincipalClassification))

	default:
		return "", fmt.Errorf("unknown PrincipalClassification type: %s", reflect.TypeOf(principal.PrincipalClassification))
	}
}
