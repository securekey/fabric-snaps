/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"crypto/x509"
	"io/ioutil"
	"path/filepath"
	"strings"

	"encoding/pem"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi/context/defprovider"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

var logger = logging.NewLogger("txnsnap")

// CredentialManagerProviderFactory is will provide custom context factory for SDK
type CredentialManagerProviderFactory struct {
	defprovider.OrgClientFactory
	CryptoPath string
}

// NewCredentialManager returns a new default implementation of the credential manager
func (f *CredentialManagerProviderFactory) NewCredentialManager(orgName string, config apiconfig.Config, cryptoProvider apicryptosuite.CryptoSuite) (fab.CredentialManager, error) {

	credentialMgr, err := NewCredentialManager(orgName, f.CryptoPath, config, cryptoProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new credential manager")
	}

	return credentialMgr, nil
}

// credentialManager is used for retriving user's signing identity (ecert + private key)
type credentialManager struct {
	orgName        string
	certDir        string
	config         apiconfig.Config
	cryptoProvider apicryptosuite.CryptoSuite
}

// NewCredentialManager Constructor for a credential manager.
// @param {string} orgName - organisation id
// @returns {CredentialManager} new credential manager
func NewCredentialManager(orgName, mspConfigPath string, config apiconfig.Config, cryptoProvider apicryptosuite.CryptoSuite) (apifabclient.CredentialManager, error) {

	if mspConfigPath == "" {
		return nil, errors.New("mspConfigPath is required")
	}

	if orgName == "" {
		return nil, errors.New("orgName is required")
	}

	if cryptoProvider == nil {
		return nil, errors.New("cryptoProvider is required")
	}

	if config == nil {
		return nil, errors.New("config is required")
	}

	return &credentialManager{orgName: orgName, config: config, certDir: mspConfigPath + "/signcerts", cryptoProvider: cryptoProvider}, nil
}

// GetSigningIdentity will sign the given object with provided key,
func (mgr *credentialManager) GetSigningIdentity(userName string) (*apifabclient.SigningIdentity, error) {

	if userName == "" {
		return nil, errors.New("username is required")
	}

	enrollmentCertDir := strings.Replace(mgr.certDir, "{userName}", userName, -1)

	enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)
	if err != nil {
		return nil, errors.WithMessage(err, "find enrollment cert path failed")
	}

	mspID, err := mgr.config.MspID(mgr.orgName)
	if err != nil {
		return nil, errors.WithMessage(err, "MSP ID config read failed")
	}

	enrollmentCert, err := ioutil.ReadFile(enrollmentCertPath)
	if err != nil {
		return nil, errors.Wrap(err, "reading enrollment cert path failed")
	}

	//Get Key from Pem bytes
	key, err := getCryptoSuiteKeyFromPem(enrollmentCert, mgr.cryptoProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cryptosuite key from enrollment cert")
	}

	//Get private key using SKI
	pk, err := mgr.cryptoProvider.GetKey(key.SKI())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get private key")
	}

	//Create Signing Identity
	signingIdentity := &apifabclient.SigningIdentity{MspID: mspID, PrivateKey: pk, EnrollmentCert: enrollmentCert}

	return signingIdentity, nil

}

// Gets the first path from the dir directory
func getFirstPathFromDir(dir string) (string, error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(err, "read directory failed")
	}

	for _, p := range files {
		if p.IsDir() {
			continue
		}

		fullName := filepath.Join(dir, string(filepath.Separator), p.Name())
		logger.Debugf("Reading file %s\n", fullName)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		fullName := filepath.Join(dir, string(filepath.Separator), f.Name())
		return fullName, nil
	}

	return "", errors.New("no paths found")
}

func getCryptoSuiteKeyFromPem(idBytes []byte, cryptoSuite apicryptosuite.CryptoSuite) (apicryptosuite.Key, error) {
	if idBytes == nil {
		return nil, errors.New("getCryptoSuiteKeyFromPem error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf("getCryptoSuiteKeyFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "getCryptoSuiteKeyFromPem error: failed to parse x509 cert")
	}

	// get the public key in the right format
	certPubK, err := cryptoSuite.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})

	return certPubK, nil
}
