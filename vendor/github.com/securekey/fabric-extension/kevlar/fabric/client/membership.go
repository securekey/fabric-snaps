/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package client

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/securekey/fabric-extension/kevlar/api"
	"github.com/securekey/fabric-extension/kevlar/config"
)

const (
	peerProviderSCC      = "mscc"
	peerProviderfunction = "getPeersOfChannel"
)

// ChannelMembership defines membership for a channel
type ChannelMembership struct {
	// Peers on the channel
	Peers config.PeerConfigs
	// PollingEnabled is polling for membership enabled for this channel
	PollingEnabled bool
	// QueryError Error from the last query/polling operation
	QueryError error
}

// MembershipManager maintains a peer membership lists on channels
type MembershipManager interface {
	// GetPeersOfChannel returns the peers on the given channel. It returns
	// ChannelMembership.QueryError is there was an error querying or polling
	// peers on the channel. It also returns the last known membership list
	// in case there was a polling error
	// @param {string} name of the channel
	// @param {bool} enable membership polling for this channel
	// @returns {ChannelMembership} channel membership object
	GetPeersOfChannel(string, bool) ChannelMembership
}

type membershipManagerImpl struct {
	sync.RWMutex
	peersOfChannel map[string]ChannelMembership
}

var manager *membershipManagerImpl
var membershipSyncOnce sync.Once

const (
	defaultPollInterval = 5 * time.Second
)

// GetMembershipInstance returns an instance of the membership manager
func GetMembershipInstance() MembershipManager {
	membershipSyncOnce.Do(func() {
		peersOfChannel := make(map[string]ChannelMembership)
		manager = &membershipManagerImpl{
			peersOfChannel: peersOfChannel,
		}

		go manager.pollPeersOfChannel()
	})
	return manager
}

func (m *membershipManagerImpl) GetPeersOfChannel(channel string,
	enablePolling bool) ChannelMembership {
	m.RLock()
	membership := m.peersOfChannel[channel]
	m.RUnlock()

	if membership.Peers == nil {
		peers, err := queryPeersOfChannel(channel)
		membership = ChannelMembership{
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
	pollInterval := config.GetMembershipPollInterval()
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

			peers, err := queryPeersOfChannel(channel)
			if err != nil {
				logger.Warningf("Error polling peers of channel %s: %s", channel, err)
			}

			m.Lock()
			m.peersOfChannel[channel] = ChannelMembership{Peers: peers, QueryError: err}
			m.Unlock()
		}
		time.Sleep(time.Second * pollInterval)
	}
}

func queryPeersOfChannel(channel string) (config.PeerConfigs, error) {
	response, err := queryChaincode(channel, peerProviderSCC, []string{peerProviderfunction, channel})
	if err != nil {
		return nil, fmt.Errorf("error querying for peers on channel [%s]: %s", channel, err)
	}

	// return unmarshalled response
	peerEndpoints := &api.PeerEndpoints{}
	err = proto.Unmarshal(response.ProposalResponse.Response.Payload, peerEndpoints)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling response: %s Raw Payload: %+v",
			err, response.ProposalResponse.Response.Payload)
	}
	peers, err := parsePeerEndpoints(peerEndpoints)
	if err != nil {
		return nil, fmt.Errorf("Error parsing peer endpoints: %s", err)
	}
	return peers, nil
}

func queryChaincode(channelID string, ccID string, args []string) (*sdkApi.TransactionProposalResponse, error) {
	logger.Debugf("queryChaincode channelID:%s", channelID)
	client, err := GetInstance()
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

	proposal, err := channel.CreateTransactionProposal(ccID, channelID, args, false, nil)
	if err != nil {
		return nil, formatQueryError(channelID, err)
	}

	// Query the anchor peers in order until we get a response
	var queryErrors []string
	var response *sdkApi.TransactionProposalResponse
	anchors := channel.AnchorPeers()
	if anchors == nil || len(anchors) == 0 {
		return nil, fmt.Errorf("GetAnchorPeers didn't return any peer")
	}
	for _, anchor := range anchors {
		// Load anchor peer
		peer, err := sdkFabApi.NewPeer(fmt.Sprintf("%s:%d", anchor.Host,
			anchor.Port), config.GetTLSRootCertPath(), "", client.GetConfig())
		if err != nil {
			queryErrors = append(queryErrors, err.Error())
			continue
		}
		// Send query to anchor peer
		responses, err := channel.SendTransactionProposal(proposal,
			0, []sdkApi.Peer{peer})
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

func parsePeerEndpoints(endpoints *api.PeerEndpoints) (config.PeerConfigs, error) {
	var err error
	peers := config.PeerConfigs{}
	localPeer, err := config.GetLocalPeer()
	if err != nil {
		return nil, err
	}
	for _, endpoint := range endpoints.GetEndpoints() {
		address := strings.Split(endpoint.Endpoint, ":")
		if len(address) != 2 {
			return nil, fmt.Errorf("Invalid endpoint: %s. Expected host:port", endpoint.Endpoint)
		}
		peer := config.PeerConfig{}
		peer.Host = address[0]
		peer.EventHost = address[0]
		peer.Port, err = strconv.Atoi(address[1])
		if err != nil {
			return nil, fmt.Errorf("Invalid endpoint: %s. Expected host:port", endpoint.Endpoint)
		}
		peer.EventPort = localPeer.EventPort
		peer.MSPid = endpoint.MSPid

		peers = append(peers, peer)
	}

	return peers, nil
}

func formatQueryError(channel string, err error) error {
	return fmt.Errorf("Error querying peers on channel %s: %s", channel, err)
}
