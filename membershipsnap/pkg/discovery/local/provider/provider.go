/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package provider

import (
	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkapi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
)

// Impl implements a DiscoveryProvider that may be
// used by other snaps localy (in the peer process)
type Impl struct {
	clientConfig apiconfig.Config
}

func New(clientConfig apiconfig.Config) *Impl {
	return &Impl{
		clientConfig: clientConfig,
	}
}

func (p *Impl) NewDiscoveryService(channelID string) (sdkapi.DiscoveryService, error) {
	return service.New(channelID, p.clientConfig), nil
}
