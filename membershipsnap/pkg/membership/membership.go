/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"fmt"
	"time"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/service"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/util/bcinfo"
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

// channelsInfoProvider provides info about all channels that
// the peer is joined to
type channelsInfoProvider interface {
	GetChannelsInfo() []*pb.ChannelInfo
}

// blockchainInfoProvider provides block chain info for a given channel
type blockchainInfoProvider interface {
	GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error)
}

// Service provides functions to query peers
type Service struct {
	gossipService    service.GossipService
	mspProvider      mspIDProvider
	chInfoProvider   channelsInfoProvider
	bciProvider      blockchainInfoProvider
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

	peerEndpoint, err := peer.GetPeerEndpoint()
	if err != nil {
		return nil, errors.Wrap(errors.SystemError, err, "error reading peer endpoint")
	}

	gossipService := service.GetGossipService()
	return newServiceWithOpts(peerEndpoint.Address, []byte(localMSPID), gossipService, newMSPIDMgr(gossipService), &peerChInfoProvider{}, bcinfo.NewProvider()), nil
}

// newServiceWithOpts returns a new Membership Service using the given options
func newServiceWithOpts(localPeerAddress string, localMSPID []byte, gossipService service.GossipService,
	mspProvider mspIDProvider, chInfoProvider channelsInfoProvider, bciProvider blockchainInfoProvider) *Service {

	service := &Service{
		localPeerAddress: localPeerAddress,
		localMSPID:       localMSPID,
		gossipService:    gossipService,
		mspProvider:      mspProvider,
		chInfoProvider:   chInfoProvider,
		bciProvider:      bciProvider,
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

// GetPeersOfChannel returns all peers on the gossip network joined to the given channel
func (s *Service) GetPeersOfChannel(channelID string) ([]*memserviceapi.PeerEndpoint, error) {
	peersOfChannel, err := s.peersOfChannel.Get(lazycache.NewStringKey(channelID))
	if err != nil {
		return nil, err
	}
	return peersOfChannel.([]*memserviceapi.PeerEndpoint), nil
}

func (s *Service) doGetAllPeers() []*memserviceapi.PeerEndpoint {
	return s.getEndpoints("", s.gossipService.Peers(), true)
}

func (s *Service) doGetPeersOfChannel(channelID string) ([]*memserviceapi.PeerEndpoint, error) {
	if channelID == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "channel ID must be provided")
	}
	localPeerJoined := false
	for _, ch := range s.chInfoProvider.GetChannelsInfo() {
		if ch.ChannelId == channelID {
			localPeerJoined = true
			break
		}
	}
	return s.getEndpoints(channelID, s.gossipService.PeersOfChannel(common.ChainID(channelID)), localPeerJoined), nil
}

func (s *Service) getEndpoints(channelID string, members []discovery.NetworkMember, includeLocalPeer bool) []*memserviceapi.PeerEndpoint {
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

	if includeLocalPeer {
		// Add self since Gossip only contains other peers
		var ledgerHeight uint64
		if channelID != "" {
			bcInfo, err := s.bciProvider.GetBlockchainInfo(channelID)
			if err != nil {
				logger.Errorf(errors.WithMessage(errors.SystemError, err, fmt.Sprintf("Error getting ledger height for channel [%s] on local peer. Ledger height will be set to 0.\n", channelID)).GenerateLogMsg())
			} else {
				ledgerHeight = bcInfo.Height
			}
		}

		self := &memserviceapi.PeerEndpoint{
			Endpoint:     s.localPeerAddress,
			MSPid:        s.localMSPID,
			LedgerHeight: ledgerHeight,
			LeftChannel:  false,
			Roles:        ledgerconfig.RolesAsString(),
		}

		peerEndpoints = append(peerEndpoints, self)
		logger.Debugf("[%s] Adding self [%s] - MSPID: [%s], LedgerHeight: %d, Roles: %s", self.Endpoint, self.MSPid, self.LedgerHeight, self.Roles)
	}

	return peerEndpoints
}

type peerChInfoProvider struct {
}

// GetChannelsInfo delegates to the peer to return an array with
// information about all channels for this peer
func (p *peerChInfoProvider) GetChannelsInfo() []*pb.ChannelInfo {
	return peer.GetChannelsInfo()
}
