/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	fabricmetrics "github.com/hyperledger/fabric/common/metrics"
)

var (
	refreshTimer = fabricmetrics.HistogramOpts{
		Namespace: "config_service",
		Name:      "refresh_duration",
		Help:      "The config refresh duration.",
	}
)

//Metrics contain graphs
type Metrics struct {
	RefreshTimer fabricmetrics.Histogram
}

//NewMetrics create new instance of metrics
func NewMetrics(p fabricmetrics.Provider) *Metrics {
	return &Metrics{
		RefreshTimer: p.NewHistogram(refreshTimer),
	}
}
