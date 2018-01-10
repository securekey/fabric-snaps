/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"io/ioutil"

	"github.com/hyperledger/fabric-sdk-go/api/apicryptosuite"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/pkg/errors"
)

// ImportBCCSPKeyFromPEM attempts to create a private BCCSP key from a pem file keyFile
func ImportBCCSPKeyFromPEM(keyFile string, myCSP apicryptosuite.CryptoSuite, temporary bool) (apicryptosuite.Key, error) {
	keyBuff, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	key, err := ImportBCCSPKeyFromPEMBytes(keyBuff, myCSP, temporary)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Failed parsing private key from key file %s", keyFile))
	}
	return key, nil
}

// ImportBCCSPKeyFromPEMBytes attempts to create a private BCCSP key from a pem byte slice
func ImportBCCSPKeyFromPEMBytes(keyBuff []byte, myCSP apicryptosuite.CryptoSuite, temporary bool) (apicryptosuite.Key, error) {
	keyFile := "pem bytes"
	key, err := pemToPrivateKey(keyBuff, nil)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Failed parsing private key from %s", keyFile))
	}
	switch key.(type) {
	case *ecdsa.PrivateKey:
		priv, err := privateKeyToDER(key.(*ecdsa.PrivateKey))
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("Failed to convert ECDSA private key for '%s'", keyFile))
		}
		sk, err := myCSP.KeyImport(priv, getECDSAPrivateKeyImportOpts(temporary))
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("Failed to import ECDSA private key for '%s'", keyFile))
		}
		return sk, nil
	case *rsa.PrivateKey:
		return nil, errors.Errorf("Failed to import RSA key from %s; RSA private key import is not supported", keyFile)
	default:
		return nil, errors.Errorf("Failed to import key from %s: invalid secret key type", keyFile)
	}
}

// PEMtoPrivateKey is a bridge for bccsp utils.PEMtoPrivateKey()
func pemToPrivateKey(raw []byte, pwd []byte) (interface{}, error) {
	return utils.PEMtoPrivateKey(raw, pwd)
}

// PrivateKeyToDER marshals is bridge for utils.PrivateKeyToDER
func privateKeyToDER(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return utils.PrivateKeyToDER(privateKey)
}

//GetECDSAPrivateKeyImportOpts options for ECDSA secret key importation in DER format
// or PKCS#8 format.
func getECDSAPrivateKeyImportOpts(ephemeral bool) apicryptosuite.KeyImportOpts {
	return &bccsp.ECDSAPrivateKeyImportOpts{Temporary: ephemeral}
}
