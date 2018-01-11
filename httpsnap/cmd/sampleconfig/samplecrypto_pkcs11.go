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

	//TODO KeyStorePath should point to an empty directory in future to make sure that lookup by SKI is picking up EC keys
	// currently KeyStorePath points to a directory where test keys are already placed and PKCS11 'find key by SKI' is
	// falling back to internal SW implementation which uses this SW keystore
	//Note: this keystore is not used by HSM, it is only used by PKCS11 internal SW CSP

	pkks := pkcs11.FileKeystoreOpts{KeyStorePath: ksPath}
	pkcsOpt := pkcs11.PKCS11Opts{
		SecLevel:     256,
		HashFamily:   "SHA2",
		FileKeystore: &pkks,
		Library:      "/usr/lib/softhsm/libsofthsm2.so",
		Pin:          "98765432",
		//TODO: use Label 'ForFabric' slot once it is available in image
		Label:     "SkLogs",
		Ephemeral: false,
	}

	return &factory.FactoryOpts{
		ProviderName: "PKCS11",
		Pkcs11Opts:   &pkcsOpt,
	}
}
