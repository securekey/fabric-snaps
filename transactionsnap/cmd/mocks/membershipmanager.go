/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	client "github.com/securekey/fabric-snaps/transactionsnap/cmd/client"
)

// MockMembershipManager implements mock membership manager
type MockMembershipManager struct {
	peerConfigs map[string][]sdkApi.Peer
	err         error
}

// GetPeersOfChannel is mock implementation of GetPeersOfChannel
func (m *MockMembershipManager) GetPeersOfChannel(channelID string, poll bool) client.ChannelMembership {
	if m.err != nil {
		return client.ChannelMembership{Peers: m.peerConfigs[channelID], QueryError: m.err}
	}
	return client.ChannelMembership{Peers: m.peerConfigs[channelID], PollingEnabled: poll}
}

// NewMockMembershipManager creates new mock membership manager
func NewMockMembershipManager(err error) *MockMembershipManager {
	return &MockMembershipManager{peerConfigs: make(map[string][]sdkApi.Peer), err: err}
}

//Add adds peers for channel
func (m *MockMembershipManager) Add(channelID string, peers ...sdkApi.Peer) *MockMembershipManager {
	m.peerConfigs[channelID] = []sdkApi.Peer(peers)
	return m
}
