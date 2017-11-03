/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/service"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	pb "github.com/hyperledger/fabric/protos/peer"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/membershipsnap/cmd/api"
)

var logger = logging.MustGetLogger("membershipsnap")

// Available function:
const (
	getAllPeersFunction       = "getAllPeers"
	getPeersOfChannelFunction = "getPeersOfChannel"
	registerGossipFunction    = "registerGossip"
)

// mspMap manages a map of PKI IDs to MSP IDs
type mspIDProvider interface {
	GetMSPID(pkiID common.PKIidType) string
}

// MembershipSnap is the System Chaincode that provides information about peer membership
type MembershipSnap struct {
	policyChecker    policy.PolicyChecker
	gossipService    service.GossipService
	mspprovider      mspIDProvider
	localMSPID       []byte
	localPeerAddress string
}

// NewMSCC returns a new Membership System Chaincode
func NewMSCC() *MembershipSnap {
	return &MembershipSnap{}
}

var initOnce sync.Once
var mspProvider mspIDProvider

type ccInitializer func(*MembershipSnap, shim.ChaincodeStubInterface) error

var initializer ccInitializer = func(mscc *MembershipSnap, stub shim.ChaincodeStubInterface) error {
	initOnce.Do(func() {
		mspProvider = newMSPIDMgr(service.GetGossipService())
	})

	localMSPID, err := mspmgmt.GetLocalMSP().GetIdentifier()
	if err != nil {
		return fmt.Errorf("Error getting local MSP Identifier: %s", err)
	}

	peerEndpoint, err := peer.GetPeerEndpoint()
	if err != nil {
		return fmt.Errorf("Error reading peer endpoint: %s", err)
	}

	// Init policy checker for access control
	policyChecker := policy.NewPolicyChecker(
		peer.NewChannelPolicyManagerGetter(),
		mspmgmt.GetLocalMSP(),
		mspmgmt.NewLocalMSPPrincipalGetter(),
	)

	mscc.localMSPID = []byte(localMSPID)
	mscc.localPeerAddress = peerEndpoint.Address
	mscc.gossipService = service.GetGossipService()
	mscc.mspprovider = mspProvider
	mscc.policyChecker = policyChecker

	logger.Infof("Successfully initialized")

	return nil
}

// Init is called once when the chaincode started the first time
func (t *MembershipSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	err := initializer(t, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error initializing MSCC: %s", err))
	}
	return shim.Success(nil)
}

// Invoke is the main entry point for invocations
func (t *MembershipSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) == 0 {
		return shim.Error(fmt.Sprintf("Function not provided. Expecting one of %s or %s", getAllPeersFunction, getPeersOfChannelFunction))
	}

	functionName := string(args[0])

	// Check ACL
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed getting signed proposal from stub: [%s]", err))
	}
	if err = t.policyChecker.CheckPolicyNoChannel(mspmgmt.Members, sp); err != nil {
		return shim.Error(fmt.Sprintf("\"%s\" request failed authorization check: [%s]", functionName, err))
	}

	switch functionName {
	case getAllPeersFunction:
		return t.getAllPeers(stub, args[1:])
	case getPeersOfChannelFunction:
		return t.getPeersOfChannel(stub, args[1:])
	default:
		return shim.Error(fmt.Sprintf("Invalid function: %s. Expecting one of %s or %s", functionName, getAllPeersFunction, getPeersOfChannelFunction))
	}
}

//getAllPeers retrieves all of the peers (excluding this one) that are currently alive
func (t *MembershipSnap) getAllPeers(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	payload, err := t.marshalEndpoints(t.gossipService.Peers(), true)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(payload)
}

//getPeersOfChannel retrieves all of the peers (excluding this one) that are currently alive and joined to the given channel
func (t *MembershipSnap) getPeersOfChannel(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("Expecting channel ID")
	}

	channelID := string(args[0])
	if channelID == "" {
		return shim.Error("Expecting channel ID")
	}

	localPeerJoined := false
	for _, ch := range peer.GetChannelsInfo() {
		if ch.ChannelId == channelID {
			localPeerJoined = true
			break
		}
	}

	payload, err := t.marshalEndpoints(t.gossipService.PeersOfChannel(common.ChainID(channelID)), localPeerJoined)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(payload)
}

func (t *MembershipSnap) marshalEndpoints(members []discovery.NetworkMember, includeLocalPeer bool) ([]byte, error) {
	peerEndpoints := &api.PeerEndpoints{}
	for _, member := range members {
		peerEndpoints.Endpoints = append(peerEndpoints.Endpoints, &api.PeerEndpoint{
			Endpoint:         member.Endpoint,
			InternalEndpoint: member.InternalEndpoint,
			MSPid:            []byte(t.mspprovider.GetMSPID(member.PKIid)),
		})
	}

	if includeLocalPeer {
		// Add self since Gossip only contains other peers
		self := &api.PeerEndpoint{
			Endpoint:         t.localPeerAddress,
			InternalEndpoint: t.localPeerAddress,
			MSPid:            t.localMSPID,
		}

		peerEndpoints.Endpoints = append(peerEndpoints.Endpoints, self)
	}

	payload, err := proto.Marshal(peerEndpoints)
	if err != nil {
		return nil, fmt.Errorf("error marshalling peer endpoints: %v", err)
	}
	return payload, nil
}

func main() {
}
