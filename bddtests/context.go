/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/context"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// ADMIN type
var ADMIN = "admin"

// USER type
var USER = "user"

// BDDContext ...
type BDDContext struct {
	composition          *Composition
	clientConfig         coreApi.Config
	mutex                sync.RWMutex
	orgs                 []string
	ordererOrgID         string
	peersByChannel       map[string][]*PeerConfig
	orgsByChannel        map[string][]string
	collectionConfigs    map[string]*CollectionConfig
	clients              map[string]*fabsdk.ClientContext
	contexts             map[string]contextApi.Client
	orgChannelClients    map[string]*channel.Client
	peersMspID           map[string]string
	clientConfigFilePath string
	clientConfigFileName string
	snapsConfigFilePath  string
	testCCPath           string
	createdChannels      map[string]bool
}

// PeerConfig holds the peer configuration and org ID
type PeerConfig struct {
	OrgID  string
	Config coreApi.PeerConfig
	MspID  string
	PeerID string
}

// CollectionConfig contains the private data collection config
type CollectionConfig struct {
	Name              string
	Policy            string
	RequiredPeerCount int32
	MaxPeerCount      int32
}

// NewBDDContext create new BDDContext
func NewBDDContext(orgs []string, ordererOrgID string, clientConfigFilePath string, clientConfigFileName string,
	snapsConfigFilePath string, peersMspID map[string]string, testCCPath string) (*BDDContext, error) {
	instance := BDDContext{
		orgs:                 orgs,
		peersByChannel:       make(map[string][]*PeerConfig),
		contexts:             make(map[string]contextApi.Client),
		orgsByChannel:        make(map[string][]string),
		clients:              make(map[string]*fabsdk.ClientContext),
		collectionConfigs:    make(map[string]*CollectionConfig),
		orgChannelClients:    make(map[string]*channel.Client),
		clientConfigFilePath: clientConfigFilePath,
		clientConfigFileName: clientConfigFileName,
		snapsConfigFilePath:  snapsConfigFilePath,
		peersMspID:           peersMspID,
		testCCPath:           testCCPath,
		ordererOrgID:         ordererOrgID,
		createdChannels:      make(map[string]bool),
	}
	return &instance, nil
}

// BeforeScenario execute code before bdd scenario
func (b *BDDContext) BeforeScenario(scenarioOrScenarioOutline interface{}) {
	sdk, err := fabsdk.New(config.FromFile(b.clientConfigFilePath + b.clientConfigFileName))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}
	b.clientConfig = sdk.Config()
	for _, org := range b.orgs {
		// load org admin
		orgAdmin := fmt.Sprintf("%s_%s", org, ADMIN)
		b.contexts[orgAdmin], err = sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg(org))()
		if err != nil {
			panic(fmt.Sprintf("Failed to get admin context: %s", err))
		}
		b.clients[orgAdmin] = sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(org))
		// load org user
		orgUser := fmt.Sprintf("%s_%s", org, USER)
		b.contexts[orgUser], err = sdk.Context(fabsdk.WithUser("Admin"), fabsdk.WithOrg(org))()
		if err != nil {
			panic(fmt.Sprintf("Failed to get admin context: %s", err))
		}
		b.clients[orgUser] = sdk.NewClient(fabsdk.WithUser("User1"), fabsdk.WithOrg(org))

	}

}

// AfterScenario execute code after bdd scenario
func (b *BDDContext) AfterScenario(interface{}, error) {

	for _, orgChannelClient := range b.orgChannelClients {
		orgChannelClient.Close()
	}

}

//FindPKCS11Lib find lib based on configuration
func FindPKCS11Lib(configuredLib string) string {
	logger.Debugf("PKCS library configurations paths  %s ", configuredLib)
	var lib string
	if configuredLib != "" {
		possibilities := strings.Split(configuredLib, ",")
		for _, path := range possibilities {
			trimpath := strings.TrimSpace(path)
			if _, err := os.Stat(trimpath); !os.IsNotExist(err) {
				lib = trimpath
				break
			}
		}
	}
	logger.Debugf("Found pkcs library '%s'", lib)
	return lib
}

