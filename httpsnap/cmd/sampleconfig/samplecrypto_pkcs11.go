// +build pkcs11

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sampleconfig

import (
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
		Library:      "/usr/lib/softhsm/libsofthsm2.so",
		Pin:          "98765432",
		Label:        "ForFabric",
		Ephemeral:    false,
		Sensitive:    true,
		SoftVerify:   true,
	}

	return &factory.FactoryOpts{
		ProviderName: "PKCS11",
		Pkcs11Opts:   &pkcsOpt,
	}
}
