/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"fmt"
	"hash"
	"reflect"

	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/securekey/fabric-snaps/metrics/cmd/filter/metrics"
)

type cryptoSuite struct {
	bccsp bccsp.BCCSP
}

//GetSuite returns cryptosuite adaptor for given bccsp.BCCSP implementation
func GetSuite(bccsp bccsp.BCCSP) coreApi.CryptoSuite {
	return &cryptoSuite{bccsp}
}

func (c *cryptoSuite) KeyGen(opts coreApi.KeyGenOpts) (k coreApi.Key, err error) {
	key, err := c.bccsp.KeyGen(getBCCSPKeyGenOpts(opts))
	return GetKey(key), err
}

func (c *cryptoSuite) KeyImport(raw interface{}, opts coreApi.KeyImportOpts) (k coreApi.Key, err error) {
	key, err := c.bccsp.KeyImport(raw, getBCCSPKeyImportOpts(opts))
	return GetKey(key), err
}

func (c *cryptoSuite) GetKey(ski []byte) (k coreApi.Key, err error) {
	if metrics.IsDebug() {
		stopWatch := metrics.RootScope.Timer("crypto_snaps_getkey_time_seconds").Start()
		defer stopWatch.Stop()
	}
	key, err := c.bccsp.GetKey(ski)
	return GetKey(key), err
}

func (c *cryptoSuite) Hash(msg []byte, opts coreApi.HashOpts) (hash []byte, err error) {
	return c.bccsp.Hash(msg, getBCCSPHashOpts(opts))
}

func (c *cryptoSuite) GetHash(opts coreApi.HashOpts) (h hash.Hash, err error) {
	return c.bccsp.GetHash(getBCCSPHashOpts(opts))
}

func (c *cryptoSuite) Sign(k coreApi.Key, digest []byte, opts coreApi.SignerOpts) (signature []byte, err error) {
	if metrics.IsDebug() {
		stopWatch := metrics.RootScope.Timer("crypto_snaps_sign_time_seconds").Start()
		defer stopWatch.Stop()
	}
	return c.bccsp.Sign(k.(*key).key, digest, opts)
}

func (c *cryptoSuite) Verify(k coreApi.Key, signature, digest []byte, opts coreApi.SignerOpts) (valid bool, err error) {
	if metrics.IsDebug() {
		stopWatch := metrics.RootScope.Timer("crypto_snaps_verify_time_seconds").Start()
		defer stopWatch.Stop()
	}
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

func (k *key) PublicKey() (coreApi.Key, error) {
	key, err := k.key.PublicKey()
	return GetKey(key), err
}

//getBCCSPKeyImportOpts converts KeyImportOpts to fabric bccsp KeyImportTypes
//Reason: Reflect check on opts type in bccsp implementation types other than bccsp.KeyImportTypes
func getBCCSPKeyImportOpts(opts coreApi.KeyImportOpts) bccsp.KeyImportOpts {

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
func getBCCSPHashOpts(opts coreApi.HashOpts) bccsp.HashOpts {

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
func getBCCSPKeyGenOpts(opts coreApi.KeyGenOpts) bccsp.KeyGenOpts {
	keyGenOpts := reflect.TypeOf(opts).String()

	BCCSPKeyGenOptsRegistry := map[string]bccsp.KeyGenOpts{
		"*bccsp.ECDSAKeyGenOpts":     &bccsp.ECDSAKeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.ECDSAP256KeyGenOpts": &bccsp.ECDSAP256KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.ECDSAP384KeyGenOpts": &bccsp.ECDSAP384KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.AESKeyGenOpts":       &bccsp.AESKeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.AES256KeyGenOpts":    &bccsp.AES256KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.AES192KeyGenOpts":    &bccsp.AES192KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.AES128KeyGenOpts":    &bccsp.AES128KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.RSAKeyGenOpts":       &bccsp.RSAKeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.RSA1024KeyGenOpts":   &bccsp.RSA1024KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.RSA2048KeyGenOpts":   &bccsp.RSA2048KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.RSA3072KeyGenOpts":   &bccsp.RSA3072KeyGenOpts{Temporary: opts.Ephemeral()},
		"*bccsp.RSA4096KeyGenOpts":   &bccsp.RSA4096KeyGenOpts{Temporary: opts.Ephemeral()},
	}
	value, ok := BCCSPKeyGenOptsRegistry[keyGenOpts]

	if ok {
		return value
	}
	panic(fmt.Sprintf("Unknown KeyGenOpts type provided : %s", keyGenOpts))
}

//GetKey returns implementation of of cryptosuite.Key
func GetKey(newkey bccsp.Key) coreApi.Key {
	return &key{newkey}
}
