/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"fmt"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/pkg/errors"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service/channelpeer"
)

var logger = logging.NewLogger("local-discovery-service")

// MemSnapService struct
type MemSnapService struct {
	channelID      string
	endpointConfig fabApi.EndpointConfig
	service        memserviceapi.Service
}

// New return MemSnapService
func New(channelID string, endpointConfig fabApi.EndpointConfig, service memserviceapi.Service) *MemSnapService {
	if service == nil {
		panic("membership service is nil")
	}
	return &MemSnapService{
		channelID:      channelID,
		endpointConfig: endpointConfig,
		service:        service,
	}
}

// GetPeers return []sdkapi.Peer
func (s *MemSnapService) GetPeers() ([]fabApi.Peer, error) {
	peerEndpoints, err := s.service.GetPeersOfChannel(s.channelID)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying for peers on channel [%s]", s.channelID)
	}

	peers, err := s.parsePeerEndpoints(peerEndpoints)
	if err != nil {
		return nil, fmt.Errorf("Error parsing peer endpoints: %s", err)
	}
	return peers, nil

}

func (s *MemSnapService) parsePeerEndpoints(endpoints []*protosPeer.PeerEndpoint) ([]fabApi.Peer, error) {
	var peers []fabApi.Peer
	for _, endpoint := range endpoints {
		url := endpoint.GetEndpoint()
		if url == "" {
			logger.Warnf("Endpoint for %s has missing url, skipping it in GetPeers()..", endpoint.GetMSPid())
			continue
		}
		peerConfig, err := s.endpointConfig.PeerConfigByURL(url)
		if err != nil {
			return nil, fmt.Errorf("error get peer config by url: %v", err)
		}
		peer, err := peer.New(s.endpointConfig, peer.FromPeerConfig(&fabApi.NetworkPeer{PeerConfig: *peerConfig, MSPID: string(endpoint.GetMSPid())}))
		if err != nil {
			return nil, fmt.Errorf("error creating new peer: %v", err)
		}
		channelPeer, err := channelpeer.New(peer, s.channelID, endpoint.LedgerHeight, s.service)
		if err != nil {
			return nil, err
		}
		peers = append(peers, channelPeer)
	}

	return peers, nil
}
