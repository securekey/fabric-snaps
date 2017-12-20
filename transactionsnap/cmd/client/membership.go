/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/pkg/errors"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var manager *membershipManagerImpl
var membershipSyncOnce sync.Once

type membershipManagerImpl struct {
	config api.Config
}

// GetMembershipInstance returns an instance of the membership manager
func GetMembershipInstance(config api.Config) api.MembershipManager {
	membershipSyncOnce.Do(func() {
		manager = &membershipManagerImpl{
			config: config,
		}
	})
	return manager
}

func (m *membershipManagerImpl) GetPeersOfChannel(channel string) api.ChannelMembership {
	peers, err := queryPeersOfChannel(channel, m.config)
	return api.ChannelMembership{
		Peers:      peers,
		QueryError: err,
	}
}

func queryPeersOfChannel(channelID string, config api.Config) ([]api.ChannelPeer, error) {
	memService, err := memservice.Get()
	if err != nil {
		return nil, errors.Wrap(err, "error getting membership service")
	}

	peerEndpoints, err := memService.GetPeersOfChannel(channelID)
	if err != nil {
		return nil, errors.Wrapf(err, "error querying for peers on channel [%s]", channelID)
	}
	peers, err := parsePeerEndpoints(channelID, peerEndpoints, config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing peer endpoints: %s", err)
	}
	return peers, nil

}

func parsePeerEndpoints(channelID string, endpoints []*protosPeer.PeerEndpoint, config api.Config) ([]api.ChannelPeer, error) {
	var peers []api.ChannelPeer
	clientInstance, err := GetInstance(config)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints {
		enpoint := config.GetGRPCProtocol() + endpoint.GetEndpoint()
		peer, err := sdkFabApi.NewPeer(enpoint, "", "", clientInstance.GetConfig())
		if err != nil {
			return nil, fmt.Errorf("Error creating new peer: %s", err)
		}
		peer.SetMSPID(string(endpoint.GetMSPid()))
		peers = append(peers, NewChannelPeer(peer, channelID, endpoint.LedgerHeight))
	}

	return peers, nil
}

func formatQueryError(channel string, err error) error {
	return fmt.Errorf("Error querying peers on channel %s: %s", channel, err)
}

// ChannelPeerImpl implements ChannelPeer
type ChannelPeerImpl struct {
	sdkApi.Peer
	channelID   string
	blockHeight uint64
}

// NewChannelPeer creates a new ChannelPeer
func NewChannelPeer(peer sdkApi.Peer, channelID string, blockHeight uint64) *ChannelPeerImpl {
	return &ChannelPeerImpl{
		Peer:        peer,
		channelID:   channelID,
		blockHeight: blockHeight,
	}
}

// ChannelID returns the channel ID of the ChannelPeer
func (p *ChannelPeerImpl) ChannelID() string {
	return p.channelID
}

// BlockHeight returns the block height of the peer in the channel
func (p *ChannelPeerImpl) BlockHeight() uint64 {
	return p.blockHeight
}

// GetBlockHeight returns the block height of the peer in the specified channel
func (p *ChannelPeerImpl) GetBlockHeight(channelID string) uint64 {
	if channelID == p.channelID {
		return p.blockHeight
	}

	mem := manager.GetPeersOfChannel(channelID)
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
func (p *ChannelPeerImpl) String() string {
	return fmt.Sprintf("[%s] - [%s] - Height[%d]\n", p.MSPID(), p.URL(), p.BlockHeight())
}
