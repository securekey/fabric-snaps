/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	pb "github.com/hyperledger/fabric/protos/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("membershipsnap")

// Available function:
const (
	getAllPeersFunction       = "getAllPeers"
	getPeersOfChannelFunction = "getPeersOfChannel"
	registerGossipFunction    = "registerGossip"
)

// MembershipSnap is the System Chaincode that provides information about peer membership
type MembershipSnap struct {
	policyChecker     policy.PolicyChecker
	membershipService memserviceapi.Service
}

// New returns a new Membership Snap
func New() shim.Chaincode {
	return &MembershipSnap{}
}

type ccInitializer func(*MembershipSnap) error

var initializer ccInitializer = func(mscc *MembershipSnap) error {
	service, err := memservice.Get()
	if err != nil {
		logger.Errorf("Error getting membership service: %s\n", err)
		return errors.Wrap(errors.GeneralError, err, "error getting membership service")
	}

	// Init policy checker for access control
	policyChecker := policy.NewPolicyChecker(
		peer.NewChannelPolicyManagerGetter(),
		mspmgmt.GetLocalMSP(),
		mspmgmt.NewLocalMSPPrincipalGetter(),
	)

	mscc.policyChecker = policyChecker
	mscc.membershipService = service

	logger.Info("Successfully initialized membership snap")

	return nil
}

// Init is called once when the chaincode started the first time
func (t *MembershipSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	if stub.GetChannelID() == "" {
		logger.Info("Initializing membership snap...\n")
		err := initializer(t)
		if err != nil {
			return shim.Error(fmt.Sprintf("Error initializing Membership Snap: %s", err))
		}
		logger.Info("... successfully initialized membership snap\n")
	} else {
		logger.Infof("Initializing membership snap - nothing to do for channel [%s]\n", stub.GetChannelID())
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

//getAllPeers retrieves all of the peers that are currently alive
func (t *MembershipSnap) getAllPeers(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	payload, err := t.marshalEndpoints(t.membershipService.GetAllPeers())
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(payload)
}

//getPeersOfChannel retrieves all of the peers that are currently alive and joined to the given channel
func (t *MembershipSnap) getPeersOfChannel(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return shim.Error("Expecting channel ID")
	}

	channelID := string(args[0])
	if channelID == "" {
		return shim.Error("Expecting channel ID")
	}

	endpoints, err := t.membershipService.GetPeersOfChannel(channelID)
	if err != nil {
		return shim.Error(err.Error())
	}

	payload, err := t.marshalEndpoints(endpoints)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(payload)
}

func (t *MembershipSnap) marshalEndpoints(endpoints []*memserviceapi.PeerEndpoint) ([]byte, error) {
	payload, err := proto.Marshal(&memserviceapi.PeerEndpoints{Endpoints: endpoints})
	if err != nil {
		return nil, fmt.Errorf("error marshalling peer endpoints: %s", err)
	}
	return payload, nil
}

func main() {
}
