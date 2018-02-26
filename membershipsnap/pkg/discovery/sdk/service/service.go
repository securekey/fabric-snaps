/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkapi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/chpvdr"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
)

// MembershipService contains the dynamic discovery context
type MembershipService struct {
	channelID         string
	clientConfig      apiconfig.Config
	clientIDCtx       sdkapi.IdentityContext
	mtx               sync.RWMutex
	channelPeersCache []sdkapi.Peer
	refreshInterval   time.Duration
	cacheExpired      bool
}

var logger = logging.NewLogger("dynamic-discovery-service")
var instance *MembershipService
var once sync.Once

// New will create a new MembershipService to query the list of member peers on a given channel set in the client configs.
// It will cache the list of peers per channel and automatically refreshes periodically
// assumptions to properly use this service:
// 1. clientConfig is properly set with client's org name
// 2. userID is pre-enrolled at the peers of that channelID: TODO load users from the client config's organizations users
func New(channelID string, clientConfig apiconfig.Config, userID sdkapi.IdentityContext, refreshInterval time.Duration) *MembershipService {
	once.Do(func() {
		instance = &MembershipService{
			channelID:       channelID,
			clientConfig:    clientConfig,
			clientIDCtx:     userID,
			refreshInterval: refreshInterval,
			cacheExpired:    true,
		}
		instance.channelPeersCache = []sdkapi.Peer{}
		logger.Debugf("Created cache instance %v", time.Unix(time.Now().Unix(), 0))
	})

	go func() {
		for {
			instance.GetPeers()
			instance.cacheExpired = false
			time.Sleep(refreshInterval)
			instance.cacheExpired = true // done sleeping, expire cache
			logger.Debugf("Discovery service cache refresh delay expired, executing again: %s", instance.cacheExpired)
		}
	}()
	return instance

}

// GetPeers will invoke the membership snap for the specified channelID to retrieve the list of peers
func (s *MembershipService) GetPeers() ([]sdkapi.Peer, error) {
	if s.cacheExpired {
		s.mtx.Lock()
		defer s.mtx.Unlock()
		logger.Debugf("Refreshing cache instance %v", time.Unix(time.Now().Unix(), 0))
		return s.invokeMembershipSnap()
	}

	return s.channelPeersCache, nil
}

func (s *MembershipService) invokeMembershipSnap() ([]sdkapi.Peer, error) {
	sdk, err := s.newSDK()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get new SDK for membership service invoke")
	}

	ch, err := s.getChannelClient(sdk, s.channelID)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get new channel [%s] for membership service invoke", s.channelID)
	}

	npeers, err := s.clientConfig.ChannelPeers(s.channelID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get client's network peers for membership service invoke")
	}
	var targets []sdkapi.ProposalProcessor
	var nbErrs = 0
	var errArr = []error{}
	for _, p := range npeers {
		sdkPeer, err := sdk.FabricProvider().CreatePeerFromConfig(&p.NetworkPeer)
		logger.Debugf("sdkPeer returned: '%+v' \n", sdkPeer)
		if err != nil {
			nbErrs++
			errArr = append(errArr, err)
			continue
		}
		targets = append(targets, sdkPeer)
	}
	logger.Debugf("nbErrs: %d, npeers length: %d, targets length: %d \n", nbErrs, len(npeers), len(targets))
	if nbErrs == len(npeers) {
		return nil, fmt.Errorf("Failed to get at least 1 peer target from the config for invocation. Nb errors found: %d. Errors: '%s'", nbErrs, errArr)
	}

	logger.Debugf("**** list of network peers to be queried: \n\t%s\n", targets)

	request := sdkapi.ChaincodeInvokeRequest{
		Fcn:          "getPeersOfChannel",
		Args:         [][]byte{[]byte(s.channelID)},
		TransientMap: nil,
		ChaincodeID:  "mscc",
	}

	// TODO: Replace this call with the latest GO SDK's ChannelClient call
	responses, _, err := ch.SendTransactionProposal(request, targets)
	if err != nil {
		return nil, errors.WithMessage(err, "Error sending transaction proposal for invoking membership snap")
	}
	var endpoints []*protosPeer.PeerEndpoint
	for _, p := range responses {
		if p.Err == nil {
			pes := &protosPeer.PeerEndpoints{}
			err := proto.Unmarshal(p.ProposalResponse.GetResponse().Payload, pes)
			if err != nil {
				return nil, errors.Wrapf(err, "Failed to unmarshal proposal response after memebership snap is invoked on channel %s", s.channelID)
			}
			endpoints = append(endpoints, pes.Endpoints...)
		}
	}

	peers, err := s.parsePeerEndpoints(endpoints, s.channelID)
	if err != nil {
		return nil, errors.WithMessage(err, "Error parsing peer endpoints")
	}
	s.channelPeersCache = peers

	return s.channelPeersCache, nil
}

func (s *MembershipService) parsePeerEndpoints(endpoints []*protosPeer.PeerEndpoint, channelID string) ([]sdkapi.Peer, error) {
	var peers []sdkapi.Peer
	for _, endpoint := range endpoints {
		et := endpoint.GetEndpoint()
		peer, err := peer.New(s.clientConfig, peer.WithURL(et))
		if err != nil {
			return nil, errors.WithMessage(err, "Error creating new peer: %s")
		}
		peer.SetMSPID(string(endpoint.GetMSPid()))

		peers = append(peers, peer)
	}

	return peers, nil
}

func (s *MembershipService) newSDK() (*fabsdk.FabricSDK, error) {
	sdk, err := fabsdk.New(fabsdk.WithConfig(s.clientConfig)) //	fabsdk.WithCorePkg(&factories.DefaultCryptoSuiteProviderFactory{ProviderName: s.clientConfig.SecurityProvider()})

	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}
	return sdk, nil
}

func (s *MembershipService) getChannelClient(sdk *fabsdk.FabricSDK, channelID string) (sdkapi.Channel, error) {
	fp := sdk.FabricProvider()
	cp, err := chpvdr.New(fp)
	if err != nil {
		return nil, errors.WithMessage(err, "New channel provider failed")
	}

	cs, err := cp.NewChannelService(s.clientIDCtx, channelID)
	if err != nil {
		return nil, errors.WithMessage(err, "New channel service failed")
	}
	chClient, err := cs.Channel()
	if err != nil {
		return nil, errors.WithMessage(err, "New channel client failed")
	}
	return chClient, nil
}
