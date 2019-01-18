/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	fabricmetrics "github.com/hyperledger/fabric/common/metrics"
)

var (
	transactionRetryCounter = fabricmetrics.CounterOpts{
		Namespace: "snap",
		Subsystem: "txn",
		Name:      "retry",
		Help:      "The number of transaction retry.",
	}
)

//Metrics contain graphs
type Metrics struct {
	TransactionRetryCounter fabricmetrics.Counter
}

//NewMetrics create new instance of metrics
func NewMetrics(p fabricmetrics.Provider) *Metrics {
	return &Metrics{
		TransactionRetryCounter: p.NewCounter(transactionRetryCounter),
	}
}
