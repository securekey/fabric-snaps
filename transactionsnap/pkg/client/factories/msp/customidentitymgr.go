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
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/msp"
	pb_msp "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/securekey/fabric-snaps/sanitize-master"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// CustomIdentityManager is used for retriving user's identity manager
type CustomIdentityManager struct {
	*msp.IdentityManager
	orgName        string
	mspID          string
	embeddedUsers  map[string]fabApi.CertKeyPair
	keyDir         string
	certDir        string
	config         fabApi.EndpointConfig
	cryptoProvider coreApi.CryptoSuite
}

// User is representation of a Fabric user
type User struct {
	mspID                 string
	id                    string
	enrollmentCertificate []byte
	privateKey            coreApi.Key
}

// NewCustomIdentityManager Constructor for a custom identity manager.
func NewCustomIdentityManager(orgName string, cryptoProvider coreApi.CryptoSuite, config fabApi.EndpointConfig, mspConfigPath string) (mspApi.IdentityManager, error) {
	if orgName == "" {
		return nil, errors.New(errors.GeneralError, "orgName is required")
	}

	if cryptoProvider == nil {
		return nil, errors.New(errors.GeneralError, "cryptoProvider is required")
	}

	if config == nil {
		return nil, errors.New(errors.GeneralError, "config is required")
	}

	netwkConfig := config.NetworkConfig()

	// viper keys are case insensitive
	orgConfig, ok := netwkConfig.Organizations[strings.ToLower(orgName)]
	if !ok {
		return nil, errors.New(errors.GeneralError, "org config retrieval failed")
	}

	if mspConfigPath == "" && len(orgConfig.Users) == 0 {
		return nil, errors.New(errors.GeneralError, "either mspConfigPath or an embedded list of users is required")
	}

	mspConfigPath = filepath.Join(orgConfig.CryptoPath, mspConfigPath)

	return &CustomIdentityManager{orgName: orgName, mspID: orgConfig.MSPID, config: config, embeddedUsers: orgConfig.Users, keyDir: mspConfigPath + "/keystore", certDir: mspConfigPath + "/signcerts", cryptoProvider: cryptoProvider}, nil
}

// GetSigningIdentity will sign the given object with provided key,
func (c *CustomIdentityManager) GetSigningIdentity(id string) (mspApi.SigningIdentity, error) {
	user, err := c.GetUser(id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser returns a user for the given user name
func (c *CustomIdentityManager) GetUser(userName string) (*User, error) {
	if userName == "" {
		return nil, errors.New(errors.GeneralError, "username is required")
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

	return &User{
		mspID: c.mspID,
		id:    userName,
		enrollmentCertificate: enrollmentCert,
		privateKey:            privateKey,
	}, nil

}

func (c *CustomIdentityManager) getEnrollmentCert(userName string) ([]byte, error) {

	enrollmentCertBytes := c.embeddedUsers[strings.ToLower(userName)].Cert
	if len(enrollmentCertBytes) > 0 {
		return enrollmentCertBytes, nil
	}

	if c.certDir != "" {
		enrollmentCertDir := strings.Replace(c.certDir, "{userName}", userName, -1)
		enrollmentCertPath, err := getFirstPathFromDir(enrollmentCertDir)
		if err != nil {
			return nil, errors.WithMessage(errors.GeneralError, err, "find enrollment cert path failed")
		}
		sanitize.Path(enrollmentCertPath)
		enrollmentCertBytes, err = ioutil.ReadFile(enrollmentCertPath) //nolint: gas
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
	if err != nil {
		return nil, err
	}
	return certPubK, nil
}

//Serialize return serialized identity
func (u *User) Serialize() ([]byte, error) {
	serializedIdentity := &pb_msp.SerializedIdentity{Mspid: u.mspID,
		IdBytes: u.EnrollmentCertificate()}
	identity, err := proto.Marshal(serializedIdentity)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "marshal serializedIdentity failed")
	}
	return identity, nil
}

//PrivateKey return private key
func (u *User) PrivateKey() coreApi.Key {
	return u.privateKey
}

//EnrollmentCertificate return enrollment certificate
func (u *User) EnrollmentCertificate() []byte {
	return u.enrollmentCertificate
}

// Identifier returns user identifier
func (u *User) Identifier() *mspApi.IdentityIdentifier {
	return &mspApi.IdentityIdentifier{MSPID: u.mspID, ID: u.id}
}

// Verify a signature over some message using this identity as reference
func (u *User) Verify(msg []byte, sig []byte) error {
	return errors.New(errors.GeneralError, "not implemented")
}

// PublicVersion returns the public parts of this identity
func (u *User) PublicVersion() mspApi.Identity {
	return u
}

// Sign the message
func (u *User) Sign(msg []byte) ([]byte, error) {
	return nil, errors.New(errors.GeneralError, "not implemented")
}
