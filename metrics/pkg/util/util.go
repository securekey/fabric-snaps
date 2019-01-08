package util

import (
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/core/peer"
)

func GetMetricsInstance() metrics.Provider {
	metricsInstance := peer.GetMetricsProvider()
	if metricsInstance == nil {
		metricsInstance = &disabled.Provider{}
	}
	return metricsInstance
}
