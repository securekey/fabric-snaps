/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"regexp"

	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/peer"
)

var reg = regexp.MustCompile("[^a-zA-Z0-9_]+")

//GetMetricsInstance return metrics instance
func GetMetricsInstance() metrics.Provider {
	metricsInstance := peer.GetMetricsProvider()
	if metricsInstance == nil {
		metricsInstance = &disabled.Provider{}
	}
	return metricsInstance
}

func FilterMetricName(name string) string {
	return reg.ReplaceAllString(name, "_")
}
