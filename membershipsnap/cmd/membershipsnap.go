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
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/core/policy"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	pb "github.com/hyperledger/fabric/protos/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/util"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("membershipsnap")

// Available function:
const (
	getAllPeersFunction       = "getAllPeers"
	getPeersOfChannelFunction = "getPeersOfChannel"
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
		errObj := errors.Wrap(errors.SystemError, err, "error getting membership service")
		logger.Errorf(errObj.GenerateLogMsg())
		return errObj
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
		if !ledgerconfig.HasRole(ledgerconfig.EndorserRole) {
			logger.Infof("Not starting Membership Snap on channel [%s] since this peer is not an endorser", stub.GetChannelID())
			return shim.Success(nil)
		}

		logger.Info("Initializing membership snap...")
		err := initializer(t)
		if err != nil {
			return util.CreateShimResponseFromError(errors.WithMessage(errors.InitializeSnapError, err, "Error initializing Membership Snap"), logger, stub)
		}
		logger.Info("... successfully initialized membership snap")
	} else {
		logger.Debugf("Initializing membership snap - nothing to do for channel [%s]", stub.GetChannelID())
	}
	return shim.Success(nil)
}

// Invoke is the main entry point for invocations
func (t *MembershipSnap) Invoke(stub shim.ChaincodeStubInterface) (resp pb.Response) {

	defer util.HandlePanic(&resp, logger, stub)

	args := stub.GetArgs()
	if len(args) == 0 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, fmt.Sprintf("Function not provided. Expecting one of: %s or %s", getAllPeersFunction, getPeersOfChannelFunction)), logger, stub)
	}

	functionName := string(args[0])

	// Check ACL
	sp, err := stub.GetSignedProposal()
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.SystemError, err, "Failed getting signed proposal from stub"), logger, stub)
	}
	if err = t.policyChecker.CheckPolicyNoChannel(mspmgmt.Members, sp); err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.ACLCheckError, err, fmt.Sprintf("\"%s\" request failed authorization check", functionName)), logger, stub)
	}

	switch functionName {
	case getAllPeersFunction:
		return t.getAllPeers(stub, args[1:])
	case getPeersOfChannelFunction:
		return t.getPeersOfChannel(stub, args[1:])
	default:
		return util.CreateShimResponseFromError(errors.New(errors.InvalidFunctionError, fmt.Sprintf("Invalid function: %s. Expecting one of: %s or %s", functionName, getAllPeersFunction, getPeersOfChannelFunction)), logger, stub)
	}

}

//getAllPeers retrieves all of the peers that are currently alive
func (t *MembershipSnap) getAllPeers(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	payload, err := t.marshalEndpoints(t.membershipService.GetAllPeers())
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.MembershipError, err, "Failed to marshal endpoints"), logger, stub)
	}
	return shim.Success(payload)
}

//getPeersOfChannel retrieves all of the peers that are currently alive and joined to the given channel
func (t *MembershipSnap) getPeersOfChannel(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) == 0 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Expecting channel ID"), logger, stub)
	}

	channelID := string(args[0])
	if channelID == "" {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Expecting channel ID"), logger, stub)
	}

	endpoints, err := t.membershipService.GetPeersOfChannel(channelID)
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.MembershipError, err, "Failed to get peers of channel"), logger, stub)
	}

	payload, err := t.marshalEndpoints(endpoints)
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.SystemError, err, "Marshal endpoints failed"), logger, stub)
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
