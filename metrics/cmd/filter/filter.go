/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"net/http"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/handlers/auth"
	"github.com/hyperledger/fabric/protos/peer"
	pb "github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
	"github.com/securekey/fabric-snaps/metrics/cmd/filter/metrics"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/uber-go/tally"
	"golang.org/x/net/context"
)

var logger = logging.NewLogger("metricsfilter")

type filter struct {
	next                 peer.EndorserServer
	proposalCounter      tally.Counter
	proposalErrorCounter tally.Counter
	proposalTimer        tally.Timer
}

// NewFilter creates a new Filter
func NewFilter() auth.Filter { //nolint: deadcode
	return &filter{}
}

// Init initializes the Filter with the next EndorserServer
func (f *filter) Init(next peer.EndorserServer) {
	f.next = next

	f.proposalCounter = metrics.RootScope.Counter("proposal_count")
	f.proposalErrorCounter = metrics.RootScope.Counter("proposal_error_count")
	if metrics.IsDebug() {
		f.proposalTimer = metrics.RootScope.Timer("proposal_processing_time_seconds")
	}
	logger.Info("Metrics filter initialized")
}

// ProcessProposal processes a signed proposal
func (f *filter) ProcessProposal(ctx context.Context, signedProp *peer.SignedProposal) (*peer.ProposalResponse, error) {
	// increment proposal count
	f.proposalCounter.Inc(1)

	// Time proposal
	if metrics.IsDebug() {
		stopwatch := f.proposalTimer.Start()
		defer stopwatch.Stop()
	}
	resp, err := f.next.ProcessProposal(ctx, signedProp)

	// increment proposal error count, if required.
	if err != nil || resp.GetResponse().GetStatus() != http.StatusOK {
		f.proposalErrorCounter.Inc(1)
	}

	return resp, err
}

func (f *filter) getCCProposalPayloadAndCis(signedProp *pb.SignedProposal) (*pb.ChaincodeProposalPayload, *pb.ChaincodeInvocationSpec, errors.Error) {
	prop, err := putils.GetProposal(signedProp.ProposalBytes)
	if err != nil {
		return nil, nil, errors.WithMessage(errors.SystemError, err, "Failed to extract proposal from proposal bytes")
	}

	cis, err := putils.GetChaincodeInvocationSpec(prop)
	if err != nil {
		return nil, nil, errors.WithMessage(errors.SystemError, err, "Failed to get chaincode invocation spec")
	}

	ccProp, err := putils.GetChaincodeProposalPayload(prop.Payload)
	if err != nil {
		return nil, nil, errors.WithMessage(errors.SystemError, err, "Failed to get chaincode proposal payload")
	}

	return ccProp, cis, nil
}

func main() {}
