/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package client

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	mocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/protos/common"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/client/pgresolver"
	config "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/config"
)

var configImp = mocks.NewMockConfig()

const (
	org1  = "Org1MSP"
	org2  = "Org2MSP"
	org3  = "Org3MSP"
	org4  = "Org4MSP"
	org5  = "Org5MSP"
	org6  = "Org6MSP"
	org7  = "Org7MSP"
	org8  = "Org8MSP"
	org9  = "Org9MSP"
	org10 = "Org10MSP"
)

const (
	channel1 = "channel1"
	channel2 = "channel2"
)

const (
	cc1 = "cc1"
	cc2 = "cc2"
	cc3 = "cc3"
)

const (
	o1 = iota
	o2
	o3
	o4
	o5
)

var p1 = peer("peer1")
var p2 = peer("peer2")
var p3 = peer("peer3")
var p4 = peer("peer4")
var p5 = peer("peer5")
var p6 = peer("peer6")
var p7 = peer("peer7")
var p8 = peer("peer8")
var p9 = peer("peer9")
var p10 = peer("peer10")
var p11 = peer("peer11")
var p12 = peer("peer12")

var pc1 = peerConfig(p1, org1)
var pc2 = peerConfig(p2, org1)
var pc3 = peerConfig(p3, org2)
var pc4 = peerConfig(p4, org2)
var pc5 = peerConfig(p5, org3)
var pc6 = peerConfig(p6, org3)
var pc7 = peerConfig(p7, org3)
var pc8 = peerConfig(p8, org4)
var pc9 = peerConfig(p9, org4)
var pc10 = peerConfig(p10, org4)
var pc11 = peerConfig(p11, org5)
var pc12 = peerConfig(p12, org5)

func TestMain(m *testing.M) {
	err := config.Init("")
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}
	fabricClient, err := GetInstance()
	if err != nil {

		panic(fmt.Sprintf("Client GetInstance return error %v", err))
	}
	fabricClient.GetConfig().FabricClientViper().Set("client.tls.enabled", false)

	os.Exit(m.Run())
}

func TestGetEndorsersForChaincodeOneCC(t *testing.T) {
	service := newMockSelectionService(
		newMockMembershipManager().
			add(channel1, pc1, pc2, pc3, pc4, pc5, pc6, pc7, pc8),
		newMockCCDataProvider().
			add(channel1, cc1, getPolicy1()),
		pgresolver.NewRoundRobinLBP())

	// Channel1(Policy(cc1)) = Org1
	expected := []pgresolver.PeerGroup{
		// Org1
		pg(p1), pg(p2),
	}
	verify(t, service, expected, channel1, cc1)
}

func TestGetEndorsersForChaincodeTwoCCs(t *testing.T) {
	service := newMockSelectionService(
		newMockMembershipManager().
			add(channel1, pc1, pc2, pc3, pc4, pc5, pc6, pc7, pc8),
		newMockCCDataProvider().
			add(channel1, cc1, getPolicy1()).
			add(channel1, cc2, getPolicy2()),
		pgresolver.NewRoundRobinLBP())

	// Channel1(Policy(cc1) and Policy(cc2)) = Org1 and (1 of [(2 of [Org1,Org2]),(2 of [Org1,Org3,Org4])])
	expected := []pgresolver.PeerGroup{
		// Org1 and Org2
		pg(p1, p3), pg(p1, p4), pg(p2, p3), pg(p2, p4),
		// Org1 and Org3
		pg(p1, p5), pg(p1, p6), pg(p1, p7), pg(p2, p5), pg(p2, p6), pg(p2, p7),
		// Org1 and Org4
		pg(p1, p8), pg(p1, p9), pg(p1, p10), pg(p2, p8), pg(p2, p9), pg(p2, p10),
		// Org1 and Org3 and Org4
		pg(p1, p5, p8), pg(p1, p5, p9), pg(p1, p5, p10), pg(p1, p6, p8), pg(p1, p6, p9), pg(p1, p6, p10), pg(p1, p7, p8), pg(p1, p7, p9), pg(p1, p7, p10),
		pg(p2, p5, p8), pg(p2, p5, p9), pg(p2, p5, p10), pg(p2, p6, p8), pg(p2, p6, p9), pg(p2, p6, p10), pg(p2, p7, p8), pg(p2, p7, p9), pg(p2, p7, p10),
	}
	verify(t, service, expected, channel1, cc1, cc2)
}

