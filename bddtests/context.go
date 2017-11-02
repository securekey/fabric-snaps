/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
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

	// Create SDK setup for the integration tests
	b.Sdk = sdk

	client := sdkFabApi.NewSystemClient(clientConfig)

	b.Org1Admin, err = sdk.NewPreEnrolledUser("peerorg1", "Admin")
	if err != nil {
		panic(fmt.Sprintf("Error getting admin user: %v", err))
	}

	b.OrdererAdmin, err = sdk.NewPreEnrolledUser("peerorg1", "Admin")
	if err != nil {
		panic(fmt.Sprintf("Error getting orderer admin user: %v", err))
	}

	b.Org1User, err = sdk.NewPreEnrolledUser("peerorg1", "Admin")
	if err != nil {
		panic(fmt.Sprintf("Error getting org admin user: %v", err))
	}

	client.SetCryptoSuite(sdk.CryptoSuiteProvider())
	client.SetStateStore(sdk.StateStoreProvider())
	client.SetSigningManager(sdk.SigningManager())
	client.SetUserContext(b.Org1User)
	b.Client = client

}
func (b *BDDContext) afterScenario(interface{}, error) {
	// Holder for common functionality

}
