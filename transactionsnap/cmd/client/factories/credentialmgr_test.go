/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"testing"

	"io/ioutil"

	"strings"

	"os"

	apiconfig "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	mocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	"github.com/hyperledger/fabric/bccsp/factory"
)

const (
	mspConfigPath              = "../../sampleconfig/msp"
	keyStorePath               = "../../sampleconfig/msp/keystore/"
	invalidMspConfigPath       = "/some/sample/msp/configpath"
	orgName                    = "sampleOrg"
	errorMspConfigPathRequired = "failed to create new credential manager: either mspConfigPath or an embedded list of users is required"
	errorOrgNameRequired       = "failed to create new credential manager: orgName is required"
	errorCryptoSuiteRequired   = "failed to create new credential manager: cryptoProvider is required"
	errorConfigRequired        = "failed to create new credential manager: config is required"
	errorFindPrivateKeyfailed  = "find private key path from Dir failed: read directory failed:"
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
func (c *testConfig) NetworkConfig() (*apiconfig.NetworkConfig, error) {
	return tNetworkConfig, nil
}

func initNetworkConfigWithOrgEmbeddedUsers() *apiconfig.NetworkConfig {
	org1KeyPair := map[string]apiconfig.TLSKeyPair{
		txnSnapUser: {
			Key:  apiconfig.TLSConfig{Path: "/path/to/sampleOrg/Txn-Snap-User/key", Pem: "some_sampleOrg_Txn-Snap-User_key_content"},
			Cert: apiconfig.TLSConfig{Path: "/path/to/sampleOrg/Txn-Snap-User/cert", Pem: "some_sampleOrg_Txn-Snap-User_cert_content"},
		},
	}

	orgs := map[string]apiconfig.OrganizationConfig{
		strings.ToLower(orgName): { // simulate viper using lower case
			Users: org1KeyPair,
			MspID: mspID,
		},
	}
	return initNetworkConfig(orgs)
}

func initNetworkConfigWithMSPConfigPath() *apiconfig.NetworkConfig {
	orgs := map[string]apiconfig.OrganizationConfig{
		strings.ToLower(orgName): { // simulate viper key name structure using lowercase
			CryptoPath: "../test/org1",
			MspID:      mspID,
		},
	}
	return initNetworkConfig(orgs)
}

func initNetworkConfig(orgs map[string]apiconfig.OrganizationConfig) *apiconfig.NetworkConfig {
	network := &apiconfig.NetworkConfig{Organizations: orgs}

	return network
}

func TestCredentialManagerProviderFactory(t *testing.T) {
	//Positive Scenario
	credentialMgrfactory := &CredentialManagerProviderFactory{CryptoPath: mspConfigPath}
	credentialMgr, err := credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))

	if err != nil {
		t.Fatalf("Not supposed to get error for getting new credential manager, error: %s", err)
	}

	if credentialMgr == nil {
		t.Fatalf("Expected valid credential manager")
	}

	// temporarily remove the list of embedded users to check for the org's CryptoPath
	tNetworkConfig = initNetworkConfigWithMSPConfigPath()
	// test empty embedded user certs and empty mspConfigPath
	credentialMgrfactory = &CredentialManagerProviderFactory{}
	credentialMgr, err = credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))
	if err == nil || err.Error() != errorMspConfigPathRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorMspConfigPathRequired, err)
	}

	// reset credentialMgrfactory with mspConfigPath
	credentialMgrfactory = &CredentialManagerProviderFactory{CryptoPath: mspConfigPath}
	// test happy path with org.CryptPath and no embedded users (tNetWorkConfig assignment above)
	credentialMgr, err = credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))
	if err != nil {
		t.Fatalf("Unexpected error '%s'", err)
	}

	// reset config with list of embedded users
	tNetworkConfig = initNetworkConfigWithOrgEmbeddedUsers()

	// test empty org name
	credentialMgr, err = credentialMgrfactory.NewCredentialManager("", &testConfig{}, GetSuite(factory.GetDefault()))
	if err == nil || err.Error() != errorOrgNameRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorOrgNameRequired, err)
	}
	// test empty config
	credentialMgr, err = credentialMgrfactory.NewCredentialManager(orgName, nil, GetSuite(factory.GetDefault()))
	if err == nil || err.Error() != errorConfigRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorConfigRequired, err)
	}
	// test empty cryptoProvider
	credentialMgr, err = credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, nil)
	if err == nil || err.Error() != errorCryptoSuiteRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorCryptoSuiteRequired, err)
	}
	// test happey path using embedded users without org.CryptoPath (latest tNetWorkConfig assignment above)
	credentialMgr, err = credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))
	if err != nil {
		t.Fatalf("Unexpected error '%s'", err)
	}
}

func TestCredentialManagerGetSigningIdentity(t *testing.T) {

	credentialMgrfactory := &CredentialManagerProviderFactory{CryptoPath: mspConfigPath}
	credentialMgr, err := credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))

	if err != nil {
		t.Fatalf("Not supposed to get error when getting credentialMgr, but got : %s", err.Error())
	}

	signingIdentity, err := credentialMgr.GetSigningIdentity(txnSnapUser)

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

	if !verifyBytes(t, signingIdentity.EnrollmentCert, "../../sampleconfig/msp/signcerts/cert.pem") {
		t.Fatalf(" signingIdentity.EnrollmentCert cert is invalid")
	}

	//Negative Case
	credentialMgrfactory = &CredentialManagerProviderFactory{CryptoPath: invalidMspConfigPath}
	credentialMgr, err = credentialMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))
	signingIdentity, err = credentialMgr.GetSigningIdentity(txnSnapUser)

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
