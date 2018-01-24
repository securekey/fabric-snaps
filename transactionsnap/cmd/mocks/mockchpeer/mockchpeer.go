/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockchpeer

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	sdkpeer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var config = mocks.NewMockConfig()

// MockChannelPeer implements ChannelPeer
type MockChannelPeer struct {
	sdkApi.Peer
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
	peer, err := sdkpeer.New(config, sdkpeer.WithURL("grpc://"+name+":7051"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}
	peer.SetName(name)
	peer.SetMSPID(mspID)

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
