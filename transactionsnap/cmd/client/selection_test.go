/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apilogging"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	mocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/common/ccprovider"
	"github.com/hyperledger/fabric/protos/common"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/pgresolver"
	config "github.com/securekey/fabric-snaps/transactionsnap/cmd/config"
)

var configImp = mocks.NewMockConfig()
var channelID = "testChannel"
var mspID = "Org1MSP"

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

var p1 = peer("peer1", org1)
var p2 = peer("peer2", org1)
var p3 = peer("peer3", org2)
var p4 = peer("peer4", org2)
var p5 = peer("peer5", org3)
var p6 = peer("peer6", org3)
var p7 = peer("peer7", org3)
var p8 = peer("peer8", org4)
var p9 = peer("peer9", org4)
var p10 = peer("peer10", org4)
var p11 = peer("peer11", org5)
var p12 = peer("peer12", org5)

type sampleConfig struct {
	api.Config
}

// Override GetMspConfigPath for relative path, just to avoid using new core.yaml for this purpose
func (c *sampleConfig) GetMspConfigPath() string {
	return "../sampleconfig/msp"
}

func TestMain(m *testing.M) {

	opts := &bccspFactory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &bccspFactory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &bccspFactory.FileKeystoreOpts{KeyStorePath: "../sampleconfig/msp/keystore/"},
		},
	}
	bccspFactory.InitFactories(opts)

	//to upload config to HL for testChannel
	uploadConfigPerChannelToHL(channelID)

	os.Exit(m.Run())
}

func TestCachedItem(t *testing.T) {
	isCached = false

	chnlConfig, err := getConfig()
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	//get instance of client
	_, err = GetInstance(channelID, &sampleConfig{chnlConfig})
	if err != nil {
		t.Fatalf("Error getting instance of config: %v", err)
	}
	if isCached {
		t.Fatalf("This item should have been cached")
	}
	if cache != nil {
		cache.Delete(channelID)
	}

}

func TestGetInstanceWithNilConfig(t *testing.T) {
	_, err := GetInstance(channelID, nil)
	if err == nil {
		t.Fatalf("Expected error: 'Config is nil. Cannot create instance of client'")
	}

	_, err = GetInstance("", nil)
	if err == nil {
		t.Fatalf("Expected error: 'Channel name is mandatory'")
	}

}

func TestGettingCachedOnTwoChannels(t *testing.T) {

	chnlConfig, err := getConfig()
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	//get instance of client
	_, err = GetInstance(channelID, &sampleConfig{chnlConfig})
	if err != nil {
		t.Fatalf("Error getting instance of config: %v", err)
	}
	if isCached {
		t.Fatalf("This item should NOT have been cached")
	}

	//initialize config for channel 2
	chnlName := "chl2"
	uploadConfigPerChannelToHL(chnlName)
	//get config instance for testChannel
	cfgClient, err := GetInstance(channelID, &sampleConfig{chnlConfig})
	if err != nil {
		t.Fatalf("Error getting instance of config: %v", err)
	}
	//get config instance for ch2
	cfgClientCh2, err := GetInstance(chnlName, &sampleConfig{chnlConfig})
	if err != nil {
		t.Fatalf("Error getting instance of config: %v", err)
	}
	if cfgClient == nil {
		t.Fatalf("Error getting config from cache for channelID %s: %v", channelID, err)
	}

	if cfgClientCh2 == nil {
		t.Fatalf("Error getting config from cache for channelID %s: %v", chnlName, err)
	}
	//get directly from cache and check expiration
	x, expiration, found := cache.GetWithExpiration(chnlName)
	if !found {
		t.Error("Cached item for chnlName was not found")
	}
	if x == nil {
		t.Error("Config should not be nil")
	}
	if expiration.IsZero() {
		t.Errorf("expiration should be greater then zero")
	}
	if expiration.UnixNano() < time.Now().UnixNano() {
		t.Error("expiration for config is in the past")
	}
	//remove from cache
	if cache != nil {
		cache.Delete(channelID)
		cache.Delete(chnlName)
	}
	//check if item was removed
	x, expiration, found = cache.GetWithExpiration(chnlName)
	if found {
		t.Error("Cached item should not exist. It was deleted")
	}
	if x != nil {
		t.Error("Config should be nil. Item was removed from cache")
	}
	if !expiration.IsZero() {
		t.Errorf("expiration should be  zero")
	}

}

func uploadConfigPerChannelToHL(chnlID string) {
	configData, err := ioutil.ReadFile("../sampleconfig/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("File error: %v\n", err))
	}
	configMsg := &configmanagerApi.ConfigMessage{MspID: mspID,
		Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{
			PeerID: "jdoe", App: []configmanagerApi.AppConfig{
				configmanagerApi.AppConfig{AppName: "txnsnap", Config: string(configData)}}}}}
	configBytes, err := json.Marshal(configMsg)
	if err != nil {
		panic(fmt.Sprintf("Cannot Marshal %s\n", err))
	}
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = chnlID
	//upload valid message to HL
	err = uplaodConfigToHL(stub, configBytes)
	if err != nil {
		panic(fmt.Sprintf("Cannot upload %s\n", err))
	}
	configmgmtService.Initialize(stub, mspID)

}

