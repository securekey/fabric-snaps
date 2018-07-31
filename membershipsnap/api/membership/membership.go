/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

// Service provides functions to query peers
type Service interface {
	// GetAllPeers returns all peers in the Gossip network
	GetAllPeers() []*PeerEndpoint

	// GetPeersOfChannel returns all peers on the Gossip network that are joined to the given channel
	GetPeersOfChannel(channelID string) ([]*PeerEndpoint, error)
}
