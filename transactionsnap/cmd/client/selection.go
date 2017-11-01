/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"

	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/protos/common"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/pgresolver"
)

const (
	ccDataProviderSCC      = "lscc"
	ccDataProviderfunction = "getccdata"
)

type selectionServiceImpl struct {
	membershipManager api.MembershipManager
	mutex             sync.RWMutex
	pgResolvers       map[string]api.PeerGroupResolver
	pgLBP             api.LoadBalancePolicy
	ccDataProvider    api.CCDataProvider
	config            api.Config
}

// NewSelectionService creates a selection service
func NewSelectionService(config api.Config) api.SelectionService {
	return &selectionServiceImpl{
		membershipManager: GetMembershipInstance(config),
		pgResolvers:       make(map[string]api.PeerGroupResolver),
		pgLBP:             pgresolver.NewRandomLBP(),
		ccDataProvider:    newCCDataProvider(config),
		config:            config,
	}
}

func (s *selectionServiceImpl) GetEndorsersForChaincode(channelID string,
	chaincodeIDs ...string) ([]sdkApi.Peer, error) {

	if len(chaincodeIDs) == 0 {
		return nil, fmt.Errorf("no chaincode IDs provided")
	}

	resolver, err := s.getPeerGroupResolver(channelID, chaincodeIDs)
	if err != nil {
		return nil, fmt.Errorf("Error getting peer group resolver for chaincodes [%v] on channel [%s]: %s", chaincodeIDs, channelID, err)
	}
	return resolver.Resolve().Peers(), nil
}

func (s *selectionServiceImpl) GetPeerForEvents(channelID string) (*api.PeerConfig, error) {
	peerConfig := &api.PeerConfig{}
	channelMembership := s.membershipManager.GetPeersOfChannel(channelID, false)
	if channelMembership.QueryError != nil && len(channelMembership.Peers) == 0 {
		// Query error and there is no cached membership list
		return peerConfig, channelMembership.QueryError
	}

	rs := rand.NewSource(time.Now().Unix())
	r := rand.New(rs)
	randomPeer := r.Intn(len(channelMembership.Peers))

	// Membership Service does not know the event port. We assume it is the same
	// as the local peer
	localPeer, err := s.config.GetLocalPeer()
	if err != nil {
		return peerConfig, err
	}
	selectedPeer := channelMembership.Peers[randomPeer]
	host, _, err := net.SplitHostPort(selectedPeer.URL())
	if err != nil {
		return peerConfig, err
	}

	peerConfig = &api.PeerConfig{
		EventHost: host,
		EventPort: localPeer.EventPort,
		MSPid:     []byte(selectedPeer.MSPID()),
	}

	return peerConfig, nil
}

func (s *selectionServiceImpl) getPeerGroupResolver(channelID string, chaincodeIDs []string) (api.PeerGroupResolver, error) {
	key := newResolverKey(channelID, chaincodeIDs...)

	s.mutex.RLock()
	resolver := s.pgResolvers[key.String()]
	s.mutex.RUnlock()

	if resolver == nil {
		var err error
		if resolver, err = s.createPGResolver(key); err != nil {
			return nil, fmt.Errorf("unable to create new peer group resolver for chaincode(s) [%v] on channel [%s]: %s", chaincodeIDs, channelID, err)
		}
	}
	return resolver, nil
}

func (s *selectionServiceImpl) createPGResolver(key *resolverKey) (api.PeerGroupResolver, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	resolver := s.pgResolvers[key.String()]
	if resolver != nil {
		// Already cached
		return resolver, nil
	}

	// Retrieve the signature policies for all of the chaincodes
	var policyGroups []api.Group
	for _, ccID := range key.chaincodeIDs {
		policyGroup, err := s.getPolicyGroupForCC(key.channelID, ccID)
		if err != nil {
			return nil, fmt.Errorf("error retrieving signature policy for chaincode [%s] on channel [%s]: %s", ccID, key.channelID, err)
		}
		policyGroups = append(policyGroups, policyGroup)
	}

	// Perform an 'and' operation on all of the peer groups
	aggregatePolicyGroup, err := pgresolver.NewGroupOfGroups(policyGroups).Nof(int32(len(policyGroups)))
	if err != nil {
		return nil, fmt.Errorf("error computing signature policy for chaincode(s) [%v] on channel [%s]: %s", key.chaincodeIDs, key.channelID, err)
	}

	// Create the resolver
	if resolver, err = pgresolver.NewPeerGroupResolver(aggregatePolicyGroup, s.pgLBP); err != nil {
		return nil, fmt.Errorf("error creating peer group resolver for chaincodes [%v] on channel [%s]: %s", key.chaincodeIDs, key.channelID, err)
	}

	s.pgResolvers[key.String()] = resolver

	return resolver, nil
}

