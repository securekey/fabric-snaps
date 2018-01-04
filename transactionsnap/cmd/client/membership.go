/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/pkg/errors"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/channelpeer"
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
		peers = append(peers, channelpeer.New(peer, channelID, endpoint.LedgerHeight, manager))
	}

	return peers, nil
}

func formatQueryError(channel string, err error) error {
	return fmt.Errorf("Error querying peers on channel %s: %s", channel, err)
}