// Orgs returns the orgs
func (b *BDDContext) Orgs() []string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.orgs
}

// PeersByChannel returns the peers for the given channel
func (b *BDDContext) PeersByChannel(channelID string) []*PeerConfig {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.peersByChannel[channelID]
}

// OrgsByChannel returns the orgs for the given channel
func (b *BDDContext) OrgsByChannel(channelID string) []string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.orgsByChannel[channelID]
}

// CollectionConfig returns the private data collection configuration for the given collection name.
// If the collection configuration does not exist then nil is returned.
func (b *BDDContext) CollectionConfig(coll string) *CollectionConfig {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.collectionConfigs[coll]
}

// OrgClient returns the org client
func (b *BDDContext) OrgClient(org, userType string) *fabsdk.ClientContext {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.clients[fmt.Sprintf("%s_%s", org, userType)]
}

// OrgChannelClient returns the org channel client
func (b *BDDContext) OrgChannelClient(org, userType, channelID string) (*channel.Client, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	if orgChanClient, ok := b.orgChannelClients[fmt.Sprintf("%s_%s_%s", org, userType, channelID)]; ok {
		return orgChanClient, nil
	}

	orgChanClient, err := b.OrgClient(org, userType).Channel(channelID)
	if err != nil {
		return nil, err
	}
	b.orgChannelClients[fmt.Sprintf("%s_%s_%s", org, userType, channelID)] = orgChanClient
	return orgChanClient, nil
}

// OrgUserContext returns the org user context
func (b *BDDContext) OrgUserContext(org, userType string) contextApi.Client {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.contexts[fmt.Sprintf("%s_%s", org, userType)]
}

// ClientConfig returns client config
func (b *BDDContext) ClientConfig() coreApi.Config {
	return b.clientConfig
}

// OrdererOrgID returns orderer org id
func (b *BDDContext) OrdererOrgID() string {
	return b.ordererOrgID
}

// ChannelCreated returns true if channel already created
func (b *BDDContext) ChannelCreated(channelID string) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.createdChannels[channelID]
}

// PeerConfigForChannel returns a single peer for the given channel or nil if
// no peers are configured for the channel
func (b *BDDContext) PeerConfigForChannel(channelID string) *PeerConfig {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	pconfigs := b.peersByChannel[channelID]
	if len(pconfigs) == 0 {
		logger.Warnf("Peer config not found for channel [%s]\n", channelID)
		return nil
	}
	return pconfigs[rand.Intn(len(pconfigs))]
}

// OrgIDForChannel returns a single org ID for the given channel or an error if
// no orgs are configured for the channel
func (b *BDDContext) OrgIDForChannel(channelID string) (string, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	orgIDs := b.orgsByChannel[channelID]
	if len(orgIDs) == 0 {
		return "", fmt.Errorf("org not found for channel [%s]", channelID)
	}
	return orgIDs[rand.Intn(len(orgIDs))], nil
}

// AddPeerConfigToChannel adds a peer to a channel
func (b *BDDContext) AddPeerConfigToChannel(pconfig *PeerConfig, channelID string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	pconfigs := b.peersByChannel[channelID]
	for _, pc := range pconfigs {
		if pc.OrgID == pconfig.OrgID && pc.Config.URL == pconfig.Config.URL {
			// Already added
			return
		}
	}
	pconfigs = append(pconfigs, pconfig)
	b.peersByChannel[channelID] = pconfigs

	orgsForChannel := b.orgsByChannel[channelID]
	for _, orgID := range orgsForChannel {
		if orgID == pconfig.OrgID {
			// Already added
			return
		}
	}
	b.orgsByChannel[channelID] = append(orgsForChannel, pconfig.OrgID)
}

// DefineCollectionConfig defines a new private data collection configuration
func (b *BDDContext) DefineCollectionConfig(id, name, policy string, requiredPeerCount, maxPeerCount int32) *CollectionConfig {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	config := &CollectionConfig{
		Name:              name,
		Policy:            policy,
		RequiredPeerCount: requiredPeerCount,
		MaxPeerCount:      maxPeerCount,
	}
	b.collectionConfigs[id] = config
	return config
}
