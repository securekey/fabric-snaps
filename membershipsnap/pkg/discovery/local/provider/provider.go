/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

// Impl implements a DiscoveryProvider that may be
// used by other snaps localy (in the peer process)
type Impl struct {
	endpointConfig fabApi.EndpointConfig
}

// New return Impl
func New(endpointConfig fabApi.EndpointConfig) *Impl {
	return &Impl{
		endpointConfig: endpointConfig,
	}
}

// CreateDiscoveryService return impl of DiscoveryService
func (p *Impl) CreateDiscoveryService(channelID string) (fabApi.DiscoveryService, error) {
	memService, err := memservice.Get()
	if err != nil {
		return nil, errors.Wrap(err, "error getting membership service")
	}
	return service.New(channelID, p.endpointConfig, memService), nil
}
