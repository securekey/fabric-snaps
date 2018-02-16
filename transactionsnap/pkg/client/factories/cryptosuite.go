/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"hash"

	"fmt"

	"reflect"

	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defcore"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/fabpvdr"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/securekey/fabric-snaps/util/errors"
)

// DefaultCryptoSuiteProviderFactory is will provide custom factory default bccsp cryptosuite
type DefaultCryptoSuiteProviderFactory struct {
	defcore.ProviderFactory
	ProviderName string
}

// NewCryptoSuiteProvider returns a implementation of factory default bccsp cryptosuite
func (f *DefaultCryptoSuiteProviderFactory) NewCryptoSuiteProvider(config apiconfig.Config) (apicryptosuite.CryptoSuite, error) {
	bccspSuite, err := factory.GetBCCSP(f.ProviderName)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error creating new cryptosuite provider")
	}
	return GetSuite(bccspSuite), nil
}

func (f *DefaultCryptoSuiteProviderFactory) NewFabricProvider(context fab.ProviderContext) (apisdk.FabricProvider, error) {
	return &CustomFabricProvider{FabricProvider: fabpvdr.New(context), providerContext: context}, nil

}

type CustomFabricProvider struct {
	*fabpvdr.FabricProvider
	providerContext fab.ProviderContext
}

func (f *CustomFabricProvider) CreateEventHub(ic fab.IdentityContext, channelID string) (fab.EventHub, error) {
	return nil, nil
}

//GetSuite returns cryptosuite adaptor for given bccsp.BCCSP implementation
func GetSuite(bccsp bccsp.BCCSP) apicryptosuite.CryptoSuite {
	return &cryptoSuite{bccsp}
}

//GetKey returns implementation of of cryptosuite.Key
func GetKey(newkey bccsp.Key) apicryptosuite.Key {
	return &key{newkey}
}

type cryptoSuite struct {
	bccsp bccsp.BCCSP
}

func (c *cryptoSuite) KeyGen(opts apicryptosuite.KeyGenOpts) (k apicryptosuite.Key, err error) {
	key, err := c.bccsp.KeyGen(getBCCSPKeyGenOpts(opts))
	return GetKey(key), err
}

func (c *cryptoSuite) KeyImport(raw interface{}, opts apicryptosuite.KeyImportOpts) (k apicryptosuite.Key, err error) {
	key, err := c.bccsp.KeyImport(raw, getBCCSPKeyImportOpts(opts))
	return GetKey(key), err
}

func (c *cryptoSuite) GetKey(ski []byte) (k apicryptosuite.Key, err error) {
	key, err := c.bccsp.GetKey(ski)
	return GetKey(key), err
}

func (c *cryptoSuite) Hash(msg []byte, opts apicryptosuite.HashOpts) (hash []byte, err error) {
	return c.bccsp.Hash(msg, getBCCSPHashOpts(opts))
}

func (c *cryptoSuite) GetHash(opts apicryptosuite.HashOpts) (h hash.Hash, err error) {
	return c.bccsp.GetHash(getBCCSPHashOpts(opts))
}

func (c *cryptoSuite) Sign(k apicryptosuite.Key, digest []byte, opts apicryptosuite.SignerOpts) (signature []byte, err error) {
	return c.bccsp.Sign(k.(*key).key, digest, opts)
}

func (c *cryptoSuite) Verify(k apicryptosuite.Key, signature, digest []byte, opts apicryptosuite.SignerOpts) (valid bool, err error) {
	return c.bccsp.Verify(k.(*key).key, signature, digest, opts)
}

type key struct {
	key bccsp.Key
}

func (k *key) Bytes() ([]byte, error) {
	return k.key.Bytes()
}

func (k *key) SKI() []byte {
	return k.key.SKI()
}

func (k *key) Symmetric() bool {
	return k.key.Symmetric()
}

func (k *key) Private() bool {
	return k.key.Private()
}

