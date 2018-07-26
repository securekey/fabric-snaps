/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localprovider

import (
	"fmt"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	txsnapconfig "github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// Factory is configured with local provider
type Factory struct {
	defsvc.ProviderFactory
	LocalPeer           *api.PeerConfig
	LocalPeerTLSCertPem []byte
}

// CreateLocalDiscoveryProvider returns a new implementation of a dynamic discovery provider
// that always returns the local peer
func (l *Factory) CreateLocalDiscoveryProvider(config fabApi.EndpointConfig) (fabApi.LocalDiscoveryProvider, error) {
	logger.Debug("create local Provider Impl")
	return &impl{config, l.LocalPeer, l.LocalPeerTLSCertPem}, nil
}

// impl implements a LocalProviderFactory
type impl struct {
	clientConfig        fabApi.EndpointConfig
	localPeer           *api.PeerConfig
	localPeerTLSCertPem []byte
}

// CreateLocalDiscoveryService returns impl of local discovery service
func (l *impl) CreateLocalDiscoveryService(mspID string) (fabApi.DiscoveryService, error) {
	return &localDiscoveryService{l.clientConfig, l.localPeer, l.localPeerTLSCertPem}, nil
}

// localDiscoveryService struct
type localDiscoveryService struct {
	clientConfig        fabApi.EndpointConfig
	localPeer           *api.PeerConfig
	localPeerTLSCertPem []byte
}

// GetPeers return []sdkapi.Peer
func (s *localDiscoveryService) GetPeers() ([]fabApi.Peer, error) {
	url := fmt.Sprintf("%s:%d", s.localPeer.Host, s.localPeer.Port)
	peerConfig, ok := s.clientConfig.PeerConfig(url)
	if !ok {
		return nil, errors.Errorf(errors.MissingConfigDataError, "unable to find peer config for url [%s]", url)
	}

	networkPeer, err := txsnapconfig.NewNetworkPeer(peerConfig, string(s.localPeer.MSPid), s.localPeerTLSCertPem)
	if err != nil {
		logger.Errorf("Error creating network peer for [%s]", url)
		return nil, err
	}

	peer, err := peer.New(s.clientConfig, peer.FromPeerConfig(networkPeer))
	if err != nil {
		return nil, fmt.Errorf("error creating new peer: %v", err)
	}
	logger.Debugf("return local peer(%+v) from GetPeers DiscoveryService", peer)
	return []fabApi.Peer{peer}, nil

}