func TestChangeCacheConfig(t *testing.T) {
	//configureTestCache(1*time.Millisecond, 1*time.Millisecond)

	chnlConfig, err := getConfig()
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	//get instance of client
	_, err = GetInstance(channelID, &sampleConfig{chnlConfig})
	if err != nil {
		t.Fatalf("Error getting instance of config: %v", err)
	}
	if isCached {
		t.Fatalf("This item should have been cached")
	}
	time.Sleep(2 * time.Second)
	_, err = GetInstance(channelID, &sampleConfig{chnlConfig})
	if err != nil {
		t.Fatalf("Error getting instance of config: %v", err)
	}
	//due to expiry time of 1ms this item should not come from cache
	if isCached {
		t.Fatalf("This item should have been cached")
	}
	if cache != nil {
		cache.Delete(channelID)
		cache.DeleteExpired()
	}

}

func TestGetEndorsersForChaincodeOneCC(t *testing.T) {
	service := newMockSelectionService(
		newMockMembershipManager().
			add(channel1, p1, p2, p3, p4, p5, p6, p7, p8),
		newMockCCDataProvider().
			add(channel1, cc1, getPolicy1()),
		pgresolver.NewRoundRobinLBP())

	// Channel1(Policy(cc1)) = Org1
	expected := []api.PeerGroup{
		// Org1
		pg(p1), pg(p2),
	}
	verify(t, service, expected, channel1, cc1)
}

func TestGetEndorsersForChaincodeTwoCCs(t *testing.T) {
	service := newMockSelectionService(
		newMockMembershipManager().
			add(channel1, p1, p2, p3, p4, p5, p6, p7, p8),
		newMockCCDataProvider().
			add(channel1, cc1, getPolicy1()).
			add(channel1, cc2, getPolicy2()),
		pgresolver.NewRoundRobinLBP())

	// Channel1(Policy(cc1) and Policy(cc2)) = Org1 and (1 of [(2 of [Org1,Org2]),(2 of [Org1,Org3,Org4])])
	expected := []api.PeerGroup{
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
			add(channel1, p1, p2, p3, p4, p5, p6, p7, p8).
			add(channel2, p1, p2, p3, p4, p5, p6, p7, p8, p9, p10, p11, p12),
		newMockCCDataProvider().
			add(channel1, cc1, getPolicy1()).
			add(channel1, cc2, getPolicy2()).
			add(channel2, cc1, getPolicy3()).
			add(channel2, cc2, getPolicy2()),
		pgresolver.NewRoundRobinLBP(),
	)

	// Channel1(Policy(cc1) and Policy(cc2)) = Org1 and (1 of [(2 of [Org1,Org2]),(2 of [Org1,Org3,Org4])])
	expected := []api.PeerGroup{
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
	expected = []api.PeerGroup{
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

func verify(t *testing.T, service api.SelectionService, expectedPeerGroups []api.PeerGroup, channelID string, chaincodeIDs ...string) {
	// Set the log level to WARNING since the following spits out too much info in DEBUG
	module := "pg-resolver"
	level := logging.GetLevel(module)
	logging.SetLevel(module, apilogging.WARNING)
	defer logging.SetLevel(module, level)

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

func getConfig() (transactionsnapApi.Config, error) {
	chnlConfig, err := config.NewConfig("../sampleconfig", channelID)
	if err != nil {
		return nil, err
	}
	return chnlConfig, nil
}
func getMockStub() *shim.MockStub {
	stub := shim.NewMockStub("testConfigState", nil)
	stub.MockTransactionStart("saveConfiguration")
	stub.ChannelID = channelID
	return stub
}

//uplaodConfigToHL to upload key&config to repository
func uplaodConfigToHL(stub *shim.MockStub, config []byte) error {
	configManager := mgmt.NewConfigManager(stub)
	if configManager == nil {
		return fmt.Errorf("Cannot instantiate config manager")
	}
	err := configManager.Save(config)
	return err

}

func containsPeerGroup(groups []api.PeerGroup, peers []apifabclient.Peer) bool {
	for _, g := range groups {
		if containsAllPeers(peers, g) {
			return true
		}
	}
	return false
}

func containsAllPeers(peers []apifabclient.Peer, pg api.PeerGroup) bool {
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

func containsPeer(peers []apifabclient.Peer, peer apifabclient.Peer) bool {
	for _, p := range peers {
		if p.URL() == peer.URL() {
			return true
		}
	}
	return false
}

func pg(peers ...apifabclient.Peer) api.PeerGroup {
	return pgresolver.NewPeerGroup(peers...)
}

func peer(name string, mspID string) apifabclient.Peer {
	peer, err := sdkFabApi.NewPeer(name+":7051", "", "", configImp)
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}
	peer.SetName(name)
	peer.SetMSPID(mspID)
	return peer
}

func newMockSelectionService(membershipManager api.MembershipManager, ccDataProvider api.CCDataProvider, lbp api.LoadBalancePolicy) api.SelectionService {
	return &selectionServiceImpl{
		membershipManager: membershipManager,
		ccDataProvider:    ccDataProvider,
		pgLBP:             lbp,
		pgResolvers:       make(map[string]api.PeerGroupResolver),
	}
}

type mockMembershipManager struct {
	peerConfigs map[string][]sdkApi.Peer
}

func (m *mockMembershipManager) GetPeersOfChannel(channelID string, poll bool) api.ChannelMembership {
	return api.ChannelMembership{Peers: m.peerConfigs[channelID], PollingEnabled: poll}
}

func newMockMembershipManager() *mockMembershipManager {
	return &mockMembershipManager{peerConfigs: make(map[string][]sdkApi.Peer)}
}

func (m *mockMembershipManager) add(channelID string, peers ...sdkApi.Peer) *mockMembershipManager {
	m.peerConfigs[channelID] = []sdkApi.Peer(peers)
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

func toString(peers []apifabclient.Peer) string {
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
