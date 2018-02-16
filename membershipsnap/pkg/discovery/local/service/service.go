/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"fmt"

	sdkapi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	"github.com/pkg/errors"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service/channelpeer"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

type MemSnapService struct {
	channelID      string
	clientConfig   apiconfig.Config
	peerTLSEnabled bool
}

func New(channelID string, clientConfig apiconfig.Config) *MemSnapService {
	return &MemSnapService{
		channelID:    channelID,
		clientConfig: clientConfig,
	}
}

func (s *MemSnapService) GetPeers() ([]sdkapi.Peer, error) {
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

func (s *MemSnapService) parsePeerEndpoints(endpoints []*protosPeer.PeerEndpoint) ([]sdkapi.Peer, error) {
	var peers []sdkapi.Peer
	for _, endpoint := range endpoints {
		enpoint := s.getGRPCProtocol() + endpoint.GetEndpoint()

		peer, err := peer.NewPeer(enpoint, s.clientConfig)
		if err != nil {
			return nil, fmt.Errorf("Error creating new peer: %s", err)
		}
		peer.SetMSPID(string(endpoint.GetMSPid()))
		channelPeer, err := channelpeer.New(peer, s.channelID, endpoint.LedgerHeight)
		if err != nil {
			return nil, err
		}
		peers = append(peers, channelPeer)
	}

	return peers, nil
}

func (s *MemSnapService) getGRPCProtocol() string {
	if s.peerTLSEnabled {
		return "grpcs://"
	}
	return "grpc://"
}
