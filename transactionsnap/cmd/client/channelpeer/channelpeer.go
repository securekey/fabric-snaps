/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channelpeer

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("transaction-fabric-client/channelpeer")

// channelPeer implements ChannelPeer
type channelPeer struct {
	sdkApi.Peer
	channelID   string
	blockHeight uint64
	mgr         api.MembershipManager
}

// New creates a new ChannelPeer
func New(peer sdkApi.Peer, channelID string, blockHeight uint64, mgr api.MembershipManager) api.ChannelPeer {
	return &channelPeer{
		Peer:        peer,
		channelID:   channelID,
		blockHeight: blockHeight,
		mgr:         mgr,
	}
}

// ChannelID returns the channel ID of the ChannelPeer
func (p *channelPeer) ChannelID() string {
	return p.channelID
}

// BlockHeight returns the block height of the peer in the channel
func (p *channelPeer) BlockHeight() uint64 {
	return p.blockHeight
}

// GetBlockHeight returns the block height of the peer in the specified channel
func (p *channelPeer) GetBlockHeight(channelID string) uint64 {
	if channelID == p.channelID {
		return p.blockHeight
	}

	mem := p.mgr.GetPeersOfChannel(channelID)
	if mem.QueryError != nil {
		logger.Errorf("Error querying for peers of channel [%s]: %s\n", channelID, mem.QueryError)
		return 0
	}

	for _, peer := range mem.Peers {
		if peer.URL() == p.URL() {
			return peer.BlockHeight()
		}
	}

	logger.Warnf("Peer [%s] not found for channel [%s]\n", p.URL(), channelID)

	return 0
}

// String returns the string representation of the ChannelPeer
func (p *channelPeer) String() string {
	return fmt.Sprintf("[%s] - [%s] - Height[%d]\n", p.MSPID(), p.URL(), p.BlockHeight())
}
