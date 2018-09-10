/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bcinfo

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/hyperledger/fabric/core/peer"
	cb "github.com/hyperledger/fabric/protos/common"
)

const (
	defaultRefreshInterval = 500 * time.Millisecond
)

// Provider implements a caching Blockchain Info Provider
type Provider struct {
	cache *lazycache.Cache
}

// ledger is an abstraction of the peer ledger
type ledger interface {
	GetBlockchainInfo() (*cb.BlockchainInfo, error)
}

// Singleton provider
var provider = createProvider()

// ledgerPrvdr returns the peer ledger
var ledgerPrvdr = func(channelID string) ledger {
	return peer.GetLedger(channelID)
}

// NewProvider returns a Blockchain Info Provider
func NewProvider() *Provider {
	return provider
}

// GetBlockchainInfo delegates to the peer to return basic info about the blockchain
func (l *Provider) GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error) {
	bcinfo, err := l.cache.Get(lazycache.NewStringKey(channelID))
	if err != nil {
		return nil, err
	}
	return bcinfo.(*cb.BlockchainInfo), nil
}

func createProvider() *Provider {
	// TODO: Make refresh interval configurable
	refreshInterval := defaultRefreshInterval

	return &Provider{
		cache: lazycache.New("bcinfo_cache",
			func(key lazycache.Key) (interface{}, error) {
				return ledgerPrvdr(key.String()).GetBlockchainInfo()
			},
			lazyref.WithRefreshInterval(lazyref.InitImmediately, refreshInterval),
		),
	}
}
