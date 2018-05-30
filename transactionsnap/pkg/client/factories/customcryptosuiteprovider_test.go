/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"hash"
	"os"
	"testing"

	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/pkg/errors"
)

const (
	mockIdentifier   = "mock-test"
	signedIdentifier = "-signed"
	signingKey       = "signing-key"
	hashMessage      = "-msg-bytes"
	sampleKey        = "sample-key"
	getKey           = "-getkey"
	keyImport        = "-keyimport"
	keyGen           = "-keygent"
	keyStorePath     = "../../../cmd/sampleconfig/msp/keystore/"
)

func TestDefaultCryptoSuiteFactory(t *testing.T) {

	cryptoSuiteFactory := &CustomCorePkg{ProviderName: "SW"}
	cryptoSuiteProvider, err := cryptoSuiteFactory.CreateCryptoSuiteProvider(nil)

	if err != nil {
		t.Fatal("Not supposed to get error from cryptoSuiteFactory.NewCryptoSuiteProvider")
	}

	if cryptoSuiteProvider == nil {
		t.Fatal("expected to get valid cryptosuite from cryptoSuiteFactory.NewCryptoSuiteProvider")
	}

}

func TestGetSuite(t *testing.T) {
	//Get BCCSP implementation
	samplebccsp := getMockBCCSP(mockIdentifier)

	//Get cryptosuite
	samplecryptoSuite := GetSuite(samplebccsp)

	//Verify CryptSuite
	verifyCryptoSuite(t, samplecryptoSuite)
}

func verifyCryptoSuite(t *testing.T, samplecryptoSuite coreApi.CryptoSuite) {
	//Test cryptosuite.Sign
	signedBytes, err := samplecryptoSuite.Sign(GetKey(getMockKey(signingKey)), nil, nil)
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.GetKey : %s", err)
	VerifyTrue(t, string(signedBytes) == mockIdentifier+signedIdentifier, "Got unexpected result from samplecryptoSuite.Sign")

	//Test cryptosuite.Hash
	hashBytes, err := samplecryptoSuite.Hash([]byte(hashMessage), &bccsp.SHAOpts{})
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.GetKey")
	VerifyTrue(t, string(hashBytes) == mockIdentifier+hashMessage, "Got unexpected result from samplecryptoSuite.Hash")

	//Test cryptosuite.GetKey
	key, err := samplecryptoSuite.GetKey([]byte(sampleKey))
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.GetKey")
	VerifyNotEmpty(t, key, "Not supposed to get empty key for samplecryptoSuite.GetKey")

	keyBytes, err := key.Bytes()
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.GetKey().GetBytes()")
	VerifyTrue(t, string(keyBytes) == sampleKey+getKey, "Not supposed to get empty bytes for samplecryptoSuite.GetKey().GetBytes()")

	skiBytes := key.SKI()
	VerifyTrue(t, string(skiBytes) == sampleKey+getKey, "Not supposed to get empty bytes for samplecryptoSuite.GetKey().GetSKI()")

	VerifyTrue(t, key.Private(), "Not supposed to get false for samplecryptoSuite.GetKey().Private()")
	VerifyTrue(t, key.Symmetric(), "Not supposed to get false for samplecryptoSuite.GetKey().Symmetric()")

	publikey, err := key.PublicKey()
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.GetKey().PublicKey()")
	VerifyNotEmpty(t, publikey, "Not supposed to get empty key for samplecryptoSuite.GetKey().PublicKey()")

	//Test cryptosuite.KeyImport
	key, err = samplecryptoSuite.KeyImport(nil, &bccsp.X509PublicKeyImportOpts{Temporary: true})
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.KeyImport")
	VerifyNotEmpty(t, key, "Not supposed to get empty key for samplecryptoSuite.KeyImport")

	keyBytes, err = key.Bytes()
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.KeyImport().GetBytes()")
	VerifyTrue(t, string(keyBytes) == mockIdentifier+keyImport, "Unexpected bytes for samplecryptoSuite.KeyImport().GetBytes()")

	skiBytes = key.SKI()
	VerifyTrue(t, string(skiBytes) == mockIdentifier+keyImport, "Unexpected bytes for samplecryptoSuite.KeyImport().GetSKI()")

	VerifyTrue(t, key.Private(), "Not supposed to get false for samplecryptoSuite.KeyImport().Private()")
	VerifyTrue(t, key.Symmetric(), "Not supposed to get false for samplecryptoSuite.KeyImport().Symmetric()")

	publikey, err = key.PublicKey()
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.KeyImport().PublicKey()")
	VerifyNotEmpty(t, publikey, "Not supposed to get empty key for samplecryptoSuite.KeyImport().PublicKey()")

	//Test cryptosuite.KeyGen
	key, err = samplecryptoSuite.KeyGen(&bccsp.ECDSAKeyGenOpts{})
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.KeyGen")
	VerifyNotEmpty(t, key, "Not supposed to get empty key for samplecryptoSuite.KeyGen")

	keyBytes, err = key.Bytes()
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.KeyGen().GetBytes()")
	VerifyTrue(t, string(keyBytes) == mockIdentifier+keyGen, "Unexpected bytes for samplecryptoSuite.KeyGen().GetBytes()")

	skiBytes = key.SKI()
	VerifyTrue(t, string(skiBytes) == mockIdentifier+keyGen, "Unexpected bytes for samplecryptoSuite.KeyGen().GetSKI()")

	VerifyTrue(t, key.Private(), "Not supposed to get false for samplecryptoSuite.KeyGen().Private()")
	VerifyTrue(t, key.Symmetric(), "Not supposed to get false for samplecryptoSuite.KeyGen().Symmetric()")

	publikey, err = key.PublicKey()
	VerifyEmpty(t, err, "Not supposed to get any error for samplecryptoSuite.KeyGen().PublicKey()")
	VerifyNotEmpty(t, publikey, "Not supposed to get empty key for samplecryptoSuite.KeyGen().PublicKey()")

	//Test cryptosuite.GetHash
	hash, err := samplecryptoSuite.GetHash(&bccsp.SHA256Opts{})
	VerifyNotEmpty(t, err, "Supposed to get error for samplecryptoSuite.GetHash")
	VerifyEmpty(t, hash, "Supposed to get empty hash for samplecryptoSuite.GetHash")

	//Test cryptosuite.GetHash
	valid, err := samplecryptoSuite.Verify(GetKey(getMockKey(signingKey)), nil, nil, nil)
	VerifyEmpty(t, err, "Not supposed to get error for samplecryptoSuite.Verify")
	VerifyTrue(t, valid, "Supposed to get true for samplecryptoSuite.Verify")
}

