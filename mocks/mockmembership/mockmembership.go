/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockmembership

import (
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
)

// Service is a mock membership service
type Service struct {
	Peers          []*memserviceapi.PeerEndpoint
	PeersOfChannel map[string][]*memserviceapi.PeerEndpoint
	Error          error
}

// GetAllPeers returns all peers on the gossip network
func (s *Service) GetAllPeers() []*memserviceapi.PeerEndpoint {
	return s.Peers
}

// GetPeersOfChannel returns all peers on the gossip network joined to the given channel
func (s *Service) GetPeersOfChannel(channelID string) ([]*memserviceapi.PeerEndpoint, error) {
	if s.Error != nil {
		return nil, s.Error
	}
	return s.PeersOfChannel[channelID], nil
}
