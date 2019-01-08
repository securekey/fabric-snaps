/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"github.com/hyperledger/fabric/common/metrics"
)

var (
	transactionRetryCounter = metrics.CounterOpts{
		Namespace: "transaction",
		Name:      "retry",
		Help:      "The number of transaction retry.",
	}
)

//Metrics contain graphs
type Metrics struct {
	TransactionRetryCounter metrics.Counter
}

//NewMetrics create new instance of metrics
func NewMetrics(p metrics.Provider) *Metrics {
	return &Metrics{
		TransactionRetryCounter: p.NewCounter(transactionRetryCounter),
	}
}
