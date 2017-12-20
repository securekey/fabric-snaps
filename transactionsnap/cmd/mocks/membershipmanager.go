/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

// MockMembershipManager implements mock membership manager
type MockMembershipManager struct {
	peerConfigs map[string][]api.ChannelPeer
	err         error
}

// GetPeersOfChannel is mock implementation of GetPeersOfChannel
func (m *MockMembershipManager) GetPeersOfChannel(channelID string) api.ChannelMembership {
	if m.err != nil {
		return api.ChannelMembership{Peers: m.peerConfigs[channelID], QueryError: m.err}
	}
	return api.ChannelMembership{Peers: m.peerConfigs[channelID]}
}

// NewMockMembershipManager creates new mock membership manager
func NewMockMembershipManager(err error) *MockMembershipManager {
	return &MockMembershipManager{peerConfigs: make(map[string][]api.ChannelPeer), err: err}
}

//Add adds peers for channel
func (m *MockMembershipManager) Add(channelID string, peers ...api.ChannelPeer) *MockMembershipManager {
	m.peerConfigs[channelID] = []api.ChannelPeer(peers)
	return m
}
