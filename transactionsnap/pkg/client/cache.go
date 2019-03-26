/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"time"

	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/securekey/fabric-snaps/util/errors"
)

// CacheKey config cache reference cache key
type CacheKey interface {
	lazycache.Key
	ChannelID() string
	ServiceProviderFactory() apisdk.ServiceProviderFactory
	ConfigProvider() ConfigProvider
	MetricsProvider() *Metrics
}

// cacheKey holds a key for the cache
type cacheKey struct {
	channelID              string
	configProvider         ConfigProvider
	serviceProviderFactory apisdk.ServiceProviderFactory
	metrics                *Metrics
}

func newCacheKey(channelID string, configProvider ConfigProvider, serviceProviderFactory apisdk.ServiceProviderFactory, metrics *Metrics) *cacheKey {
	return &cacheKey{
		channelID:              channelID,
		configProvider:         configProvider,
		serviceProviderFactory: serviceProviderFactory,
		metrics:                metrics,
	}
}

// String returns the channel ID
func (k *cacheKey) String() string {
	return k.channelID
}

// ChannelID returns the channelID
func (k *cacheKey) ChannelID() string {
	return k.channelID
}

// ServiceProviderFactory returns the provider factory  reference
func (k *cacheKey) ServiceProviderFactory() apisdk.ServiceProviderFactory {
	return k.serviceProviderFactory
}

// ConfigProvider returns the config provider
func (k *cacheKey) ConfigProvider() ConfigProvider {
	return k.configProvider
}

// MetricsProvider returns the metrics provider
func (k *cacheKey) MetricsProvider() *Metrics {
	return k.metrics
}

func newRefCache(refresh time.Duration) *lazycache.Cache {
	initializer := func(key lazycache.Key) (interface{}, error) {
		ck, ok := key.(CacheKey)
		if !ok {
			return nil, errors.New(errors.GeneralError, "unexpected cache key")
		}
		return lazyref.New(
			newInitializer(ck.ChannelID(), ck.ConfigProvider(), ck.ServiceProviderFactory(), ck.MetricsProvider()),
			lazyref.WithRefreshInterval(lazyref.InitImmediately, refresh),
		), nil
	}
	return lazycache.New("Client_Cache", initializer)
}

func newInitializer(channelID string, configProvider ConfigProvider, serviceProviderFactory apisdk.ServiceProviderFactory, metrics *Metrics) lazyref.Initializer {
	var client *clientImpl
	return func() (interface{}, error) {
		newClient, err := checkClient(channelID, client, configProvider, serviceProviderFactory, metrics)
		if err != nil {
			return nil, err
		}
		client = newClient
		return client, nil
	}
}

func checkClient(channelID string, currentClient *clientImpl, configProvider ConfigProvider, serviceProviderFactory apisdk.ServiceProviderFactory, metrics *Metrics) (*clientImpl, errors.Error) {
	cfg, err := configProvider(channelID)
	if err != nil {
		return nil, errors.WithMessage(errors.InitializeConfigError, err, "Failed to initialize config")
	}
	if cfg == nil || cfg.GetConfigBytes() == nil {
		return nil, errors.New(errors.InitializeConfigError, "config is nil")
	}
	cfgHash := generateHash(cfg.GetConfigBytes())

	var currentHash string
	if currentClient != nil {
		currentHash = currentClient.configHash
		logger.Debugf("Checking if client needs to be updated for channel [%s]. Current config hash [%s], new config hash [%s].", channelID, currentHash, cfgHash)
		if cfgHash == currentHash {
			logger.Debugf("The client config was not changed for channel [%s].", channelID)
			return currentClient, nil
		}
	}

	logger.Infof("The client config was updated for channel [%s]. Existing hash [%s] new hash [%s]. Initializing new SDK ...", channelID, currentHash, cfgHash)

	newClient, e := newClient(channelID, cfg, serviceProviderFactory, currentClient, metrics)
	if e != nil {
		return nil, e
	}

	logger.Infof("New client [%s] successfully created on channel [%s].", newClient.configHash, channelID)

	if currentClient != nil {
		// Close the old client in the background
		go func() {
			logger.Debugf("Closing old client [%s] on channel [%s] ...", currentClient.configHash, channelID)
			if !currentClient.Close() {
				logger.Warnf("Unable to close old client [%s] on channel [%s]", currentClient.configHash, channelID)
			} else {
				logger.Debugf("... old client [%s] successfully closed on channel [%s]", currentClient.configHash, channelID)
			}
		}()
	}

	return newClient, nil
}
