/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package balancer

import (
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client/lbp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreferLocalPeerBalancer(t *testing.T) {
	peer0 := mocks.NewMockPeer("p0", "peer0.test.com:9898")
	peer1 := mocks.NewMockPeer("p1", "peer1.test.com:9898")
	peer2 := mocks.NewMockPeer("p2", "peer2.test.com:9898")

	localPeerURL := peer2.URL()
	balancer := NewPreferPeer(localPeerURL, lbp.NewRoundRobin())

	// Ensure the local peer is always chosen since it's in the list
	for i := 0; i < 5; i++ {
		chosenPeer, err := balancer.Choose([]fab.Peer{peer0, peer1, peer2})
		require.NoError(t, err)
		assert.Equalf(t, localPeerURL, chosenPeer.URL(), "local peer should have been chosen since it's in the provided list of peers")
	}

	// Ensure the local peer is never chosen since it's not in the list
	for i := 0; i < 5; i++ {
		chosenPeer, err := balancer.Choose([]fab.Peer{peer0, peer1})
		require.NoError(t, err)
		assert.NotEqualf(t, localPeerURL, chosenPeer.URL(), "local peer should not have been chosen since it's not in the provided list of peers")
	}
}
