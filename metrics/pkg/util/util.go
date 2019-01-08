/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/peer"
)

//GetMetricsInstance return metrics instance
func GetMetricsInstance() metrics.Provider {
	metricsInstance := peer.GetMetricsProvider()
	if metricsInstance == nil {
		metricsInstance = &disabled.Provider{}
	}
	return metricsInstance
}