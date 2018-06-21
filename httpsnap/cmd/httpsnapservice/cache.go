/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	//"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/httpsnap/api"
)

// CacheKey config cache reference cache key
type CacheKey interface {
	lazycache.Key
	HttpSnapConfig() api.Config
}

// cacheKey holds a key for the cache
type cacheKey struct {
	channelID      string
	httpSnapConfig api.Config
}

func newCacheKey(channelID string, httpSnapConfig api.Config) *cacheKey {
	return &cacheKey{
		httpSnapConfig: httpSnapConfig,
		channelID:      channelID,
	}
}

// String returns the channel ID
func (k *cacheKey) String() string {
	return k.channelID
}

// TxnSnapConfig returns the transaction snap config reference
func (k *cacheKey) HttpSnapConfig() api.Config {
	return k.httpSnapConfig
}

func newRefCache(refresh time.Duration) *lazycache.Cache {
	initializer := func(key lazycache.Key) (interface{}, error) {
		ck, ok := key.(CacheKey)
		if !ok {
			return nil, errors.New("unexpected cache key")
		}
		return lazyref.New(
			newInitializer(ck.HttpSnapConfig()),
			lazyref.WithRefreshInterval(lazyref.InitImmediately, refresh),
		), nil
	}
	return lazycache.New("Client_Cache", initializer)
}

func newInitializer(httpSnapConfig api.Config) lazyref.Initializer {
	var serviceImpl *HTTPServiceImpl
	return func() (interface{}, error) {
		if serviceImpl == nil {
			serviceImpl = &HTTPServiceImpl{}
		}
		//Just call init if HTTPServiceImpl instance already there
		serviceImpl.init(httpSnapConfig)

		return serviceImpl, nil
	}
}
