/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package mocks

import (
	"fmt"

	"github.com/securekey/fabric-extension/kevlar/config"
	"github.com/securekey/fabric-extension/kevlar/fabric/client"
)

type MockMembershipManager struct {
	TestPeers config.PeerConfigs
}

func (m *MockMembershipManager) GetPeersOfChannel(channel string,
	enablePolling bool) client.ChannelMembership {
	var testPeers config.PeerConfigs
	if len(m.TestPeers) == 0 {
		for i := 0; i < 10; i++ {
			peerConfig := config.PeerConfig{Host: fmt.Sprintf("testHost%d", i)}
			testPeers = append(testPeers, peerConfig)
		}
	} else {
		testPeers = m.TestPeers
	}
	return client.ChannelMembership{Peers: testPeers}
}
