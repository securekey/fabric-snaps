/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi/context/defprovider"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories/util"
)

var logger = logging.NewLogger("transaction-fabric-client-factories")

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
		return nil, errors.New("orgName is required")
	}

	if cryptoProvider == nil {
		return nil, errors.New("cryptoProvider is required")
	}

	if config == nil {
		return nil, errors.New("config is required")
	}

	netwkConfig, err := config.NetworkConfig()
	if err != nil {
		return nil, err
	}

	// viper keys are case insensitive
	orgConfig, ok := netwkConfig.Organizations[strings.ToLower(orgName)]
	if !ok {
		return nil, errors.New("org config retrieval failed")
	}

	if mspConfigPath == "" && len(orgConfig.Users) == 0 {
		return nil, errors.New("either mspConfigPath or an embedded list of users is required")
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
		return nil, errors.New("username is required")
	}

	mspID, err := mgr.config.MspID(mgr.orgName)
	if err != nil {
		return nil, errors.WithMessage(err, "MSP ID config read failed")
	}

	privateKey, err := mgr.getPrivateKey(userName)

	if err != nil {
		return nil, err
	}

	enrollmentCert, err := mgr.getEnrollmentCert(userName)

	if err != nil {
		return nil, err
	}

	signingIdentity := &apifabclient.SigningIdentity{MspID: mspID, PrivateKey: privateKey, EnrollmentCert: enrollmentCert}

	return signingIdentity, nil
}

func (mgr *credentialManager) getPrivateKey(userName string) (apicryptosuite.Key, error) {
	keyPem := mgr.embeddedUsers[strings.ToLower(userName)].Key.Pem
	keyPath := mgr.embeddedUsers[strings.ToLower(userName)].Key.Path

	var privateKey apicryptosuite.Key
	var err error

	if keyPem != "" {
		// First try importing from the Embedded Pem
		privateKey, err = util.ImportBCCSPKeyFromPEMBytes([]byte(keyPem), mgr.cryptoProvider, true)

		if err != nil {
			return nil, errors.Wrapf(err, "import private key failed %v", keyPem)
		}
	} else if keyPath != "" {
		// Then try importing from the Embedded Path
		privateKey, err = util.ImportBCCSPKeyFromPEM(keyPath, mgr.cryptoProvider, true)

		if err != nil {
			return nil, errors.Wrapf(err, "import private key failed. keyPath: %s", keyPath)
		}
	} else if mgr.keyDir != "" {
		// Then try importing from the Crypto Path

		privateKeyDir := strings.Replace(mgr.keyDir, "{userName}", userName, -1)

		privateKeyPath, err := getFirstPathFromDir(privateKeyDir)

		if err != nil {
			return nil, errors.WithMessage(err, "find private key path from Dir failed")
		}

		privateKey, err = util.ImportBCCSPKeyFromPEM(privateKeyPath, mgr.cryptoProvider, true)

		if err != nil {
			return nil, errors.Wrapf(err, "import private key failed. keyDir: %s", mgr.keyDir)
		}
	} else {
		return nil, errors.Errorf("failed to find private key for user %s, verify the configs", userName)
	}

	return privateKey, nil
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
			return nil, errors.Wrap(err, "reading enrollment cert path failed")
		}
	} else if mgr.certDir != "" {
		enrollmentCertDir := strings.Replace(mgr.certDir, "{userName}", userName, -1)
		enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)

		if err != nil {
			return nil, errors.WithMessage(err, "find enrollment cert path failed")
		}

		enrollmentCertBytes, err = ioutil.ReadFile(enrollmentCertPath)

		if err != nil {
			return nil, errors.WithMessage(err, "reading enrollment cert path failed")
		}
	} else {
		return nil, errors.Errorf("failed to find enrollment cert for user %s, verify the configs", userName)
	}

	return enrollmentCertBytes, nil
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
