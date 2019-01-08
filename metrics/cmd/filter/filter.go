/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"net/http"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/handlers/auth"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/metrics/pkg/util"
	"golang.org/x/net/context"
)

var logger = logging.NewLogger("metricsfilter")

type filter struct {
	next    pb.EndorserServer
	metrics *Metrics
}

// NewFilter creates a new Filter
func NewFilter() auth.Filter { //nolint: deadcode
	return &filter{}
}

// Init initializes the Filter with the next EndorserServer
func (f *filter) Init(next pb.EndorserServer) {
	f.next = next
	f.metrics = NewMetrics(util.GetMetricsInstance())
	logger.Info("Metrics filter initialized")
}

// ProcessProposal processes a signed proposal
func (f *filter) ProcessProposal(ctx context.Context, signedProp *pb.SignedProposal) (*pb.ProposalResponse, error) {
	// increment proposal count
	f.metrics.ProposalCounter.Add(1)
	// Time proposal
	startTime := time.Now()
	defer func() { f.metrics.ProposalTimer.Observe(time.Since(startTime).Seconds()) }()

	resp, err := f.next.ProcessProposal(ctx, signedProp)

	// increment proposal error count, if required.
	if err != nil || resp.GetResponse().GetStatus() != http.StatusOK {
		f.metrics.ProposalErrorCounter.Add(1)
	}

	return resp, err
}

func main() {}
