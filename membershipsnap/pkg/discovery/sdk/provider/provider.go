/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkapi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/sdk/service"
)

const (
	cacheRefreshInterval time.Duration = 60 * time.Second
)

// ChannelUser contains user(identity) info to be used for specific channel
type ChannelUser struct {
	ChannelID string
	UserID    sdkapi.IdentityContext
}

// Impl implements a DiscoveryProvider
// the users must be pre-enrolled
type Impl struct {
	clientConfig    apiconfig.Config
	users           []ChannelUser
	refreshInterval time.Duration
}

// New will kickstart a new service provider to be used to spawn a new membership discovery service
func New(clientConfig apiconfig.Config, cusers []ChannelUser, refreshDelay time.Duration) *Impl {
	cacheRefresh := refreshDelay
	if cacheRefresh == 0 {
		cacheRefresh = cacheRefreshInterval
	}
	return &Impl{
		clientConfig:    clientConfig,
		users:           cusers,
		refreshInterval: cacheRefresh,
	}
}

// NewDiscoveryService will create a new membership service
func (p *Impl) NewDiscoveryService(channelID string) (sdkapi.DiscoveryService, error) {
	var channelUser *ChannelUser
	for _, p := range p.users {
		if p.ChannelID == channelID {
			channelUser = &p
			break
		}
	}

	if channelUser == nil {
		return nil, errors.New("Must provide user for channel")
	}

	return service.New(channelID, p.clientConfig, channelUser.UserID, p.refreshInterval), nil
}
