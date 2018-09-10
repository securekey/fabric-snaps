/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"fmt"

	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
)

var config = mocks.NewMockEndpointConfig()

// MockChannelPeer implements fab.Peer and fab.PeerState
type MockChannelPeer struct {
	fabApi.Peer
	blockHeight uint64
}

// BlockHeight returns the block height of the peer in the channel
func (p *MockChannelPeer) BlockHeight() uint64 {
	return p.blockHeight
}

// String returns the string representation of the ChannelPeer
func (p *MockChannelPeer) String() string {
	return fmt.Sprintf("[%s] - [%s] - Height[%d]\n", p.MSPID(), p.URL(), p.BlockHeight())
}

// ChannelHeight specifies the block height for a channel
type ChannelHeight struct {
	ChannelID string
	Height    uint64
}

// New returns a new mock ChannelPeer
func New(name string, mspID string, blockHeight uint64) *MockChannelPeer {
	peer, err := peer.New(config, peer.WithURL("grpc://"+name+":7051"), peer.WithServerName(name), peer.WithMSPID(mspID))
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %s)", err))
	}

	return &MockChannelPeer{
		Peer:        peer,
		blockHeight: blockHeight,
	}
}
