/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"fmt"

	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/pkg/errors"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service/channelpeer"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

// MemSnapService struct
type MemSnapService struct {
	channelID    string
	clientConfig coreApi.Config
	service      memserviceapi.Service
}

// New return MemSnapService
func New(channelID string, clientConfig coreApi.Config, service memserviceapi.Service) *MemSnapService {

	return &MemSnapService{
		channelID:    channelID,
		clientConfig: clientConfig,
		service:      service,
	}
}

// GetPeers return []sdkapi.Peer
func (s *MemSnapService) GetPeers() ([]fabApi.Peer, error) {
	if s.service == nil {
		var err error
		s.service, err = memservice.Get()
		if err != nil {
			return nil, errors.Wrap(err, "error getting membership service")
		}
	}
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

		peerConfig, err := s.clientConfig.PeerConfigByURL(endpoint.GetEndpoint())
		if err != nil {
			return nil, fmt.Errorf("error get peer config by url: %v", err)
		}
		peer, err := peer.New(s.clientConfig, peer.FromPeerConfig(&coreApi.NetworkPeer{PeerConfig: *peerConfig, MSPID: string(endpoint.GetMSPid())}))
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
