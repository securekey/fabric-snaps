/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
)

// Impl implements a DiscoveryProvider that may be
// used by other snaps localy (in the peer process)
type Impl struct {
	clientConfig coreApi.Config
}

// New return Impl
func New(clientConfig coreApi.Config) *Impl {
	return &Impl{
		clientConfig: clientConfig,
	}
}

// CreateDiscoveryService return impl of DiscoveryService
func (p *Impl) CreateDiscoveryService(channelID string) (fabApi.DiscoveryService, error) {
	return service.New(channelID, p.clientConfig, nil), nil
}
