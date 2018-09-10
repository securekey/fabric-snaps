/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channelpeer

import (
	"fmt"

	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

// ChannelPeer extends Peer and adds channel ID and block height
type ChannelPeer struct {
	fabApi.Peer
	channelID   string
	blockHeight uint64
}

// New creates a new ChannelPeer
func New(peer fabApi.Peer, channelID string, blockHeight uint64) (*ChannelPeer, error) {
	return &ChannelPeer{
		Peer:        peer,
		channelID:   channelID,
		blockHeight: blockHeight,
	}, nil
}

// ChannelID returns the channel ID of the ChannelPeer
func (p *ChannelPeer) ChannelID() string {
	return p.channelID
}

// BlockHeight returns the block height of the peer in the channel
func (p *ChannelPeer) BlockHeight() uint64 {
	return p.blockHeight
}

// String returns the string representation of the ChannelPeer
func (p *ChannelPeer) String() string {
	return fmt.Sprintf("[%s] - [%s] - Height[%d]\n", p.MSPID(), p.URL(), p.BlockHeight())
}
