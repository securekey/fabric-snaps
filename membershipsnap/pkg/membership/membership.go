/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"time"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/service"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("membershipsnap")

const (
	cacheExpiration = 500 * time.Millisecond // TODO: Make configurable
)

// Roles is a set of peer roles
type Roles []string

// HasRole return true if the given role is included in the set
func (r Roles) HasRole(role string) bool {
	if len(r) == 0 {
		// Return true by default in order to be backward compatible
		return true
	}
	for _, r := range r {
		if r == role {
			return true
		}
	}
	return false
}

const (
	// EndorserRole indicates that the peer may be used for endorsements
	EndorserRole = "endorser"
	// CommitterRole indicates that the peer commits transactions
	CommitterRole = "committer"
)

var membershipService = lazyref.New(func() (interface{}, error) {
	return createMembershipService()
})

// mspMap manages a map of PKI IDs to MSP IDs
type mspIDProvider interface {
	GetMSPID(pkiID common.PKIidType) string
}

// Service provides functions to query peers
type Service struct {
	gossipService    service.GossipService
	mspProvider      mspIDProvider
	localMSPID       []byte
	localPeerAddress string
	peers            *lazyref.Reference
	peersOfChannel   *lazycache.Cache
}

// Get returns the Membership Service instance.
// If the service hasn't been initialized yet then
// it will be initialized.
func Get() (memserviceapi.Service, error) {
	service, err := membershipService.Get()
	if err != nil {
		return nil, err
	}
	return service.(memserviceapi.Service), nil
}

func createMembershipService() (*Service, error) {
	memService, err := newService()
	if err != nil {
		errObj := errors.Wrap(errors.SystemError, err, "error initializing membership service")
		logger.Errorf(errObj.GenerateLogMsg())
		return nil, errObj
	}
	return memService, nil
}

func newService() (*Service, error) {
	localMSPID, err := mspmgmt.GetLocalMSP().GetIdentifier()
	if err != nil {
		return nil, errors.Wrap(errors.SystemError, err, "error getting local MSP Identifier")
	}
	gossipService := service.GetGossipService()
	return newServiceWithOpts([]byte(localMSPID), gossipService, newMSPIDMgr(gossipService)), nil
}

// newServiceWithOpts returns a new Membership Service using the given options
func newServiceWithOpts(localMSPID []byte, gossipService service.GossipService, mspProvider mspIDProvider) *Service {

	service := &Service{
		localMSPID:    localMSPID,
		gossipService: gossipService,
		mspProvider:   mspProvider,
	}

	service.peers = lazyref.New(
		func() (interface{}, error) {
			return service.doGetAllPeers(), nil
		},
		lazyref.WithAbsoluteExpiration(cacheExpiration),
	)

	service.peersOfChannel = lazycache.New(
		"membership_cache",
		func(key lazycache.Key) (interface{}, error) {
			return service.doGetPeersOfChannel(key.String())
		},
		lazyref.WithAbsoluteExpiration(cacheExpiration),
	)

	return service
}

// GetAllPeers returns all peers on the gossip network
func (s *Service) GetAllPeers() []*memserviceapi.PeerEndpoint {
	peers, err := s.peers.Get()
	if err != nil {
		logger.Warnf("Received error while attempting to get all peers: %s. Returning empty list.", err)
		return nil
	}
	return peers.([]*memserviceapi.PeerEndpoint)
}

// GetLocalPeer returns all peers on the gossip network joined to the given channel
func (s *Service) GetLocalPeer(channelID string) (*memserviceapi.PeerEndpoint, error) {
	channelInfo := s.gossipService.SelfChannelInfo(common.ChainID(channelID))
	if channelInfo == nil {
		return nil, errors.Errorf(errors.SystemError, "local peer is not joined to channel [%s]", channelID)
	}
	localEndpoint := s.getLocalEndpoint()
	properties := channelInfo.GetStateInfo().Properties
	if properties != nil {
		localEndpoint.LedgerHeight = properties.LedgerHeight
		localEndpoint.Roles = properties.Roles
	}
	logger.Debugf("Returning local peer endpoint for channel [%s]: %+v", channelID, localEndpoint)
	return localEndpoint, nil
}

// GetPeersOfChannel returns all peers on the gossip network joined to the given channel
func (s *Service) GetPeersOfChannel(channelID string) ([]*memserviceapi.PeerEndpoint, error) {
	peersOfChannel, err := s.peersOfChannel.Get(lazycache.NewStringKey(channelID))
	if err != nil {
		return nil, err
	}
	return peersOfChannel.([]*memserviceapi.PeerEndpoint), nil
}

func (s *Service) doGetAllPeers() []*memserviceapi.PeerEndpoint {
	endpoints := s.getEndpoints("", s.gossipService.Peers())
	return append(endpoints, s.getLocalEndpoint())
}

func (s *Service) doGetPeersOfChannel(channelID string) ([]*memserviceapi.PeerEndpoint, error) {
	if channelID == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "channel ID must be provided")
	}

	endpoints := s.getEndpoints(channelID, s.gossipService.PeersOfChannel(common.ChainID(channelID)))
	channelInfo := s.gossipService.SelfChannelInfo(common.ChainID(channelID))
	if channelInfo != nil {
		localEndpoint := s.getLocalEndpoint()
		if channelInfo.GetStateInfo() != nil {
			properties := channelInfo.GetStateInfo().Properties
			if properties != nil {
				localEndpoint.LedgerHeight = properties.LedgerHeight
				localEndpoint.Roles = properties.Roles
			}
		}
		endpoints = append(endpoints, localEndpoint)
	}

	return endpoints, nil
}

func (s *Service) getLocalEndpoint() *memserviceapi.PeerEndpoint {
	return &memserviceapi.PeerEndpoint{
		Endpoint: s.gossipService.SelfMembershipInfo().PreferredEndpoint(),
		MSPid:    s.localMSPID,
	}
}

func (s *Service) getEndpoints(channelID string, members []discovery.NetworkMember) []*memserviceapi.PeerEndpoint {
	var peerEndpoints []*memserviceapi.PeerEndpoint

	for _, member := range members {
		ledgerHeight := uint64(0)
		leftChannel := false
		var roles []string

		properties := member.Properties
		if properties != nil {
			ledgerHeight = properties.LedgerHeight
			leftChannel = properties.LeftChannel
			roles = properties.Roles
		}

		if ledgerHeight == 0 {
			logger.Warnf("Ledger height for channel [%s] on peer [%s] is 0.\n", channelID, member.Endpoint)
		}

		peerEndpoint := &memserviceapi.PeerEndpoint{
			Endpoint:     member.PreferredEndpoint(),
			MSPid:        []byte(s.mspProvider.GetMSPID(member.PKIid)),
			LedgerHeight: ledgerHeight,
			LeftChannel:  leftChannel,
			Roles:        roles,
		}
		logger.Debugf("[%s] Adding peer [%s] - MSPID: [%s], LedgerHeight: %d, Roles: %s", channelID, peerEndpoint.Endpoint, peerEndpoint.MSPid, peerEndpoint.LedgerHeight, peerEndpoint.Roles)
		peerEndpoints = append(peerEndpoints, peerEndpoint)
	}
	return peerEndpoints
}