/*
	Mock implementation of bccsp.BCCSP and bccsp.Key
*/

func getMockBCCSP(identifier string) bccsp.BCCSP {
	return &mockBCCSP{identifier}
}

func getMockKey(identifier string) bccsp.Key {
	return &mockKey{identifier}
}

type mockBCCSP struct {
	identifier string
}

func (mock *mockBCCSP) KeyGen(opts bccsp.KeyGenOpts) (k bccsp.Key, err error) {
	return &mockKey{mock.identifier + keyGen}, nil
}

func (mock *mockBCCSP) KeyDeriv(k bccsp.Key, opts bccsp.KeyDerivOpts) (dk bccsp.Key, err error) {
	return &mockKey{"keyderiv"}, nil
}

func (mock *mockBCCSP) KeyImport(raw interface{}, opts bccsp.KeyImportOpts) (k bccsp.Key, err error) {
	return &mockKey{mock.identifier + keyImport}, nil
}

func (mock *mockBCCSP) GetKey(ski []byte) (k bccsp.Key, err error) {
	return &mockKey{string(ski) + getKey}, nil
}

func (mock *mockBCCSP) Hash(msg []byte, opts bccsp.HashOpts) (hash []byte, err error) {
	return []byte(mock.identifier + string(msg)), nil
}

func (mock *mockBCCSP) GetHash(opts bccsp.HashOpts) (h hash.Hash, err error) {
	return nil, errors.New("Not able to Get Hash")
}

func (mock *mockBCCSP) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	return []byte(mock.identifier + signedIdentifier), nil
}

func (mock *mockBCCSP) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	return true, nil
}

func (mock *mockBCCSP) Encrypt(k bccsp.Key, plaintext []byte, opts bccsp.EncrypterOpts) (ciphertext []byte, err error) {
	return []byte(mock.identifier + "-encrypted"), nil
}

func (mock *mockBCCSP) Decrypt(k bccsp.Key, ciphertext []byte, opts bccsp.DecrypterOpts) (plaintext []byte, err error) {
	return []byte(mock.identifier + "-decrypted"), nil
}

type mockKey struct {
	identifier string
}

func (k *mockKey) Bytes() ([]byte, error) {
	return []byte(k.identifier), nil
}

func (k *mockKey) SKI() []byte {
	return []byte(k.identifier)
}

func (k *mockKey) Symmetric() bool {
	return true
}

func (k *mockKey) Private() bool {
	return true
}

func (k *mockKey) PublicKey() (bccsp.Key, error) {
	return &mockKey{k.identifier + "-public"}, nil
}

//VerifyTrue verifies if boolean input is true, if false then fails test
func VerifyTrue(t *testing.T, input bool, msgAndArgs ...interface{}) {
	if !input {
		failTest(t, msgAndArgs)
	}
}

//VerifyEmpty Verifies if input is empty, fails test if not empty
func VerifyEmpty(t *testing.T, in interface{}, msgAndArgs ...interface{}) {
	if in == nil {
		return
	} else if in == "" {
		return
	}
	failTest(t, msgAndArgs...)
}

//VerifyNotEmpty Verifies if input is not empty, fails test if empty
func VerifyNotEmpty(t *testing.T, in interface{}, msgAndArgs ...interface{}) {
	if in != nil {
		return
	} else if in != "" {
		return
	}
	failTest(t, msgAndArgs...)
}

func failTest(t *testing.T, msgAndArgs ...interface{}) {
	if len(msgAndArgs) == 1 {
		t.Fatal(msgAndArgs[0])
	}
	if len(msgAndArgs) > 1 {
		t.Fatalf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
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
