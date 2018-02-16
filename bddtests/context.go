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

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// ADMIN type
var ADMIN = "admin"

// USER type
var USER = "user"

// BDDContext ...
type BDDContext struct {
	composition          *Composition
	clientConfig         apiconfig.Config
	mutex                sync.RWMutex
	sdk                  *fabsdk.FabricSDK
	orgs                 []string
	ordererOrgID         string
	peersByChannel       map[string][]*PeerConfig
	orgsByChannel        map[string][]string
	collectionConfigs    map[string]*CollectionConfig
	clients              map[string]*fabsdk.ClientContext
	resourceClients      map[string]sdkApi.Resource
	users                map[string]sdkApi.IdentityContext
	peersMspID           map[string]string
	clientConfigFilePath string
	clientConfigFileName string
	snapsConfigFilePath  string
	testCCPath           string
}

// PeerConfig holds the peer configuration and org ID
type PeerConfig struct {
	OrgID  string
	Config apiconfig.PeerConfig
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
		users:                make(map[string]sdkApi.IdentityContext),
		orgsByChannel:        make(map[string][]string),
		resourceClients:      make(map[string]sdkApi.Resource),
		clients:              make(map[string]*fabsdk.ClientContext),
		collectionConfigs:    make(map[string]*CollectionConfig),
		clientConfigFilePath: clientConfigFilePath,
		clientConfigFileName: clientConfigFileName,
		snapsConfigFilePath:  snapsConfigFilePath,
		peersMspID:           peersMspID,
		testCCPath:           testCCPath,
		ordererOrgID:         ordererOrgID,
	}
	return &instance, nil
}

// BeforeScenario execute code before bdd scenario
func (b *BDDContext) BeforeScenario(scenarioOrScenarioOutline interface{}) {
	sdk, err := fabsdk.New(config.FromFile(b.clientConfigFilePath + b.clientConfigFileName))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}
	b.sdk = sdk
	for _, org := range b.orgs {
		// load org admin
		orgAdminClient := sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(org))
		orgAdminSession, err := orgAdminClient.Session()
		if err != nil {
			panic(fmt.Sprintf("Failed to get userSession of orgAdminClient: %s", err))
		}
		orgAdminResourceClient, err := sdk.FabricProvider().CreateResourceClient(orgAdminSession)
		if err != nil {
			panic(fmt.Sprintf("Failed to create new resource client for userSession of orgAdminClient: %s", err))
		}
		orgAdmin := fmt.Sprintf("%s_%s", org, ADMIN)
		b.users[orgAdmin] = orgAdminSession
		b.clients[orgAdmin] = orgAdminClient
		b.resourceClients[orgAdmin] = orgAdminResourceClient

		b.clientConfig = sdk.Config()

		// load org user
		orgUserClient := sdk.NewClient(fabsdk.WithUser("User1"), fabsdk.WithOrg(org))
		orgUserSession, err := orgUserClient.Session()
		if err != nil {
			panic(fmt.Sprintf("Failed to get userSession of orgUserClient: %s", err))
		}
		orgUser := fmt.Sprintf("%s_%s", org, USER)
		b.users[orgUser] = orgUserSession
		b.clients[orgUser] = orgUserClient

	}

}

// AfterScenario execute code after bdd scenario
func (b *BDDContext) AfterScenario(interface{}, error) {

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

// OrgResourceClient returns the org resource client
func (b *BDDContext) OrgResourceClient(org, userType string) sdkApi.Resource {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.resourceClients[fmt.Sprintf("%s_%s", org, userType)]
}

// OrgUser returns the org user
func (b *BDDContext) OrgUser(org, userType string) sdkApi.IdentityContext {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.users[fmt.Sprintf("%s_%s", org, userType)]
}

// ClientConfig returns client config
func (b *BDDContext) ClientConfig() apiconfig.Config {
	return b.clientConfig
}

// Sdk returns client sdk
func (b *BDDContext) Sdk() *fabsdk.FabricSDK {
	return b.sdk
}

// OrdererOrgID returns orderer org id
func (b *BDDContext) OrdererOrgID() string {
	return b.ordererOrgID
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

func (b *BDDContext) newEventHub(orgID, channelID string) sdkApi.EventHub {

	eventHub, err := b.sdk.FabricProvider().CreateEventHub(b.OrgUser(orgID, ADMIN), channelID)
	if err != nil {
		panic(fmt.Errorf("GetDefaultImplEventHub failed: %v", err))
	}

	peersConfig, err := b.clientConfig.PeersConfig(orgID)
	if err != nil {
		panic(fmt.Errorf("error reading peer config: %s", err))
	}
	if len(peersConfig) == 0 {
		panic(fmt.Errorf("no peers for org [%s]", orgID))
	}
	peerConfig := peersConfig[0]
	serverHostOverride := ""
	if str, ok := peerConfig.GRPCOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}

	peerCert, err := peerConfig.TLSCACerts.TLSCert()
	if err != nil {
		panic(fmt.Errorf("Error reading peer cert from the config: %s", err))
	}

	eventHub.SetPeerAddr(peerConfig.EventURL, peerCert, serverHostOverride)
	return eventHub
}

// EventHubForOrg returns the FabricClient for the given org
//func (b *BDDContext) EventHubForOrg(orgID string) sdkApi.EventHub {
//	b.mutex.RLock()
//	defer b.mutex.RUnlock()

//	eventHub := b.orgEventHubs[orgID]
//	if !eventHub.IsConnected() {
//		if err := eventHub.Connect(); err != nil {
//			panic(fmt.Errorf("Failed eventHub.Connect() [%s]", err))
//		}
//	}
//	return eventHub
//}
