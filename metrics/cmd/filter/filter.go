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
	"github.com/securekey/fabric-snaps/metrics/cmd/filter/metrics"
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
func NewFilter() auth.Filter {
	return &filter{}
}

// Init initializes the Filter with the next EndorserServer
func (f *filter) Init(next peer.EndorserServer) {
	f.next = next

	f.proposalCounter = metrics.RootScope.Counter("proposal_count")
	f.proposalErrorCounter = metrics.RootScope.Counter("proposal_error_count")
	f.proposalTimer = metrics.RootScope.Timer("proposal_processing_time_seconds")

	logger.Info("Metrics filter initialized")
}

// ProcessProposal processes a signed proposal
func (f *filter) ProcessProposal(ctx context.Context, signedProp *peer.SignedProposal) (*peer.ProposalResponse, error) {
	// increment proposal count
	f.proposalCounter.Inc(1)

	// Time proposal
	stopwatch := f.proposalTimer.Start()
	resp, err := f.next.ProcessProposal(ctx, signedProp)
	stopwatch.Stop()

	// increment proposal error count, if required.
	if err != nil || resp.GetResponse().GetStatus() != http.StatusOK {
		f.proposalErrorCounter.Inc(1)
	}

	return resp, err
}

func main() {}
