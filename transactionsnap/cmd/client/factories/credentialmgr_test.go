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

	mocks "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	"github.com/hyperledger/fabric/bccsp/factory"
)

const (
	mspConfigPath              = "../../sampleconfig/msp"
	keyStorePath               = "../../sampleconfig/msp/keystore/"
	invalidMspConfigPath       = "/some/sample/msp/configpath"
	orgName                    = "sampleOrg"
	errorMspConfigPathRequired = "failed to create new credential manager: mspConfigPath is required"
	errorOrgNameRequired       = "failed to create new credential manager: orgName is required"
	errorCryptoSuiteRequired   = "failed to create new credential manager: cryptoProvider is required"
	errorConfigRequired        = "failed to create new credential manager: config is required"
	errorFindPrivateKeyfailed  = "find enrollment cert path failed: read directory failed:"
	mspID                      = "sample-msp-id"
)

var configImpl = mocks.NewMockConfig()

type testConfig struct {
	mocks.MockConfig
}

// MspID not implemented
func (c *testConfig) MspID(org string) (string, error) {
	return mspID, nil
}

func TestCredentialManagerProviderFactory(t *testing.T) {

	//Positive Scenario
	credentailMgrfactory := &CredentialManagerProviderFactory{CryptoPath: mspConfigPath}
	credentailMgr, err := credentailMgrfactory.NewCredentialManager(orgName, configImpl, GetSuite(factory.GetDefault()))

	if err != nil {
		t.Fatalf("Not supposed to get error for getting new credential manager")
	}

	if credentailMgr == nil {
		t.Fatalf("Expected valid credential manager")
	}

	credentailMgrfactory = &CredentialManagerProviderFactory{}
	credentailMgr, err = credentailMgrfactory.NewCredentialManager(orgName, configImpl, GetSuite(factory.GetDefault()))
	if err == nil || err.Error() != errorMspConfigPathRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorMspConfigPathRequired, err)
	}

	credentailMgrfactory = &CredentialManagerProviderFactory{CryptoPath: mspConfigPath}
	credentailMgr, err = credentailMgrfactory.NewCredentialManager("", configImpl, GetSuite(factory.GetDefault()))
	if err == nil || err.Error() != errorOrgNameRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorOrgNameRequired, err)
	}

	credentailMgr, err = credentailMgrfactory.NewCredentialManager(orgName, nil, GetSuite(factory.GetDefault()))
	if err == nil || err.Error() != errorConfigRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorConfigRequired, err)
	}

	credentailMgr, err = credentailMgrfactory.NewCredentialManager(orgName, configImpl, nil)
	if err == nil || err.Error() != errorCryptoSuiteRequired {
		t.Fatalf("Expected error '%s' , but got : %v", errorCryptoSuiteRequired, err)
	}

}

func TestCredentialManagerGetSigningIdentity(t *testing.T) {

	credentailMgrfactory := &CredentialManagerProviderFactory{CryptoPath: mspConfigPath}
	credentailMgr, err := credentailMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))
	signingIdentity, err := credentailMgr.GetSigningIdentity("Txn-Snap-User")

	if err != nil {
		t.Fatalf("Not supposed to get error, but got : %s", err.Error())
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
	credentailMgrfactory = &CredentialManagerProviderFactory{CryptoPath: invalidMspConfigPath}
	credentailMgr, err = credentailMgrfactory.NewCredentialManager(orgName, &testConfig{}, GetSuite(factory.GetDefault()))
	signingIdentity, err = credentailMgr.GetSigningIdentity("Txn-Snap-User")

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
