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
	//"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

// CacheKey config cache reference cache key
type CacheKey interface {
	lazycache.Key
	ChannelID() string
	TxnSnapConfig() api.Config
	ServiceProviderFactory() apisdk.ServiceProviderFactory
}

// cacheKey holds a key for the cache
type cacheKey struct {
	channelID              string
	txnSnapConfig          api.Config
	serviceProviderFactory apisdk.ServiceProviderFactory
}

func newCacheKey(channelID string, txnSnapConfig api.Config, serviceProviderFactory apisdk.ServiceProviderFactory) *cacheKey {
	return &cacheKey{
		txnSnapConfig:          txnSnapConfig,
		channelID:              channelID,
		serviceProviderFactory: serviceProviderFactory,
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

// TxnSnapConfig returns the transaction snap config reference
func (k *cacheKey) TxnSnapConfig() api.Config {
	return k.txnSnapConfig
}

// ServiceProviderFactory returns the provider factory  reference
func (k *cacheKey) ServiceProviderFactory() apisdk.ServiceProviderFactory {
	return k.serviceProviderFactory
}

// localCacheKey holds a cache key for clients
// with local providers
type localCacheKey struct {
	*cacheKey
}

func newLocalCacheKey(channelID string, txnSnapConfig api.Config, serviceProviderFactory apisdk.ServiceProviderFactory) *cacheKey {
	return &cacheKey{
		channelID:              "lp" + channelID,
		txnSnapConfig:          txnSnapConfig,
		serviceProviderFactory: serviceProviderFactory,
	}
}

func newRefCache(refresh time.Duration) *lazycache.Cache {
	initializer := func(key lazycache.Key) (interface{}, error) {
		ck, ok := key.(*cacheKey)
		if !ok {
			return nil, errors.New("unexpected cache key")
		}
		return lazyref.New(
			newInitializer(ck.channelID, ck.txnSnapConfig, ck.serviceProviderFactory),
			lazyref.WithRefreshInterval(lazyref.InitImmediately, refresh),
		), nil
	}
	return lazycache.New("Client_Cache", initializer)
}

func newInitializer(channelID string, txnSnapConfig api.Config, serviceProviderFactory apisdk.ServiceProviderFactory) lazyref.Initializer {
	return func() (interface{}, error) {
		client := &clientImpl{txnSnapConfig: txnSnapConfig}
		err := client.initialize(channelID, serviceProviderFactory)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
}