func TestGetEndorsersForChaincodeTwoCCsTwoChannels(t *testing.T) {
	service := newMockSelectionService(
		newMockMembershipManager().
			add(channel1, pc1, pc2, pc3, pc4, pc5, pc6, pc7, pc8).
			add(channel2, pc1, pc2, pc3, pc4, pc5, pc6, pc7, pc8, pc9, pc10, pc11, pc12),
		newMockCCDataProvider().
			add(channel1, cc1, getPolicy1()).
			add(channel1, cc2, getPolicy2()).
			add(channel2, cc1, getPolicy3()).
			add(channel2, cc2, getPolicy2()),
		pgresolver.NewRoundRobinLBP(),
	)

	// Channel1(Policy(cc1) and Policy(cc2)) = Org1 and (1 of [(2 of [Org1,Org2]),(2 of [Org1,Org3,Org4])])
	expected := []pgresolver.PeerGroup{
		// Org1 and Org2
		pg(p1, p3), pg(p1, p4), pg(p2, p3), pg(p2, p4),
		// Org1 and Org3
		pg(p1, p5), pg(p1, p6), pg(p1, p7), pg(p2, p5), pg(p2, p6), pg(p2, p7),
		// Org1 and Org4
		pg(p1, p8), pg(p1, p9), pg(p1, p10), pg(p2, p8), pg(p2, p9), pg(p2, p10),
		// Org1 and Org3 and Org4
		pg(p1, p5, p8), pg(p1, p5, p9), pg(p1, p5, p10), pg(p1, p6, p8), pg(p1, p6, p9), pg(p1, p6, p10), pg(p1, p7, p8), pg(p1, p7, p9), pg(p1, p7, p10),
		pg(p2, p5, p8), pg(p2, p5, p9), pg(p2, p5, p10), pg(p2, p6, p8), pg(p2, p6, p9), pg(p2, p6, p10), pg(p2, p7, p8), pg(p2, p7, p9), pg(p2, p7, p10),
	}
	verify(t, service, expected, channel1, cc1, cc2)

	// Channel2(Policy(cc1) and Policy(cc2)) = Org5 and (1 of [(2 of [Org1,Org2]),(2 of [Org1,Org3,Org4])])
	expected = []pgresolver.PeerGroup{
		// Org5 and Org2
		pg(p11, p1, p3), pg(p11, p1, p4), pg(p11, p2, p3), pg(p11, p2, p4),
		pg(p12, p1, p3), pg(p12, p1, p4), pg(p12, p2, p3), pg(p12, p2, p4),
		// Org5 and Org3
		pg(p11, p1, p5), pg(p11, p1, p6), pg(p11, p1, p7), pg(p11, p2, p5), pg(p11, p2, p6), pg(p11, p2, p7),
		pg(p12, p1, p5), pg(p12, p1, p6), pg(p12, p1, p7), pg(p12, p2, p5), pg(p12, p2, p6), pg(p12, p2, p7),
		// Org5 and Org4
		pg(p11, p1, p8), pg(p11, p1, p9), pg(p11, p1, p10), pg(p11, p2, p8), pg(p11, p2, p9), pg(p11, p2, p10),
		pg(p12, p1, p8), pg(p12, p1, p9), pg(p12, p1, p10), pg(p12, p2, p8), pg(p12, p2, p9), pg(p12, p2, p10),
		// Org5 and Org3 and Org4
		pg(p11, p5, p8), pg(p11, p5, p9), pg(p11, p5, p10), pg(p11, p6, p8), pg(p11, p6, p9), pg(p11, p6, p10), pg(p11, p7, p8), pg(p11, p7, p9), pg(p11, p7, p10),
		pg(p12, p5, p8), pg(p12, p5, p9), pg(p12, p5, p10), pg(p12, p6, p8), pg(p12, p6, p9), pg(p12, p6, p10), pg(p12, p7, p8), pg(p12, p7, p9), pg(p12, p7, p10),
	}
	verify(t, service, expected, channel2, cc1, cc2)
}

func verify(t *testing.T, service SelectionService, expectedPeerGroups []pgresolver.PeerGroup, channelID string, chaincodeIDs ...string) {
	// Set the log level to WARNING since the following spits out too much info in DEBUG
	module := "pg-resolver"
	level := logging.GetLevel(module)
	logging.SetLevel(logging.WARNING, module)
	defer logging.SetLevel(level, module)

	for i := 0; i < len(expectedPeerGroups); i++ {
		peers, err := service.GetEndorsersForChaincode(channelID, chaincodeIDs...)
		if err != nil {
			t.Fatalf("error getting endorsers: %s", err)
		}
		if !containsPeerGroup(expectedPeerGroups, peers) {
			t.Fatalf("peer group %s is not one of the expected peer groups: %v", toString(peers), expectedPeerGroups)
		}
	}

}

func containsPeerGroup(groups []pgresolver.PeerGroup, peers []sdkApi.Peer) bool {
	for _, g := range groups {
		if containsAllPeers(peers, g) {
			return true
		}
	}
	return false
}

