/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localprovider

import (
	"fmt"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("txnsnap")

// Factory is configured with local provider
type Factory struct {
	defsvc.ProviderFactory
	LocalPeer *api.PeerConfig
}

// CreateDiscoveryProvider returns a new implementation of dynamic discovery provider
func (l *Factory) CreateDiscoveryProvider(config coreApi.Config, fabPvdr fabApi.InfraProvider) (fabApi.DiscoveryProvider, error) {
	logger.Debug("create local Provider Impl")
	return &impl{config, l.LocalPeer}, nil
}

// impl implements a LocalProviderFactory
type impl struct {
	clientConfig coreApi.Config
	localPeer    *api.PeerConfig
}

// CreateDiscoveryService return impl of local discovery service
func (l *impl) CreateDiscoveryService(channelID string) (fabApi.DiscoveryService, error) {
	return &localDiscoveryService{l.clientConfig, l.localPeer}, nil
}

// localDiscoveryService struct
type localDiscoveryService struct {
	clientConfig coreApi.Config
	localPeer    *api.PeerConfig
}

// GetPeers return []sdkapi.Peer
func (s *localDiscoveryService) GetPeers() ([]fabApi.Peer, error) {
	peer, err := peer.New(s.clientConfig, peer.WithURL(fmt.Sprintf("%s:%d", s.localPeer.Host,
		s.localPeer.Port)), peer.WithServerName(""), peer.WithMSPID(string(s.localPeer.MSPid)))
	if err != nil {
		return nil, err
	}
	logger.Debugf("return local peer(%v) from GetPeers DiscoveryService", peer)
	return []fabApi.Peer{peer}, nil

}
