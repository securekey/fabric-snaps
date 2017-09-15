/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"net"
	"os"
	"time"

	"fmt"

	"github.com/hyperledger/fabric/bccsp"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
	"github.com/hyperledger/fabric/bccsp/utils"
	config "github.com/securekey/fabric-snaps/httpsnap/cmd/config"
)

//GetBCCSPProvider returns BCCSP factory options
func GetBCCSPProvider() *bccspFactory.FactoryOpts {
	switch config.SecurityProvider() {
	case "SW":
		opts := &bccspFactory.FactoryOpts{
			ProviderName: "SW",
			SwOpts: &bccspFactory.SwOpts{
				HashFamily: config.SecurityAlgorithm(),
				SecLevel:   config.SecurityLevel(),
				FileKeystore: &bccspFactory.FileKeystoreOpts{
					KeyStorePath: os.TempDir(),
				},
				Ephemeral: config.Ephemeral(),
			},
		}
		bccspFactory.InitFactories(opts)
		return opts

	case "PKCS11":
		providerLib := config.SecurityProviderLibPath()
		softHSMPin := config.SecurityProviderPin()
		softHSMTokenLabel := config.SecurityProviderLabel()
		pkks := pkcs11.FileKeystoreOpts{KeyStorePath: os.TempDir()}
		//PKCS11 options
		pkcsOpt := pkcs11.PKCS11Opts{
			SecLevel:     config.SecurityLevel(),
			HashFamily:   config.SecurityAlgorithm(),
			FileKeystore: &pkks,
			Library:      providerLib,
			Pin:          softHSMPin,
			Label:        softHSMTokenLabel,
			Ephemeral:    false,
		}

		opts := &bccspFactory.FactoryOpts{
			ProviderName: "PKCS11",
			Pkcs11Opts:   &pkcsOpt,
			SwOpts:       nil,
		}

		bccspFactory.InitFactories(opts)
		return opts
	default:
		panic(fmt.Sprintf("Unsupported BCCSP Provider: %s", config.SecurityProvider()))

	}
}

//GetConfiguredCSP returns BCCSP
func GetConfiguredCSP(opts *bccspFactory.FactoryOpts) (bccsp.BCCSP, error) {

	f := &bccspFactory.PKCS11Factory{}
	//
	csp, err := f.Get(opts)
	if err != nil {
		return nil, fmt.Errorf("Cannot get factory opts %v", err)
	}
	if csp == nil {
		return nil, fmt.Errorf("Cannot configure BCCSP provider %v", err)
	}
	return csp, nil
}

//GetKeysForHandle returns private and public key
func GetKeysForHandle(csp bccsp.BCCSP, SKI []byte) (bccsp.Key, error) {
	//this is private key
	key, err := csp.GetKey(SKI)
	if err != nil {
		return nil, err
	}
	//
	return key, nil
}

//GetCertificate returns certificate based on keys from HSM
func GetCertificate(key bccsp.Key) (*x509.Certificate, error) {
	template := GetCertificateTemplate()
	publicKey, err := key.PublicKey()
	if err != nil {
		return nil, err
	}
	certRaw, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey, key)
	if err != nil {
		return nil, err
	}
	cert, err := utils.DERToX509Certificate(certRaw)
	if err != nil {
		return nil, err
	}
	res := bytes.Compare(cert.Raw, certRaw)
	if res != 0 {
		return nil, err
	}
	return cert, nil
}

//GetCertificateTemplate ... should probably come from configuration (whole or just properties)
//TODO check why direct conversion using util does not work
//TODO use this certificate in httpsnap.go.getTLSConfig
func GetCertificateTemplate() x509.Certificate {
	keyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	commonName := "test.example.com"
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"SK"},
			Country:      []string{"CA"},
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * 365 * time.Hour),

		SignatureAlgorithm: x509.ECDSAWithSHA256,

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageCertSign,

		ExtKeyUsage: keyUsage,

		BasicConstraintsValid: true,
		IsCA: true,

		DNSNames: []string{"test.example.com"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1).To4(),
			net.IPv4(127, 17, 0, 1).To4(),
			net.IPv4(127, 17, 0, 2).To4(),
			net.IPv4(127, 17, 0, 3).To4(),
			net.ParseIP("2001:4860:0:2001::68")},

		PolicyIdentifiers:   []asn1.ObjectIdentifier{[]int{1, 2, 3}},
		PermittedDNSDomains: []string{".example.com", "example.com"},
	}
	return template
}
