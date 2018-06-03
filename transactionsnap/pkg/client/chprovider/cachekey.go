/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chprovider

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

// cacheKey holds a key for the provider cache
type cacheKey struct {
	channelID string
	context   fab.ClientContext
}

// newCacheKey returns a new cacheKey
func newCacheKey(ctx fab.ClientContext, channelID string) (*cacheKey, error) {
	return &cacheKey{
		channelID: channelID,
		context:   ctx,
	}, nil
}

// String returns the key as a string
func (k *cacheKey) String() string {
	return k.channelID
}
