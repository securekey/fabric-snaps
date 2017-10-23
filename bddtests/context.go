/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	bccspFactory "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/bccsp/factory"
)

// BDDContext ...
type BDDContext struct {
	Client       sdkApi.FabricClient
	Channel      sdkApi.Channel
	Org1Admin    sdkApi.User
	OrdererAdmin sdkApi.User
	Org1User     sdkApi.User
	Composition  *Composition
	Sdk          *sdkFabApi.FabricSDK
}

// NewBDDContext create new BDDContext
func NewBDDContext() (*BDDContext, error) {
	instance := BDDContext{}
	return &instance, nil
}

func (b *BDDContext) beforeScenario(scenarioOrScenarioOutline interface{}) {

	confileFilePath := "./fixtures/clientconfig/config.yaml"
	sdkOptions := sdkFabApi.Options{
		ConfigFile: confileFilePath,
	}

	sdk, err := sdkFabApi.NewSDK(sdkOptions)
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	clientConfig := sdk.ConfigProvider()
	if err != nil {
		panic(fmt.Sprintf("Error initializaing config: %s", err))
	}
	// Initialize bccsp factories before calling get client
	err = bccspFactory.InitFactories(&bccspFactory.FactoryOpts{
		ProviderName: clientConfig.SecurityProvider(),
		SwOpts: &bccspFactory.SwOpts{
			HashFamily: clientConfig.SecurityAlgorithm(),
			SecLevel:   clientConfig.SecurityLevel(),
			FileKeystore: &bccspFactory.FileKeystoreOpts{
				KeyStorePath: clientConfig.KeyStorePath(),
			},
			Ephemeral: false,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed getting ephemeral software-based BCCSP [%s]", err))
	}

	cryptoSuite := bccspFactory.GetDefault()

	// Create SDK setup for the integration tests
	b.Sdk = sdk

	client := sdkFabApi.NewSystemClient(clientConfig)
	client.SetCryptoSuite(cryptoSuite)

	b.Org1Admin, err = GetAdmin(client, "org1", "peerorg1")
	if err != nil {
		panic(fmt.Sprintf("Error getting admin user: %v", err))
	}

	b.OrdererAdmin, err = GetOrdererAdmin(client, "peerorg1")
	if err != nil {
		panic(fmt.Sprintf("Error getting orderer admin user: %v", err))
	}

	b.Org1User, err = GetUser(client, "org1", "peerorg1")
	if err != nil {
		panic(fmt.Sprintf("Error getting org admin user: %v", err))
	}

	client.SetStateStore(sdk.StateStoreProvider())
	client.SetSigningManager(sdk.SigningManager())
	client.SetUserContext(b.Org1User)
	b.Client = client

}
func (b *BDDContext) afterScenario(interface{}, error) {
	// Holder for common functionality

}
