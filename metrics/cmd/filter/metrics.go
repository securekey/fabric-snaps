/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"github.com/hyperledger/fabric/common/metrics"
)

var (
	proposalCounter = metrics.CounterOpts{
		Namespace: "proposal",
		Name:      "count",
		Help:      "The number of proposal.",
	}
	proposalErrorCounter = metrics.CounterOpts{
		Namespace: "proposal",
		Name:      "error_count",
		Help:      "The number of failed proposal.",
	}
	proposalTimer = metrics.HistogramOpts{
		Namespace: "proposal",
		Name:      "duration",
		Help:      "The proposal duration.",
	}
)

//Metrics contain graphs
type Metrics struct {
	ProposalCounter      metrics.Counter
	ProposalErrorCounter metrics.Counter
	ProposalTimer        metrics.Histogram
}

//NewMetrics create new instance of metrics
func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		ProposalCounter:      p.NewCounter(proposalCounter),
		ProposalErrorCounter: p.NewCounter(proposalErrorCounter),
		ProposalTimer:        p.NewHistogram(proposalTimer),
	}
}
