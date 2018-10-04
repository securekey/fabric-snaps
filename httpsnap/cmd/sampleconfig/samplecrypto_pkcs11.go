// +build pkcs11

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sampleconfig

import (
	"os"
	"strings"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/pkcs11"
)

//GetSampleBCCSPFactoryOpts returns bccsp opts for PKCS11
func GetSampleBCCSPFactoryOpts(ksPath string) *factory.FactoryOpts {

	pkks := pkcs11.FileKeystoreOpts{KeyStorePath: ksPath + "/softhsm-ks"}
	pkcsOpt := pkcs11.PKCS11Opts{
		SecLevel:     256,
		HashFamily:   "SHA2",
		FileKeystore: &pkks,
		Library:      FindPKCS11Lib("/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so,/usr/lib/softhsm/libsofthsm2.so,/usr/lib/s390x-linux-gnu/softhsm/libsofthsm2.so,/usr/lib/powerpc64le-linux-gnu/softhsm/libsofthsm2.so, /usr/local/Cellar/softhsm/2.1.0/lib/softhsm/libsofthsm2.so"),
		Pin:          "98765432",
		Label:        "ForFabric",
		Ephemeral:    false,
		SoftVerify:   true,
	}

	return &factory.FactoryOpts{
		ProviderName: "PKCS11",
		Pkcs11Opts:   &pkcsOpt,
	}
}

//FindPKCS11Lib find lib based on configuration
func FindPKCS11Lib(configuredLib string) string {
	var lib string
	if configuredLib != "" {
		possibilities := strings.Split(configuredLib, ",")
		for _, path := range possibilities {
			trimpath := strings.TrimSpace(path)
			if _, err := os.Stat(trimpath); !os.IsNotExist(err) {
				lib = trimpath
				break
			}
		}
	}
	return lib
}

//ResolvPeerConfig returns peer config file updated based on build flag
func ResolvPeerConfig(peerConfigPath string) string {
	return peerConfigPath + "/core-config-pkcs11"
}