func (k *key) PublicKey() (apicryptosuite.Key, error) {
	key, err := k.key.PublicKey()
	return GetKey(key), err
}

//getBCCSPKeyImportOpts converts KeyImportOpts to fabric bccsp KeyImportTypes
//Reason: Reflect check on opts type in bccsp implementation types other than bccsp.KeyImportTypes
func getBCCSPKeyImportOpts(opts apicryptosuite.KeyImportOpts) bccsp.KeyImportOpts {

	keyImportType := reflect.TypeOf(opts).String()

	switch keyImportType {

	case "*bccsp.AES256ImportKeyOpts":
		return &bccsp.AES256ImportKeyOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.HMACImportKeyOpts":
		return &bccsp.HMACImportKeyOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.ECDSAPKIXPublicKeyImportOpts":
		return &bccsp.ECDSAPKIXPublicKeyImportOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.ECDSAPrivateKeyImportOpts":
		return &bccsp.ECDSAPrivateKeyImportOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.ECDSAGoPublicKeyImportOpts":
		return &bccsp.ECDSAGoPublicKeyImportOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.RSAGoPublicKeyImportOpts":
		return &bccsp.RSAGoPublicKeyImportOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.X509PublicKeyImportOpts":
		return &bccsp.X509PublicKeyImportOpts{Temporary: opts.Ephemeral()}
	}

	panic(fmt.Sprintf("Unknown KeyImportOpts type provided : %s", keyImportType))
}

//getBCCSPHashOpts converts HashOpts to fabric bccsp HashOpts
//Reason: Reflect check on opts type in bccsp implementation types other than bccsp.HashOpts
func getBCCSPHashOpts(opts apicryptosuite.HashOpts) bccsp.HashOpts {

	hashOpts := reflect.TypeOf(opts).String()

	switch hashOpts {

	case "*bccsp.SHAOpts":
		return &bccsp.SHAOpts{}

	case "*bccsp.SHA256Opts":
		return &bccsp.SHA256Opts{}

	case "*bccsp.SHA384Opts":
		return &bccsp.SHA384Opts{}

	case "*bccsp.SHA3_256Opts":
		return &bccsp.SHA3_256Opts{}

	case "*bccsp.SHA3_384Opts":
		return &bccsp.SHA3_384Opts{}

	}

	panic(fmt.Sprintf("Unknown HashOpts type provided : %s", hashOpts))
}

//getBCCSPKeyGenOpts converts KeyGenOpts to fabric bccsp KeyGenOpts
//Reason: Reflect check on opts type in bccsp implementation types other than bccsp.KeyGenOpts
func getBCCSPKeyGenOpts(opts apicryptosuite.KeyGenOpts) bccsp.KeyGenOpts {

	keyGenOpts := reflect.TypeOf(opts).String()

	switch keyGenOpts {

	case "*bccsp.ECDSAKeyGenOpts":
		return &bccsp.ECDSAKeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.ECDSAP256KeyGenOpts":
		return &bccsp.ECDSAP256KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.ECDSAP384KeyGenOpts":
		return &bccsp.ECDSAP384KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.AESKeyGenOpts":
		return &bccsp.AESKeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.AES256KeyGenOpts":
		return &bccsp.AES256KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.AES192KeyGenOpts":
		return &bccsp.AES192KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.AES128KeyGenOpts":
		return &bccsp.AES128KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.RSAKeyGenOpts":
		return &bccsp.RSAKeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.RSA1024KeyGenOpts":
		return &bccsp.RSA1024KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.RSA2048KeyGenOpts":
		return &bccsp.RSA2048KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.RSA3072KeyGenOpts":
		return &bccsp.RSA3072KeyGenOpts{Temporary: opts.Ephemeral()}

	case "*bccsp.RSA4096KeyGenOpts":
		return &bccsp.RSA4096KeyGenOpts{Temporary: opts.Ephemeral()}

	}

	panic(fmt.Sprintf("Unknown KeyGenOpts type provided : %s", keyGenOpts))
}
