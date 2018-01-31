/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"fmt"
	"os"
	"strings"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
	"github.com/spf13/viper"
)

var orgname = "peerorg1"

// BDDContext ...
type BDDContext struct {
	Channel      sdkApi.Channel
	Org1Admin    sdkApi.IdentityContext
	OrdererAdmin sdkApi.IdentityContext
	Org1User     sdkApi.IdentityContext
	Composition  *Composition
	// clients contains a map of user IdentityContext (keys) with their respective client Resource (values)
	clients         map[sdkApi.IdentityContext]*fabsdk.Client
	resourceClients map[sdkApi.IdentityContext]sdkApi.Resource
}

// NewBDDContext create new BDDContext
func NewBDDContext() (*BDDContext, error) {
	instance := BDDContext{resourceClients: make(map[sdkApi.IdentityContext]sdkApi.Resource, 3), clients: make(map[sdkApi.IdentityContext]*fabsdk.Client, 3)}
	return &instance, nil
}

func (b *BDDContext) beforeScenario(scenarioOrScenarioOutline interface{}) {
	//to initialize BCCSP factory based on config options
	if err := initializeFactory(); err != nil {
		panic(fmt.Sprintf("Failed to initialize BCCSP factory %v", err))
	}

	confileFilePath := "./fixtures/clientconfig/config.yaml"

	sdk, err := fabsdk.New(config.FromFile(confileFilePath), fabsdk.WithCorePkg(&factories.DefaultCryptoSuiteProviderFactory{}))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	// load org admin
	orgAdminClient := sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg(orgname))
	userSession, err := orgAdminClient.Session()
	if err != nil {
		panic(fmt.Sprintf("Failed to get userSession of orgAdminClient: %s", err))
	}
	orgAdminResourceClient, err := sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		panic(fmt.Sprintf("Failed to create new resource client for userSession of orgAdminClient: %s", err))
	}
	b.Org1Admin = orgAdminResourceClient.IdentityContext()
	b.clients[b.Org1Admin] = orgAdminClient
	b.resourceClients[b.Org1Admin] = orgAdminResourceClient

	// load org user
	orgUserClient := sdk.NewClient(fabsdk.WithUser("User1"), fabsdk.WithOrg(orgname))
	userSession, err = orgAdminClient.Session()
	if err != nil {
		panic(fmt.Sprintf("Failed to get userSession of orgUserClient: %s", err))
	}
	orgUserResourceClient, err := sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		panic(fmt.Sprintf("Failed to create new resource client for userSession of orgUserClient: %s", err))
	}
	b.Org1User = orgUserResourceClient.IdentityContext()
	b.clients[b.Org1User] = orgUserClient
	b.resourceClients[b.Org1User] = orgUserResourceClient

	// load orderer admin
	ordererAdminClient := sdk.NewClient(fabsdk.WithUser("Admin"), fabsdk.WithOrg("ordererorg"))
	userSession, err = orgAdminClient.Session()
	if err != nil {
		panic(fmt.Sprintf("Failed to get userSession of ordererAdminClient: %s", err))
	}
	ordererAdminResourceClient, err := sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		panic(fmt.Sprintf("Failed to create new resource client for userSession of ordererAdminClient: %s", err))
	}
	b.OrdererAdmin = ordererAdminResourceClient.IdentityContext()
	b.clients[b.OrdererAdmin] = ordererAdminClient
	b.resourceClients[b.OrdererAdmin] = ordererAdminResourceClient

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
