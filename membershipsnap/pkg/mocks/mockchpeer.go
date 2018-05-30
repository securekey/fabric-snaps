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
	"github.com/securekey/fabric-snaps/membershipsnap/api"
)

var config = mocks.NewMockEndpointConfig()

// MockChannelPeer implements ChannelPeer
type MockChannelPeer struct {
	fabApi.Peer
	channelID    string
	blockHeights map[string]uint64
}

// ChannelID returns the channel ID of the ChannelPeer
func (p *MockChannelPeer) ChannelID() string {
	return p.channelID
}

// BlockHeight returns the block height of the peer in the channel
func (p *MockChannelPeer) BlockHeight() uint64 {
	return p.blockHeights[p.channelID]
}

// GetBlockHeight returns the block height of the peer in the specified channel
func (p *MockChannelPeer) GetBlockHeight(channelID string) uint64 {
	return p.blockHeights[channelID]
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
func New(name string, mspID string, channelID string, blockHeight uint64, chHeights ...ChannelHeight) api.ChannelPeer {
	peer, err := peer.New(config, peer.WithURL("grpc://"+name+":7051"), peer.WithServerName(name), peer.WithMSPID(mspID))
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %s)", err))
	}

	blockHeights := make(map[string]uint64)
	blockHeights[channelID] = blockHeight
	for _, ch := range chHeights {
		blockHeights[ch.ChannelID] = ch.Height
	}

	return &MockChannelPeer{
		Peer:         peer,
		channelID:    channelID,
		blockHeights: blockHeights,
	}
}
