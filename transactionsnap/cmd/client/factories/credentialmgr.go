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
	"github.com/hyperledger/fabric/bccsp"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/securekey/fabric-snaps/util/errors"
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
		return nil, errors.Wrap(errors.GeneralError, err, "failed to create new credential manager")
	}

	return credentialMgr, nil
}

// credentialManager is used for retriving user's signing identity (ecert + private key)
type credentialManager struct {
	orgName        string
	embeddedUsers  map[string]apiconfig.TLSKeyPair
	keyDir         string
	certDir        string
	config         apiconfig.Config
	cryptoProvider apicryptosuite.CryptoSuite
}

// NewCredentialManager Constructor for a credential manager.
// @param {string} orgName - organisation id
// @returns {CredentialManager} new credential manager
func NewCredentialManager(orgName, mspConfigPath string, config apiconfig.Config, cryptoProvider apicryptosuite.CryptoSuite) (apifabclient.CredentialManager, error) {
	if orgName == "" {
		return nil, errors.New(errors.GeneralError, "orgName is required")
	}

	if cryptoProvider == nil {
		return nil, errors.New(errors.GeneralError, "cryptoProvider is required")
	}

	if config == nil {
		return nil, errors.New(errors.GeneralError, "config is required")
	}

	netwkConfig, err := config.NetworkConfig()
	if err != nil {
		return nil, err
	}

	// viper keys are case insensitive
	orgConfig, ok := netwkConfig.Organizations[strings.ToLower(orgName)]
	if !ok {
		return nil, errors.New(errors.GeneralError, "org config retrieval failed")
	}

	if mspConfigPath == "" && len(orgConfig.Users) == 0 {
		return nil, errors.New(errors.GeneralError, "either mspConfigPath or an embedded list of users is required")
	}

	if !filepath.IsAbs(mspConfigPath) {
		cryptoConfPath := orgConfig.CryptoPath
		if strings.HasPrefix(mspConfigPath, "../") && cryptoConfPath != "" { // for paths starting  with '../' trim the prefix so the following line joins the absolute path correctly
			mspConfigPath = strings.Trim(mspConfigPath, "../")
		}
		mspConfigPath = filepath.Join(cryptoConfPath, mspConfigPath)
	}

	return &credentialManager{orgName: orgName, config: config, embeddedUsers: orgConfig.Users, keyDir: mspConfigPath + "/keystore", certDir: mspConfigPath + "/signcerts", cryptoProvider: cryptoProvider}, nil
}

// GetSigningIdentity will sign the given object with provided key,
func (mgr *credentialManager) GetSigningIdentity(userName string) (*apifabclient.SigningIdentity, error) {
	if userName == "" {
		return nil, errors.New(errors.GeneralError, "username is required")
	}

	mspID, err := mgr.config.MspID(mgr.orgName)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "MSP ID config read failed")
	}

	enrollmentCert, err := mgr.getEnrollmentCert(userName)

	if err != nil {
		return nil, err
	}

	//Get Key from Pem bytes
	key, err := getCryptoSuiteKeyFromPem(enrollmentCert, mgr.cryptoProvider)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to get cryptosuite key from enrollment cert")
	}

	//Get private key using SKI
	privateKey, err := mgr.cryptoProvider.GetKey(key.SKI())
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to get private key")
	}

	// make sure the key is private for the signingIdentity
	if !privateKey.Private() {
		return nil, errors.New(errors.GeneralError, "failed to get private key, found a public key instead")
	}

	signingIdentity := &apifabclient.SigningIdentity{MspID: mspID, PrivateKey: privateKey, EnrollmentCert: enrollmentCert}

	return signingIdentity, nil
}

func (mgr *credentialManager) getEnrollmentCert(userName string) ([]byte, error) {
	var err error

	certPem := mgr.embeddedUsers[strings.ToLower(userName)].Cert.Pem
	certPath := mgr.embeddedUsers[strings.ToLower(userName)].Cert.Path

	var enrollmentCertBytes []byte

	if certPem != "" {
		enrollmentCertBytes = []byte(certPem)
	} else if certPath != "" {
		enrollmentCertBytes, err = ioutil.ReadFile(certPath)

		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "reading enrollment cert path failed")
		}
	} else if mgr.certDir != "" {
		enrollmentCertDir := strings.Replace(mgr.certDir, "{userName}", userName, -1)
		enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)

		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "find enrollment cert path failed")
		}

		enrollmentCertBytes, err = ioutil.ReadFile(enrollmentCertPath)

		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "reading enrollment cert path failed")
		}
	} else {
		return nil, errors.Errorf(errors.GeneralError, "failed to find enrollment cert for user %s, verify the configs", userName)
	}

	return enrollmentCertBytes, nil
}

// Gets the first path from the dir directory
func getFirstPathFromDir(dir string) (string, error) {

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", errors.Wrap(errors.GeneralError, err, "read directory failed")
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

	return "", errors.New(errors.GeneralError, "no paths found")
}

func getCryptoSuiteKeyFromPem(idBytes []byte, cryptoSuite apicryptosuite.CryptoSuite) (apicryptosuite.Key, error) {
	if idBytes == nil {
		return nil, errors.New(errors.GeneralError, "getCryptoSuiteKeyFromPem error: nil idBytes")
	}

	// Decode the pem bytes
	pemCert, _ := pem.Decode(idBytes)
	if pemCert == nil {
		return nil, errors.Errorf(errors.GeneralError, "getCryptoSuiteKeyFromPem error: could not decode pem bytes [%v]", idBytes)
	}

	// get a cert
	cert, err := x509.ParseCertificate(pemCert.Bytes)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "getCryptoSuiteKeyFromPem error: failed to parse x509 cert")
	}

	// get the public key in the right format
	certPubK, err := cryptoSuite.KeyImport(cert, &bccsp.X509PublicKeyImportOpts{Temporary: true})

	return certPubK, nil
}
