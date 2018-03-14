/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/msp"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/msp"
	pb_msp "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// CustomIdentityManager is used for retriving user's identity manager
type CustomIdentityManager struct {
	*msp.IdentityManager
	orgName        string
	embeddedUsers  map[string]coreApi.TLSKeyPair
	keyDir         string
	certDir        string
	config         coreApi.Config
	cryptoProvider coreApi.CryptoSuite
}

// Internal representation of a Fabric user
type user struct {
	mspID                 string
	name                  string
	enrollmentCertificate []byte
	privateKey            coreApi.Key
}

// NewCustomIdentityManager Constructor for a custom identity manager.
func NewCustomIdentityManager(orgName string, cryptoProvider coreApi.CryptoSuite, config coreApi.Config, mspConfigPath string) (mspApi.IdentityManager, error) {
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

	mspConfigPath = filepath.Join(orgConfig.CryptoPath, mspConfigPath)

	return &CustomIdentityManager{orgName: orgName, config: config, embeddedUsers: orgConfig.Users, keyDir: mspConfigPath + "/keystore", certDir: mspConfigPath + "/signcerts", cryptoProvider: cryptoProvider}, nil
}

// GetSigningIdentity will sign the given object with provided key,
func (c *CustomIdentityManager) GetSigningIdentity(userName string) (*mspApi.SigningIdentity, error) {
	if userName == "" {
		return nil, errors.New(errors.GeneralError, "username is required")
	}

	mspID, err := c.config.MSPID(c.orgName)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "MSP ID config read failed")
	}

	enrollmentCert, err := c.getEnrollmentCert(userName)

	if err != nil {
		return nil, err
	}

	//Get Key from Pem bytes
	key, err := getCryptoSuiteKeyFromPem(enrollmentCert, c.cryptoProvider)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to get cryptosuite key from enrollment cert")
	}

	//Get private key using SKI
	privateKey, err := c.cryptoProvider.GetKey(key.SKI())
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to get private key")
	}

	// make sure the key is private for the signingIdentity
	if !privateKey.Private() {
		return nil, errors.New(errors.GeneralError, "failed to get private key, found a public key instead")
	}

	signingIdentity := &mspApi.SigningIdentity{MSPID: mspID, PrivateKey: privateKey, EnrollmentCert: enrollmentCert}

	return signingIdentity, nil
}

// GetUser returns a user for the given user name
func (c *CustomIdentityManager) GetUser(userName string) (mspApi.User, error) {
	signingIdentity, err := c.GetSigningIdentity(userName)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to get signing identity")
	}

	return &user{
		mspID: signingIdentity.MSPID,
		name:  userName,
		enrollmentCertificate: signingIdentity.EnrollmentCert,
		privateKey:            signingIdentity.PrivateKey,
	}, nil

}

func (c *CustomIdentityManager) getEnrollmentCert(userName string) ([]byte, error) {
	var err error

	certPem := c.embeddedUsers[strings.ToLower(userName)].Cert.Pem
	certPath := c.embeddedUsers[strings.ToLower(userName)].Cert.Path

	var enrollmentCertBytes []byte

	if certPem != "" {
		enrollmentCertBytes = []byte(certPem)
	} else if certPath != "" {
		enrollmentCertBytes, err = ioutil.ReadFile(certPath)
		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "reading enrollment cert path failed")
		}
	} else if c.certDir != "" {
		enrollmentCertDir := strings.Replace(c.certDir, "{userName}", userName, -1)
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

func getCryptoSuiteKeyFromPem(idBytes []byte, cryptoSuite coreApi.CryptoSuite) (coreApi.Key, error) {
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

//MSPID return msp id
func (u *user) MSPID() string {
	return u.mspID
}

//Name return user name
func (u *user) Name() string {
	return u.name
}

//SerializedIdentity return serialized identity
func (u *user) SerializedIdentity() ([]byte, error) {
	serializedIdentity := &pb_msp.SerializedIdentity{Mspid: u.MSPID(),
		IdBytes: u.EnrollmentCertificate()}
	identity, err := proto.Marshal(serializedIdentity)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "marshal serializedIdentity failed")
	}
	return identity, nil
}

//PrivateKey return private key
func (u *user) PrivateKey() coreApi.Key {
	return u.privateKey
}

//EnrollmentCertificate return enrollment certificate
func (u *user) EnrollmentCertificate() []byte {
	return u.enrollmentCertificate
}