func (s *selectionServiceImpl) getPolicyGroupForCC(channelID, ccID string) (api.Group, error) {
	ccData, err := s.ccDataProvider.QueryChaincodeData(channelID, ccID)
	if err != nil {
		return nil, fmt.Errorf("error querying chaincode [%s] on channel [%s]: %s", ccID, channelID, err)
	}

	sigPolicyEnv := &common.SignaturePolicyEnvelope{}
	if err := proto.Unmarshal(ccData.Policy, sigPolicyEnv); err != nil {
		return nil, fmt.Errorf("error unmarshalling SignaturePolicyEnvelope for chaincode [%s] on channel [%s]: %v", ccID, channelID, err)
	}

	return pgresolver.NewSignaturePolicyCompiler(
		func(mspID string) []sdkApi.Peer {
			return s.getAvailablePeers(channelID, mspID)
		}).Compile(sigPolicyEnv)
}

func (s *selectionServiceImpl) getAvailablePeers(channelID string, mspID string) []sdkApi.Peer {
	var peers []sdkApi.Peer
	channelMembership := s.membershipManager.GetPeersOfChannel(channelID, true)
	if channelMembership.QueryError != nil && len(channelMembership.Peers) == 0 {
		// Query error and there is no cached membership list
		logger.Errorf("unable to get membership for channel [%s]: %s", channelID, channelMembership.QueryError)
		return nil
	}
	for _, peer := range channelMembership.Peers {
		if string(peer.MSPID()) == mspID {
			peers = append(peers, peer)
		}
	}

	if logger.IsEnabledFor(logging.DEBUG) {
		str := ""
		for i, peer := range peers {
			str += peer.URL()
			if i+1 < len(peers) {
				str += ","
			}
		}
		logger.Debugf("Available peers:\n%s\n", str)
	}

	return peers
}

type ccDataProviderImpl struct {
	ccDataMap map[string]*ccprovider.ChaincodeData
	mutex     sync.RWMutex
	config    api.Config
}

func newCCDataProvider(config api.Config) api.CCDataProvider {
	return &ccDataProviderImpl{ccDataMap: make(map[string]*ccprovider.ChaincodeData), config: config}
}

func (p *ccDataProviderImpl) QueryChaincodeData(channelID string, chaincodeID string) (*ccprovider.ChaincodeData, error) {
	key := newResolverKey(channelID, chaincodeID)
	var ccData *ccprovider.ChaincodeData

	p.mutex.RLock()
	ccData = p.ccDataMap[key.String()]
	p.mutex.RUnlock()

	if ccData != nil {
		return ccData, nil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	response, err := queryChaincode(channelID, ccDataProviderSCC, []string{ccDataProviderfunction, channelID, chaincodeID}, p.config)
	if err != nil {
		return nil, fmt.Errorf("error querying chaincode data for chaincode [%s] on channel [%s]: %s", chaincodeID, channelID, err)
	}

	ccData = &ccprovider.ChaincodeData{}
	err = proto.Unmarshal(response.ProposalResponse.Response.Payload, ccData)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling chaincode data: %v", err)
	}

	p.ccDataMap[key.String()] = ccData

	return ccData, nil
}

func (p *ccDataProviderImpl) clearCache() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.ccDataMap = make(map[string]*ccprovider.ChaincodeData)
}

type resolverKey struct {
	channelID    string
	chaincodeIDs []string
	key          string
}

func (k *resolverKey) String() string {
	return k.key
}

func newResolverKey(channelID string, chaincodeIDs ...string) *resolverKey {
	arr := chaincodeIDs[:]
	sort.Strings(arr)

	key := channelID + "-"
	for i, s := range arr {
		key += s
		if i+1 < len(arr) {
			key += ":"
		}
	}
	return &resolverKey{channelID: channelID, chaincodeIDs: arr, key: key}
}
