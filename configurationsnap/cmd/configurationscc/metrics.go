/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	kitmetrics "github.com/hyperledger/fabric/common/metrics"
)

var (
	configRefresh = kitmetrics.HistogramOpts{
		Namespace: "config",
		Name:      "refresh_duration",
		Help:      "The config refresh duration.",
	}
	configPeriodicRefresh = kitmetrics.HistogramOpts{
		Namespace: "config",
		Name:      "periodic_refresh_duration",
		Help:      "The config periodic refresh duration.",
	}
)

//Metrics contain graphs
type Metrics struct {
	ConfigRefresh         kitmetrics.Histogram
	ConfigPeriodicRefresh kitmetrics.Histogram
}

//NewMetrics create new instance of metrics
func NewMetrics(p kitmetrics.Provider) *Metrics {
	return &Metrics{
		ConfigRefresh:         p.NewHistogram(configRefresh),
		ConfigPeriodicRefresh: p.NewHistogram(configPeriodicRefresh),
	}
}
