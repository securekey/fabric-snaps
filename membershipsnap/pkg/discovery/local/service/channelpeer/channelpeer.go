/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channelpeer

import (
	"fmt"
	"strings"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	memberapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

var logger = logging.NewLogger("membershipsnap/channelpeer")

// ChannelPeer extends Peer and adds channel ID and block height
type ChannelPeer struct {
	sdkApi.Peer
	channelID   string
	blockHeight uint64
	service     memberapi.Service
}

// New creates a new ChannelPeer
func New(peer sdkApi.Peer, channelID string, blockHeight uint64) (*ChannelPeer, error) {
	memService, err := membership.Get()
	if err != nil {
		return nil, errors.Wrap(err, "error getting membership service")
	}
	return &ChannelPeer{
		Peer:        peer,
		channelID:   channelID,
		blockHeight: blockHeight,
		service:     memService,
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

// GetBlockHeight returns the block height of the peer in the specified channel
func (p *ChannelPeer) GetBlockHeight(channelID string) uint64 {
	if channelID == p.channelID {
		return p.blockHeight
	}

	endpoints, err := p.service.GetPeersOfChannel(channelID)
	if err != nil {
		logger.Errorf("Error querying for peers of channel [%s]: %s\n", channelID, err)
		return 0
	}

	for _, endpoint := range endpoints {
		// p.Url() will be in the for grpc://host:port whereas
		// the endpoint will be in the form host:port
		if strings.Contains(p.URL(), endpoint.Endpoint) {
			return endpoint.LedgerHeight
		}
	}

	logger.Warnf("Peer [%s] not found for channel [%s]\n", p.URL(), channelID)

	return 0
}

// String returns the string representation of the ChannelPeer
func (p *ChannelPeer) String() string {
	return fmt.Sprintf("[%s] - [%s] - Height[%d]\n", p.MSPID(), p.URL(), p.BlockHeight())
}
