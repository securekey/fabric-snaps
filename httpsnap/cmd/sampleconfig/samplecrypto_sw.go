// +build !pkcs11

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sampleconfig

import "github.com/hyperledger/fabric/bccsp/factory"

//GetSampleBCCSPFactoryOpts returns bccsp opts for SW
func GetSampleBCCSPFactoryOpts(ksPath string) *factory.FactoryOpts {
	return &factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			FileKeystore: &factory.FileKeystoreOpts{KeyStorePath: ksPath + "/msp/keystore"},
		},
	}
}

//ResolvPeerConfig returns peer config file updated based on build flag
func ResolvPeerConfig(peerConfigPath string) string {
	//do nothing
	return peerConfigPath
}
