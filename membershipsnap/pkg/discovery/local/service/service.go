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
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service/channelpeer"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

// MemSnapService struct
type MemSnapService struct {
	channelID    string
	clientConfig coreApi.Config
}

// New return MemSnapService
func New(channelID string, clientConfig coreApi.Config) *MemSnapService {
	return &MemSnapService{
		channelID:    channelID,
		clientConfig: clientConfig,
	}
}

// GetPeers return []sdkapi.Peer
func (s *MemSnapService) GetPeers() ([]fabApi.Peer, error) {
	memService, err := memservice.Get()
	if err != nil {
		return nil, errors.Wrap(err, "error getting membership service")
	}

	peerEndpoints, err := memService.GetPeersOfChannel(s.channelID)
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

		peer, err := peer.New(s.clientConfig, peer.WithURL(endpoint.GetEndpoint()), peer.WithServerName(""), peer.WithMSPID(string(endpoint.GetMSPid())))
		if err != nil {
			return nil, fmt.Errorf("Error creating new peer: %s", err)
		}
		channelPeer, err := channelpeer.New(peer, s.channelID, endpoint.LedgerHeight)
		if err != nil {
			return nil, err
		}
		peers = append(peers, channelPeer)
	}

	return peers, nil
}
