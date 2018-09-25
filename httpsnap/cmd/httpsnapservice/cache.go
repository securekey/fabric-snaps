/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpsnapservice

import (
	"encoding/hex"
	"time"

	"fmt"

	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	httpsnapApi "github.com/securekey/fabric-snaps/httpsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

var keyCache *lazycache.Cache
var cacheLoad sync.Once

type cacheKey struct {
	ski            []byte
	cryptoProvider string
}

//String return string value for cacheKey
func (key *cacheKey) String() string {
	return fmt.Sprintf("%s_%s", hex.EncodeToString(key.ski), key.cryptoProvider)
}

//newKeyCache creates new lazycache instance of key by SKI cache
func newKeyCache(refresh time.Duration) *lazycache.Cache {
	return lazycache.New(
		"HttpSnap_KeyBySKI_Cache",
		initGetKeyBySKI(),
		lazyref.WithRefreshInterval(lazyref.InitImmediately, refresh),
	)
}

//getKey returns cryptosuite by SKI provided
// uses cache if config.KeyCacheEnabled
// if reload is true, then force updates value in cache before returning
func getKey(ski []byte, config httpsnapApi.Config, provider string, reload bool) (bccsp.Key, error) {

	if !config.IsKeyCacheEnabled() {
		return getKeyBySKI(ski, provider)
	}

	cacheLoad.Do(func() {
		keyCache = newKeyCache(config.KeyCacheRefreshInterval())
		//anyway, loading first time, no need to reload
		reload = false
	})

	key := &cacheKey{ski: ski, cryptoProvider: provider}

	if reload {
		keyCache.Delete(key)
	}

	ref, err := keyCache.Get(key)
	if err != nil {
		return nil, err
	}

	return ref.(bccsp.Key), nil
}

//initGetKeyBySKI initializer for key by SKI cache
func initGetKeyBySKI() lazycache.EntryInitializer {
	return func(key lazycache.Key) (interface{}, error) {
		cKey := key.(*cacheKey)
		return getKeyBySKI(cKey.ski, cKey.cryptoProvider)
	}
}

//getKeyBySKI returns cryptosuite key by SKI and crypto provider provided
func getKeyBySKI(ski []byte, provider string) (bccsp.Key, error) {
	//Get cryptosuite from peer bccsp pool
	cryptoSuite, e := factory.GetBCCSP(provider)
	if e != nil {
		return nil, errors.WithMessage(errors.CryptoConfigError, e, "failed to get crypto suite for httpsnap")
	}
	//Get private key using SKI
	pk, e := cryptoSuite.GetKey(ski)
	if e != nil {
		return nil, errors.Wrap(errors.GetKeyError, e, "failed to get private key from SKI")
	}
	return pk, nil
}