func containsAllPeers(peers []sdkApi.Peer, pg pgresolver.PeerGroup) bool {
	if len(peers) != len(pg.Peers()) {
		return false
	}
	for _, peer := range peers {
		if !containsPeer(pg.Peers(), peer) {
			return false
		}
	}
	return true
}

func containsPeer(peers []sdkApi.Peer, peer sdkApi.Peer) bool {
	for _, p := range peers {
		if p.URL() == peer.URL() {
			return true
		}
	}
	return false
}

func pg(peers ...sdkApi.Peer) pgresolver.PeerGroup {
	return pgresolver.NewPeerGroup(peers...)
}

func peer(name string) sdkApi.Peer {
	peer, err := sdkFabApi.NewPeer(name+":7051", "", "", configImp)
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}
	peer.SetName(name)
	return peer
}

func newMockSelectionService(membershipManager MembershipManager, ccDataProvider CCDataProvider,
	lbp pgresolver.LoadBalancePolicy) SelectionService {
	return &selectionServiceImpl{
		membershipManager: membershipManager,
		ccDataProvider:    ccDataProvider,
		pgLBP:             lbp,
		pgResolvers:       make(map[string]pgresolver.PeerGroupResolver),
	}
}

type mockMembershipManager struct {
	peerConfigs map[string]config.PeerConfigs
}

func (m *mockMembershipManager) GetPeersOfChannel(channelID string, poll bool) ChannelMembership {
	return ChannelMembership{Peers: m.peerConfigs[channelID], PollingEnabled: poll}
}

func newMockMembershipManager() *mockMembershipManager {
	return &mockMembershipManager{peerConfigs: make(map[string]config.PeerConfigs)}
}

func (m *mockMembershipManager) add(channelID string, peerConfigs ...config.PeerConfig) *mockMembershipManager {
	m.peerConfigs[channelID] = peerConfigs
	return m
}

type mockCCDataProvider struct {
	ccData map[string]*ccprovider.ChaincodeData
}

func newMockCCDataProvider() *mockCCDataProvider {
	return &mockCCDataProvider{ccData: make(map[string]*ccprovider.ChaincodeData)}
}

func (p *mockCCDataProvider) QueryChaincodeData(channelID string, chaincodeID string) (*ccprovider.ChaincodeData, error) {
	return p.ccData[newResolverKey(channelID, chaincodeID).String()], nil
}

func (p *mockCCDataProvider) add(channelID string, chaincodeID string, policy *ccprovider.ChaincodeData) *mockCCDataProvider {
	p.ccData[newResolverKey(channelID, chaincodeID).String()] = policy
	return p
}

// Policy: Org1
func getPolicy1() *ccprovider.ChaincodeData {
	signedBy, identities, err := pgresolver.GetPolicies(org1)
	if err != nil {
		panic(err)
	}

	return newCCData(&common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       signedBy[o1],
		Identities: identities,
	})
}

// Policy: 1 of [(2 of [Org1, Org2]),(2 of [Org1, Org3, Org4])]
func getPolicy2() *ccprovider.ChaincodeData {
	signedBy, identities, err := pgresolver.GetPolicies(org1, org2, org3, org4)
	if err != nil {
		panic(err)
	}

	return newCCData(&common.SignaturePolicyEnvelope{
		Version: 0,
		Rule: pgresolver.NewNOutOfPolicy(1,
			pgresolver.NewNOutOfPolicy(2,
				signedBy[o1],
				signedBy[o2],
			),
			pgresolver.NewNOutOfPolicy(2,
				signedBy[o1],
				signedBy[o3],
				signedBy[o4],
			),
		),
		Identities: identities,
	})
}

// Policy: Org5
func getPolicy3() *ccprovider.ChaincodeData {
	signedBy, identities, err := pgresolver.GetPolicies(org1, org2, org3, org4, org5)
	if err != nil {
		panic(err)
	}

	return newCCData(&common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       signedBy[o5],
		Identities: identities,
	})
}

func newCCData(sigPolicyEnv *common.SignaturePolicyEnvelope) *ccprovider.ChaincodeData {
	policyBytes, err := proto.Marshal(sigPolicyEnv)
	if err != nil {
		panic(err)
	}

	return &ccprovider.ChaincodeData{Policy: policyBytes}
}

func peerConfig(peer sdkApi.Peer, mspID string) config.PeerConfig {
	s := strings.Split(peer.URL(), ":")
	port, err := strconv.Atoi(s[1])
	if err != nil {
		panic(err)
	}
	return config.PeerConfig{Host: s[0], Port: port, MSPid: []byte(mspID)}
}

func toString(peers []sdkApi.Peer) string {
	str := "["
	for i, p := range peers {
		str += p.URL()
		if i+1 < len(peers) {
			str += ","
		}
	}
	str += "]"
	return str
}
