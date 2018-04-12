/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"time"
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

// CacheKey holds a key for the cache
type cacheKey struct {
	key                    string
	channelID              string
	txnSnapConfig          api.Config
	serviceProviderFactory apisdk.ServiceProviderFactory
}

// NewCacheKey returns a new CacheKey
func NewCacheKey(channelID string, txnSnapConfig api.Config, serviceProviderFactory apisdk.ServiceProviderFactory) (CacheKey, error) {
	return &cacheKey{
		key:                    channelID,
		txnSnapConfig:          txnSnapConfig,
		channelID:              channelID,
		serviceProviderFactory: serviceProviderFactory,
	}, nil
}

// NewRefCache a cache of configuration references that refreshed with the given interval
func NewRefCache(refresh time.Duration) *lazycache.Cache {
	initializer := func(key lazycache.Key) (interface{}, error) {
		ck, ok := key.(CacheKey)
		if !ok {
			return nil, errors.New("unexpected cache key")
		}
		return NewRef(refresh, ck.ChannelID(), ck.TxnSnapConfig(), ck.ServiceProviderFactory()), nil
	}

	return lazycache.New("Config_Cache", initializer)
}

// String returns the key as a string
func (k *cacheKey) String() string {
	return k.key
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
