/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"regexp"

	"strings"

	kitstatsd "github.com/go-kit/kit/metrics/statsd"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/common/metrics"
	"github.com/hyperledger/fabric/common/metrics/disabled"
	"github.com/hyperledger/fabric/common/metrics/prometheus"
	"github.com/hyperledger/fabric/common/metrics/statsd"
	"github.com/securekey/fabric-snaps/util/configcache"
	"github.com/securekey/fabric-snaps/util/errors"
)

const (
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
)

var reg = regexp.MustCompile("[^a-zA-Z0-9_]+")
var peerConfigCache = configcache.New(peerConfigFileName, cmdRootPrefix, "/etc/hyperledger/fabric")
var log = logging.NewLogger("metricsutil")
var provider metrics.Provider

//GetMetricsInstance return metrics instance
func GetMetricsInstance() metrics.Provider {
	if provider == nil {
		panic("provider instance is nil you need to call InitializeMetricsProvider first")
	}
	return provider
}

//FilterMetricName filter metric name
func FilterMetricName(name string) string {
	return reg.ReplaceAllString(name, "_")
}

// InitializeMetricsProvider initialize metrics provider
func InitializeMetricsProvider(peerConfigPath string) error {
	peerConfig, err := peerConfigCache.Get(peerConfigPath)
	if err != nil {
		return errors.WithMessage(errors.InitializeConfigError, err, "Failed to get peer config from cache")
	}
	providerType := peerConfig.Get("metrics.provider")
	switch providerType {
	case "statsd":
		prefix := peerConfig.GetString("metrics.statsd.prefix")
		if prefix != "" && !strings.HasSuffix(prefix, ".") {
			prefix = prefix + "."
		}

		ks := kitstatsd.New(prefix, &logger{})
		provider = &statsd.Provider{Statsd: ks}
		return nil

	case "prometheus":
		provider = &prometheus.Provider{}
		return nil

	default:
		if providerType != "disabled" {
			log.Warnf("Unknown provider type: %s; metrics disabled", providerType)
		}
		provider = &disabled.Provider{}
		return nil
	}
}

type logger struct {
}

func (l *logger) Log(keyvals ...interface{}) error {
	log.Warn(keyvals...)
	return nil
}
