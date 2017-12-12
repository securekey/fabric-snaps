/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	protosPeer "github.com/securekey/fabric-snaps/transactionsnap/api/membership"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/utils"
)

const (
	peerProviderSCC      = "mscc"
	peerProviderfunction = "getPeersOfChannel"
)

var manager *membershipManagerImpl
var membershipSyncOnce sync.Once

const (
	defaultPollInterval = 5 * time.Second
)

type membershipManagerImpl struct {
	sync.RWMutex
	peersOfChannel map[string]api.ChannelMembership
	config         api.Config
}

// GetMembershipInstance returns an instance of the membership manager
func GetMembershipInstance(config api.Config) api.MembershipManager {
	membershipSyncOnce.Do(func() {
		peersOfChannel := make(map[string]api.ChannelMembership)
		manager = &membershipManagerImpl{peersOfChannel: peersOfChannel, config: config}
		go manager.pollPeersOfChannel()
	})
	return manager
}

func (m *membershipManagerImpl) GetPeersOfChannel(channel string,
	enablePolling bool) api.ChannelMembership {
	m.RLock()
	membership := m.peersOfChannel[channel]
	m.RUnlock()

	if membership.Peers == nil {
		peers, err := queryPeersOfChannel(channel, m.config)
		membership = api.ChannelMembership{
			Peers:      peers,
			QueryError: err,
		}
	}
	membership.PollingEnabled = enablePolling

	m.Lock()
	defer m.Unlock()
	m.peersOfChannel[channel] = membership

	return membership
}

func (m *membershipManagerImpl) pollPeersOfChannel() {
	pollInterval := m.config.GetMembershipPollInterval()
	if pollInterval == 0 {
		pollInterval = defaultPollInterval
	}
	// Start polling
	for {
		logger.Debug("Polling peers on all known channels")
		for channel, membership := range m.peersOfChannel {
			if !membership.PollingEnabled {
				continue
			}

			peers, err := queryPeersOfChannel(channel, m.config)
			if err != nil {
				logger.Warnf("Error polling peers of channel %s: %s", channel, err)
			}

			m.Lock()
			m.peersOfChannel[channel] = api.ChannelMembership{Peers: peers, QueryError: err}
			m.Unlock()
		}
		time.Sleep(time.Second * pollInterval)
	}
}

func queryPeersOfChannel(channelID string, config api.Config) ([]sdkApi.Peer, error) {
	response, err := queryChaincode(channelID, peerProviderSCC, []string{peerProviderfunction, channelID}, config)
	if err != nil {
		return nil, fmt.Errorf("error querying for peers on channel [%s]: %s", channelID, err)
	}

	// return unmarshalled response
	peerEndpoints := &protosPeer.PeerEndpoints{}
	err = proto.Unmarshal(response.ProposalResponse.Response.Payload, peerEndpoints)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling response: %s Raw Payload: %+v",
			err, response.ProposalResponse.Response.Payload)
	}
	peers, err := parsePeerEndpoints(channelID, peerEndpoints, config)
	if err != nil {
		return nil, fmt.Errorf("Error parsing peer endpoints: %s", err)
	}
	return peers, nil

}

func queryChaincode(channelID string, ccID string, args []string, config api.Config) (*apitxn.TransactionProposalResponse, error) {
	logger.Debugf("queryChaincode channelID:%s", channelID)
	client, err := GetInstance(channelID, config)
	if err != nil {
		return nil, formatQueryError(channelID, err)
	}

	channel, err := client.NewChannel(channelID)
	if err != nil {
		return nil, formatQueryError(channelID, err)
	}
	err = client.InitializeChannel(channel)
	if err != nil {
		return nil, formatQueryError(channelID, err)
	}

	// Query the anchor peers in order until we get a response
	var queryErrors []string
	var response *apitxn.TransactionProposalResponse
	anchors := channel.AnchorPeers()
	if anchors == nil || len(anchors) == 0 {
		return nil, fmt.Errorf("GetAnchorPeers didn't return any peer")
	}
	for _, anchor := range anchors {
		// Load anchor peer
		//orgCertPool, err := client.GetTLSRootsForOrg(, channel)
		anchor.Host = config.GetGRPCProtocol() + anchor.Host
		peer, err := sdkFabApi.NewPeer(fmt.Sprintf("%s:%d", anchor.Host,
			anchor.Port), config.GetTLSRootCertPath(), "", client.GetConfig())
		if err != nil {
			queryErrors = append(queryErrors, err.Error())
			continue
		}
		// Send query to anchor peer
		request := apitxn.ChaincodeInvokeRequest{
			Targets:      []apitxn.ProposalProcessor{peer},
			Fcn:          args[0],
			Args:         utils.GetByteArgs(args[1:]),
			TransientMap: nil,
			ChaincodeID:  ccID,
		}

		responses, _, err := channel.SendTransactionProposal(request)
		if err != nil {
			queryErrors = append(queryErrors, err.Error())
			continue
		} else if responses[0].Err != nil {
			queryErrors = append(queryErrors, responses[0].Err.Error())
			continue
		} else {
			// Valid response obtained, stop querying
			response = responses[0]
			break
		}
	}
	logger.Debugf("queryErrors: %v", queryErrors)

	// If all queries failed, return error
	if len(queryErrors) == len(anchors) {
		return nil, fmt.Errorf(
			"Error querying peers from all configured anchors for channel %s: %s",
			channelID, strings.Join(queryErrors, "\n"))
	}

	return response, nil
}

func parsePeerEndpoints(channelID string, endpoints *protosPeer.PeerEndpoints, config api.Config) ([]sdkApi.Peer, error) {
	peers := []sdkApi.Peer{}
	clientInstance, err := GetInstance(channelID, config)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints.GetEndpoints() {
		enpoint := config.GetGRPCProtocol() + endpoint.GetEndpoint()
		peer, err := sdkFabApi.NewPeer(enpoint, "", "", clientInstance.GetConfig())
		if err != nil {
			return nil, fmt.Errorf("Error creating new peer: %s", err)
		}
		peer.SetMSPID(string(endpoint.GetMSPid()))
		peers = append(peers, peer)
	}

	return peers, nil
}

func formatQueryError(channel string, err error) error {
	return fmt.Errorf("Error querying peers on channel %s: %s", channel, err)
}
