/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"sync/atomic"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric/core/peer"
	"github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/service"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("membershipsnap")

var initialized uint32
var membershipService memserviceapi.Service

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
}

// Get returns the Membership Service instance.
// If the service hasn't been initialized yet then
// it will be initialized.
func Get() (memserviceapi.Service, error) {
	if atomic.LoadUint32(&initialized) == 1 {
		return membershipService, nil
	}

	memService, err := newService()
	if err != nil {
		logger.Errorf("error initializing membership service: %s\n", err)
		return nil, errors.Wrap(errors.GeneralError, err, "error initializing membership service")
	}

	if atomic.CompareAndSwapUint32(&initialized, 0, 1) {
		membershipService = memService
		logger.Infof("... successfully initialized membership service\n")
	}

	return membershipService, nil
}

func newService() (*Service, error) {
	localMSPID, err := mspmgmt.GetLocalMSP().GetIdentifier()
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "error getting local MSP Identifier")
	}

	peerEndpoint, err := peer.GetPeerEndpoint()
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "error reading peer endpoint")
	}

	gossipService := service.GetGossipService()
	return newServiceWithOpts(peerEndpoint.Address, []byte(localMSPID), gossipService, newMSPIDMgr(gossipService), &peerChInfoProvider{}, &peerBCInfoProvider{}), nil
}

// newServiceWithOpts returns a new Membership Service using the given options
func newServiceWithOpts(localPeerAddress string, localMSPID []byte, gossipService service.GossipService,
	mspProvider mspIDProvider, chInfoProvider channelsInfoProvider, bciProvider blockchainInfoProvider) *Service {
	return &Service{
		localPeerAddress: localPeerAddress,
		localMSPID:       localMSPID,
		gossipService:    gossipService,
		mspProvider:      mspProvider,
		chInfoProvider:   chInfoProvider,
		bciProvider:      bciProvider,
	}
}

// GetAllPeers returns all peers on the gossip network
func (s *Service) GetAllPeers() []*memserviceapi.PeerEndpoint {
	return s.getEndpoints("", s.gossipService.Peers(), true)
}

// GetPeersOfChannel returns all peers on the gossip network joined to the given channel
func (s *Service) GetPeersOfChannel(channelID string) ([]*memserviceapi.PeerEndpoint, error) {
	if channelID == "" {
		return nil, errors.New(errors.GeneralError, "channel ID must be provided")
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

		properties := member.Properties
		if properties != nil {
			ledgerHeight = properties.LedgerHeight
			leftChannel = properties.LeftChannel
		}

		if ledgerHeight == 0 {
			logger.Warnf("Ledger height for channel [%s] on peer [%s] is 0.\n", channelID, member.Endpoint)
		}

		peerEndpoints = append(peerEndpoints, &memserviceapi.PeerEndpoint{
			Endpoint:         member.Endpoint,
			InternalEndpoint: member.InternalEndpoint,
			MSPid:            []byte(s.mspProvider.GetMSPID(member.PKIid)),
			LedgerHeight:     ledgerHeight,
			LeftChannel:      leftChannel,
		})
	}

	if includeLocalPeer {
		// Add self since Gossip only contains other peers
		var ledgerHeight uint64
		if channelID != "" {
			bcInfo, err := s.bciProvider.GetBlockchainInfo(channelID)
			if err != nil {
				logger.Errorf("Error getting ledger height for channel [%s] on local peer. Ledger height will be set to 0.\n", channelID)
			} else {
				// Need to subtract 1 from the block height since the LedgerHeight in the
				// Gossip NetworkMember is really the block number (i.e. they subtract 1 also)
				// So, we need to make it match.
				ledgerHeight = bcInfo.Height - 1
			}
		}

		self := &memserviceapi.PeerEndpoint{
			Endpoint:         s.localPeerAddress,
			InternalEndpoint: s.localPeerAddress,
			MSPid:            s.localMSPID,
			LedgerHeight:     ledgerHeight,
			LeftChannel:      false,
		}

		peerEndpoints = append(peerEndpoints, self)
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

type peerBCInfoProvider struct {
	bcInfo map[string]*cb.BlockchainInfo
}

// GetBlockchainInfo delegates to the peer to return basic info about the blockchain
func (l *peerBCInfoProvider) GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error) {
	return peer.GetLedger(channelID).GetBlockchainInfo()
}
