/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defmsp"
)

// CustomMspPkg is will provide custom msp pkg
type CustomMspPkg struct {
	defmsp.ProviderFactory
	CryptoPath string
}

// CreateProvider returns a new custom implementation of msp provider
func (m *CustomMspPkg) CreateProvider(config core.Config, cryptoProvider core.CryptoSuite, userStore mspApi.UserStore) (mspApi.Provider, error) {
	return &CustomMSPProvider{config: config, cryptoProvider: cryptoProvider, cryptoPath: m.CryptoPath}, nil
}
