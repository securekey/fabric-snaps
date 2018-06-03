/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config/endpoint"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories"
)

const (
	mspConfigPath              = "../../../../cmd/sampleconfig/msp"
	keyStorePath               = "../../../../cmd/sampleconfig/msp/keystore/"
	invalidMspConfigPath       = "/some/sample/msp/configpath"
	orgName                    = "sampleOrg"
	errorMspConfigPathRequired = "failed to create new credential manager: either mspConfigPath or an embedded list of users is required"
	errorOrgNameRequired       = "failed to create new credential manager: orgName is required"
	errorCryptoSuiteRequired   = "failed to create new credential manager: cryptoProvider is required"
	errorConfigRequired        = "failed to create new credential manager: config is required"
	errorFindPrivateKeyfailed  = "find enrollment cert path failed: read directory failed:"
	mspID                      = "sample-msp-id"
	txnSnapUser                = "Txn-Snap-User"
)

var tNetworkConfig = initNetworkConfigWithOrgEmbeddedUsers()

type testConfig struct {
	mocks.MockConfig
}

// MSPID not implemented
func (c *testConfig) MSPID(org string) (string, error) {
	return mspID, nil
}

// NetworkConfig creates a test network config with some orgs for testing
func (c *testConfig) NetworkConfig() *fabApi.NetworkConfig {
	return tNetworkConfig
}

func initNetworkConfigWithOrgEmbeddedUsers() *fabApi.NetworkConfig {
	org1KeyPair := map[string]endpoint.TLSKeyPair{
		txnSnapUser: {
			Key:  endpoint.TLSConfig{Path: "/path/to/sampleOrg/Txn-Snap-User/key", Pem: "some_sampleOrg_Txn-Snap-User_key_content"},
			Cert: endpoint.TLSConfig{Path: "/path/to/sampleOrg/Txn-Snap-User/cert", Pem: "some_sampleOrg_Txn-Snap-User_cert_content"},
		},
	}

	orgs := map[string]fabApi.OrganizationConfig{
		strings.ToLower(orgName): { // simulate viper key name structure using lowercase
			Users: org1KeyPair, // set Users with embedded certs
			MSPID: mspID,
		},
	}
	return initNetworkConfig(orgs)
}

func initNetworkConfigWithMSPConfigPath() *fabApi.NetworkConfig {
	orgs := map[string]fabApi.OrganizationConfig{
		strings.ToLower(orgName): { // simulate viper key name structure using lowercase
			CryptoPath: "../test/org1", // set CryptoPath
			MSPID:      mspID,
		},
	}
	return initNetworkConfig(orgs)
}

func initNetworkConfig(orgs map[string]fabApi.OrganizationConfig) *fabApi.NetworkConfig {
	network := &fabApi.NetworkConfig{Organizations: orgs}

	return network
}

func TestCustomIdentityMgr(t *testing.T) {
	//Positive Scenario
	identityManager := getIdentityManager(t, mspConfigPath, orgName, &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager == nil {
		t.Fatal("Expected valid identity manager")
	}

	// temporarily remove the list of embedded users to check for the org's CryptoPath
	tNetworkConfig = initNetworkConfigWithMSPConfigPath()
	// test empty embedded user certs and empty mspConfigPath
	identityManager = getIdentityManager(t, "", orgName, &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager != nil {
		t.Fatal("Expected nil identity manager")
	}

	// reset customMspPkg with mspConfigPath
	identityManager = getIdentityManager(t, mspConfigPath, orgName, &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager == nil {
		t.Fatal("Expected valid identity manager")
	}

	// reset config with list of embedded users
	tNetworkConfig = initNetworkConfigWithOrgEmbeddedUsers()

	// test empty org name
	identityManager = getIdentityManager(t, mspConfigPath, "", &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager != nil {
		t.Fatal("Expected nil identity manager")
	}

	// test empty config
	identityManager = getIdentityManager(t, mspConfigPath, orgName, nil, factories.GetSuite(factory.GetDefault()))
	if identityManager != nil {
		t.Fatal("Expected nil identity manager")
	}
	// test empty cryptoProvider
	identityManager = getIdentityManager(t, mspConfigPath, orgName, &testConfig{}, nil)
	if identityManager != nil {
		t.Fatal("Expected nil identity manager")
	}
	// test happy path using embedded users without org.CryptoPath (latest tNetWorkConfig assignment above)
	identityManager = getIdentityManager(t, "", orgName, &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager == nil {
		t.Fatal("Expected vaild identity manager")
	}

}

func getIdentityManager(t *testing.T, mspConfigPath string, orgName string, config fabApi.EndpointConfig, cryptoProvider coreApi.CryptoSuite) mspApi.IdentityManager {
	customMspPkg := &CustomMspPkg{CryptoPath: mspConfigPath}
	mspProvider, err := customMspPkg.CreateIdentityManagerProvider(config, cryptoProvider, nil)
	if err != nil {
		t.Fatalf("Unexpected error '%s'", err)
	}
	if mspProvider == nil {
		t.Fatal("Expected valid msp provider")
	}
	identityManager, _ := mspProvider.IdentityManager(orgName)
	return identityManager
}

func TestGetSigningIdentity(t *testing.T) {

	identityManager := getIdentityManager(t, mspConfigPath, orgName, &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager == nil {
		t.Fatal("Expected vaild identity manager")
	}

	signingIdentity, err := identityManager.GetSigningIdentity(txnSnapUser)

	if err != nil {
		t.Fatalf("Not supposed to get error when getting signingIdentity, but got : %s", err)
	}

	if signingIdentity == nil {
		t.Fatal("Expected to get valid signing identity")
	}

	if signingIdentity.Identifier().MSPID != mspID || signingIdentity.PrivateKey() == nil || signingIdentity.PublicVersion().EnrollmentCertificate() == nil ||
		string(signingIdentity.PublicVersion().EnrollmentCertificate()) == "" {
		t.Fatal("Invalid signing identity")
	}

	if !verifyBytes(t, signingIdentity.PublicVersion().EnrollmentCertificate(), "../../../../cmd/sampleconfig/msp/signcerts/cert.pem") {
		t.Fatal(" signingIdentity.EnrollmentCert cert is invalid")
	}

	//Negative Case

	identityManager = getIdentityManager(t, invalidMspConfigPath, orgName, &testConfig{}, factories.GetSuite(factory.GetDefault()))
	if identityManager == nil {
		t.Fatal("Expected vaild identity manager")
	}
	signingIdentity, err = identityManager.GetSigningIdentity(txnSnapUser)
	if err == nil {
		t.Fatal("Supposed to get error for credential manager GetSigningIdentity for invalid msp config path")
	}
	if !strings.HasPrefix(err.Error(), errorFindPrivateKeyfailed) {
		t.Fatalf("Unexpected error for credential manager GetSigningIdentity, expected '%s', got : %s", errorFindPrivateKeyfailed, err.Error())
	}

}

func verifyBytes(t *testing.T, testBytes []byte, path string) bool {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read bytes, err : %s", err)
	}

	if string(testBytes) != string(fileBytes) {
		return false
	}

	return true
}

func TestMain(m *testing.M) {

	opts := &factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &factory.FileKeystoreOpts{KeyStorePath: keyStorePath},
		},
	}
	factory.InitFactories(opts)

	os.Exit(m.Run())
}
