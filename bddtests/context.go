/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
	"github.com/spf13/viper"
)

var ADMIN = "admin"
var USER = "user"

// BDDContext ...
type BDDContext struct {
	composition       *Composition
	clientConfig      apiconfig.Config
	mutex             sync.RWMutex
	sdk               *fabsdk.FabricSDK
	orgs              []string
	orderers          []string
	peersByChannel    map[string][]*PeerConfig
	orgsByChannel     map[string][]string
	collectionConfigs map[string]*CollectionConfig
	clients           map[string]*fabsdk.Client
	resourceClients   map[string]sdkApi.Resource
	users             map[string]sdkApi.IdentityContext
	primaryPeer       sdkApi.Peer
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
func NewBDDContext(orgs []string, orderers []string) (*BDDContext, error) {
	instance := BDDContext{
		orgs:              orgs,
		orderers:          orderers,
		peersByChannel:    make(map[string][]*PeerConfig),
		users:             make(map[string]sdkApi.IdentityContext),
		orgsByChannel:     make(map[string][]string),
		resourceClients:   make(map[string]sdkApi.Resource),
		clients:           make(map[string]*fabsdk.Client),
		collectionConfigs: make(map[string]*CollectionConfig),
	}
	return &instance, nil
}

func (b *BDDContext) beforeScenario(scenarioOrScenarioOutline interface{}) {
	//to initialize BCCSP factory based on config options
	if err := initializeFactory(); err != nil {
		panic(fmt.Sprintf("Failed to initialize BCCSP factory %v", err))
	}

	confileFilePath := "./fixtures/clientconfig/config.yaml"

	//TODO: hardcoded DefaultCryptoSuiteProviderFactory to SW, should be dynamic based on bccsp provider type (DEV-5240)
	sdk, err := fabsdk.New(config.FromFile(confileFilePath), fabsdk.WithCorePkg(&factories.DefaultCryptoSuiteProviderFactory{ProviderName: "SW"}))
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
		orgAdminResourceClient, err := sdk.FabricProvider().NewResourceClient(orgAdminSession.Identity())
		if err != nil {
			panic(fmt.Sprintf("Failed to create new resource client for userSession of orgAdminClient: %s", err))
		}
		orgAdmin := fmt.Sprintf("%s_%s", org, ADMIN)
		b.users[orgAdmin] = orgAdminSession.Identity()
		b.clients[orgAdmin] = orgAdminClient
		b.resourceClients[orgAdmin] = orgAdminResourceClient

		b.clientConfig = orgAdminResourceClient.Config()

		// load org user
		orgUserClient := sdk.NewClient(fabsdk.WithUser("User1"), fabsdk.WithOrg(org))
		orgUserSession, err := orgUserClient.Session()
		if err != nil {
			panic(fmt.Sprintf("Failed to get userSession of orgUserClient: %s", err))
		}
		orgUser := fmt.Sprintf("%s_%s", org, USER)
		b.users[orgUser] = orgUserSession.Identity()
		b.clients[orgUser] = orgUserClient
	}
	for _, orderer := range b.orderers {

		// load orderer admin
		ordererAdminClient := sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(orderer))
		ordererAdminSession, err := ordererAdminClient.Session()
		if err != nil {
			panic(fmt.Sprintf("Failed to get userSession of ordererAdminClient: %s", err))
		}
		ordererAdmin := fmt.Sprintf("%s_%s", orderer, ADMIN)
		b.users[ordererAdmin] = ordererAdminSession.Identity()
		b.clients[ordererAdmin] = ordererAdminClient
	}

}
func (b *BDDContext) afterScenario(interface{}, error) {
	// Holder for common functionality

}

func initializeFactory() error {
	//read BCCSP config from client config file and intiailize BCCSP factory
	//this test does not support the PLUGIN option
	cViper := viper.New()
	cViper.SetConfigType("yaml")
	cViper.AddConfigPath("./fixtures/clientconfig")
	viper.SetConfigName("config")
	viper.SetEnvPrefix("core")
	cViper.AutomaticEnv()

	if err := cViper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Failed to read client config file: %v", err))
	}
	configuredProvider := cViper.GetString("client.BCCSP.Security.Provider")
	var opts *factory.FactoryOpts
	lib := FindPKCS11Lib(cViper.GetString("client.BCCSP.Security.Library"))
	ksPath := cViper.GetString("client.BCCSP.Security.KeystorePath")
	level := cViper.GetInt("client.BCCSP.Security.Level")
	alg := cViper.GetString("client.BCCSP.Security.HashAlgorithm")
	pin := cViper.GetString("client.BCCSP.Security.Pin")
	label := cViper.GetString("client.BCCSP.Security.Label")
	logger.Debugf("Configured BCCSP provider \nlib %s \npin %s \nlabel %s", lib, pin, label)

	switch configuredProvider {
	case "PKCS11":
		opts = &factory.FactoryOpts{
			ProviderName: "PKCS11",
			Pkcs11Opts: &pkcs11.PKCS11Opts{
				SecLevel:   level,
				HashFamily: alg,
				Ephemeral:  false,
				Library:    lib,
				Pin:        pin,
				Label:      label,
				FileKeystore: &pkcs11.FileKeystoreOpts{
					KeyStorePath: ksPath,
				},
			},
		}
	case "SW":
		opts = &factory.FactoryOpts{
			ProviderName: "SW",
			SwOpts: &factory.SwOpts{
				HashFamily: alg,
				SecLevel:   level,
				Ephemeral:  true,
			},
		}
	default:
		return errors.New("Unsupported PKCS11 provider")
	}
	factory.InitFactories(opts)
	return nil

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

// Orderers returns the orderers
func (b *BDDContext) Orderers() []string {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.orderers
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
func (b *BDDContext) OrgClient(org, userType string) *fabsdk.Client {
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

// OrgResourceClient returns the org resource client
func (b *BDDContext) OrgUser(org, userType string) sdkApi.IdentityContext {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.users[fmt.Sprintf("%s_%s", org, userType)]
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
