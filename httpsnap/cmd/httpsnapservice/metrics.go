/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	"github.com/hyperledger/fabric/common/metrics"
)

var (
	httpCounter = metrics.CounterOpts{
		Namespace: "snap",
		Subsystem: "http",
		Name:      "count",
		Help:      "The number of http calls.",
	}
	httpTimer = metrics.HistogramOpts{
		Namespace: "snap",
		Subsystem: "http",
		Name:      "duration",
		Help:      "The http call duration.",
	}
)

//Metrics contain graphs
type Metrics struct {
	HTTPCounter metrics.Counter
	HTTPTimer   metrics.Histogram
}

//NewMetrics create new instance of metrics
func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		HTTPCounter: p.NewCounter(httpCounter),
		HTTPTimer:   p.NewHistogram(httpTimer),
	}
}
