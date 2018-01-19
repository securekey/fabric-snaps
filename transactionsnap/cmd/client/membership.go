/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"sync"

	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/channelpeer"
	"github.com/securekey/fabric-snaps/util/errors"
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
		return nil, errors.Wrap(errors.GeneralError, err, "error getting membership service")
	}

	peerEndpoints, err := memService.GetPeersOfChannel(channelID)
	if err != nil {
		return nil, errors.Wrapf(errors.GeneralError, err, "error querying for peers on channel [%s]", channelID)
	}
	peers, err := parsePeerEndpoints(channelID, peerEndpoints, config)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error parsing peer endpoints")
	}
	return peers, nil

}

func parsePeerEndpoints(channelID string, endpoints []*protosPeer.PeerEndpoint, config api.Config) ([]api.ChannelPeer, error) {
	var peers []api.ChannelPeer
	clientInstance, err := GetInstance(channelID, config)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed client GetInstance")
	}

	for _, endpoint := range endpoints {
		enpoint := config.GetGRPCProtocol() + endpoint.GetEndpoint()
		peer, err := sdkFabApi.NewPeer(enpoint, "", "", clientInstance.GetConfig())
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "Error creating new peer")
		}
		peer.SetMSPID(string(endpoint.GetMSPid()))
		peers = append(peers, channelpeer.New(peer, channelID, endpoint.LedgerHeight, manager))
	}

	return peers, nil
}

func formatQueryError(channel string, err error) error {
	return errors.Errorf(errors.GeneralError, "Error querying peers on channel %s: %s", channel, err)
}
