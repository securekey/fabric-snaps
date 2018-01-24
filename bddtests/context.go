/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
)

var orgname = "peerorg1"

// BDDContext ...
type BDDContext struct {
	Client       sdkApi.Resource
	Channel      sdkApi.Channel
	Org1Admin    sdkApi.IdentityContext
	OrdererAdmin sdkApi.IdentityContext
	Org1User     sdkApi.IdentityContext
	Composition  *Composition
	Sdk          *fabsdk.FabricSDK
	// clients contains a map of user IdentityContext (keys) with their respective client Resource (values)
	clients map[sdkApi.IdentityContext]sdkApi.Resource
}

// NewBDDContext create new BDDContext
func NewBDDContext() (*BDDContext, error) {
	instance := BDDContext{clients: make(map[sdkApi.IdentityContext]sdkApi.Resource, 3)}
	return &instance, nil
}

func (b *BDDContext) beforeScenario(scenarioOrScenarioOutline interface{}) {

	confileFilePath := "./fixtures/clientconfig/config.yaml"

	sdk, err := fabsdk.New(config.FromFile(confileFilePath), fabsdk.WithCorePkg(&factories.DefaultCryptoSuiteProviderFactory{}))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	// Create SDK setup for the integration tests
	b.Sdk = sdk

	userSession, err := sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(orgname)).Session()
	if err != nil {
		panic(fmt.Sprintf("Failed to create new userSession for orgAdmin1: %s", err))
	}

	client, err := sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		panic(fmt.Sprintf("Failed to create new client for userSession of orgAdmin1: %s", err))
	}

	b.Org1Admin = client.IdentityContext()
	b.clients[b.Org1Admin] = client

	userSession, err = sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(orgname)).Session()
	if err != nil {
		panic(fmt.Sprintf("Failed to create new userSession for OrdererAdmin: %s", err))
	}

	client, err = sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		panic(fmt.Sprintf("Failed to create new client for userSession of OrdererAdmin: %s", err))
	}

	b.OrdererAdmin = client.IdentityContext()
	b.clients[b.OrdererAdmin] = client

	userSession, err = sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(orgname)).Session()
	if err != nil {
		panic(fmt.Sprintf("Failed to create new userSession for Org1User: %s", err))
	}

	client, err = sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		panic(fmt.Sprintf("Failed to create new client for userSession of Org1User: %s", err))
	}

	b.Org1User = client.IdentityContext()
	b.clients[b.Org1User] = client

	// the current user client is Org1User's
	b.Client = client

}
func (b *BDDContext) afterScenario(interface{}, error) {
	// Holder for common functionality

}

// SetCurrentUserClient will set the current user's IdentityContext's client and return it
func (b *BDDContext) SetCurrentUserClient(ic sdkApi.IdentityContext) sdkApi.Resource {
	b.Client = b.clients[ic]
	return b.Client
}

// GetUserFromClient will get the corresponding user for the client passed in the argument
func (b *BDDContext) GetUserFromClient(client sdkApi.Resource) sdkApi.IdentityContext {
	for i := range b.clients {
		if b.clients[i] == client {
			return i
		}
	}
	return nil
}
