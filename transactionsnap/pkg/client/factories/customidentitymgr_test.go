/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config/endpoint"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric/bccsp/factory"
)

const (
	mspConfigPath              = "../../../cmd/sampleconfig/msp"
	keyStorePath               = "../../../cmd/sampleconfig/msp/keystore/"
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

// MspID not implemented
func (c *testConfig) MspID(org string) (string, error) {
	return mspID, nil
}

// NetworkConfig creates a test network config with some orgs for testing
func (c *testConfig) NetworkConfig() (*coreApi.NetworkConfig, error) {
	return tNetworkConfig, nil
}

func initNetworkConfigWithOrgEmbeddedUsers() *coreApi.NetworkConfig {
	org1KeyPair := map[string]coreApi.TLSKeyPair{
		txnSnapUser: {
			Key:  endpoint.TLSConfig{Path: "/path/to/sampleOrg/Txn-Snap-User/key", Pem: "some_sampleOrg_Txn-Snap-User_key_content"},
			Cert: endpoint.TLSConfig{Path: "/path/to/sampleOrg/Txn-Snap-User/cert", Pem: "some_sampleOrg_Txn-Snap-User_cert_content"},
		},
	}

	orgs := map[string]coreApi.OrganizationConfig{
		strings.ToLower(orgName): { // simulate viper key name structure using lowercase
			Users: org1KeyPair, // set Users with embedded certs
			MspID: mspID,
		},
	}
	return initNetworkConfig(orgs)
}

func initNetworkConfigWithMSPConfigPath() *coreApi.NetworkConfig {
	orgs := map[string]coreApi.OrganizationConfig{
		strings.ToLower(orgName): { // simulate viper key name structure using lowercase
			CryptoPath: "../test/org1", // set CryptoPath
			MspID:      mspID,
		},
	}
	return initNetworkConfig(orgs)
}

func initNetworkConfig(orgs map[string]coreApi.OrganizationConfig) *coreApi.NetworkConfig {
	network := &coreApi.NetworkConfig{Organizations: orgs}

	return network
}

func TestCustomIdentityMgr(t *testing.T) {
	//Positive Scenario
	customCorePkg := &CustomCorePkg{CryptoPath: mspConfigPath}
	identityManager, err := customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})

	if err != nil {
		t.Fatalf("Not supposed to get error for getting create identity manager, error: %s", err)
	}

	if identityManager == nil {
		t.Fatalf("Expected valid identity manager")
	}

	// temporarily remove the list of embedded users to check for the org's CryptoPath
	tNetworkConfig = initNetworkConfigWithMSPConfigPath()
	// test empty embedded user certs and empty mspConfigPath
	customCorePkg = &CustomCorePkg{}
	_, err = customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})
	if err == nil || err.Error() != errorMspConfigPathRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorMspConfigPathRequired, err)
	}

	// reset customCorePkg with mspConfigPath
	customCorePkg = &CustomCorePkg{CryptoPath: mspConfigPath}
	// test happy path with org.CryptPath and no embedded users (tNetWorkConfig assignment above)
	_, err = customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})
	if err != nil {
		t.Fatalf("Unexpected error '%s'", err)
	}

	// reset config with list of embedded users
	tNetworkConfig = initNetworkConfigWithOrgEmbeddedUsers()

	// test empty org name
	_, err = customCorePkg.CreateIdentityManager("", mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})
	if err == nil || err.Error() != errorOrgNameRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorOrgNameRequired, err)
	}
	// test empty config
	_, err = customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), nil)
	if err == nil || err.Error() != errorConfigRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorConfigRequired, err)
	}
	// test empty cryptoProvider
	_, err = customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), nil, &testConfig{})
	if err == nil || err.Error() != errorCryptoSuiteRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorCryptoSuiteRequired, err)
	}
	// test happy path using embedded users without org.CryptoPath (latest tNetWorkConfig assignment above)
	_, err = customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})
	if err != nil {
		t.Fatalf("Unexpected error '%s'", err)
	}
}

func TestGetSigningIdentity(t *testing.T) {

	customCorePkg := &CustomCorePkg{CryptoPath: mspConfigPath}
	identityManager, err := customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})
	if err != nil {
		t.Fatalf("Not supposed to get error for getting create identity manager, error: %s", err)
	}

	signingIdentity, err := identityManager.GetSigningIdentity(txnSnapUser)

	if err != nil {
		t.Fatalf("Not supposed to get error when getting signingIdentity, but got : %s", err.Error())
	}

	if signingIdentity == nil {
		t.Fatalf("Expected to get valid signing identity")
	}

	if signingIdentity.MspID != mspID || signingIdentity.PrivateKey == nil || signingIdentity.EnrollmentCert == nil ||
		string(signingIdentity.EnrollmentCert) == "" {
		t.Fatalf("Invalid signing identity")
	}

	if !verifyBytes(t, signingIdentity.EnrollmentCert, "../../../cmd/sampleconfig/msp/signcerts/cert.pem") {
		t.Fatalf(" signingIdentity.EnrollmentCert cert is invalid")
	}

	//Negative Case
	customCorePkg = &CustomCorePkg{CryptoPath: invalidMspConfigPath}
	identityManager, _ = customCorePkg.CreateIdentityManager(orgName, mocks.NewMockStateStore(), GetSuite(factory.GetDefault()), &testConfig{})
	signingIdentity, err = identityManager.GetSigningIdentity(txnSnapUser)
	if err == nil {
		t.Fatalf("Supposed to get error for credential manager GetSigningIdentity for invalid msp config path")
	}
	if !strings.HasPrefix(err.Error(), errorFindPrivateKeyfailed) {
		t.Fatalf("Unexpected error for credential manager GetSigningIdentity, expected '%s', got : %s", errorFindPrivateKeyfailed, err.Error())
	}

}

func verifyBytes(t *testing.T, testBytes []byte, path string) bool {
	fileBytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read bytes, err : %v ", err)
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
