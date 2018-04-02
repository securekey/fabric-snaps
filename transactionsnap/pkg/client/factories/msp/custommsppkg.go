/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	defmsp "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defmsp"
)

// CustomMspPkg is will provide custom msp pkg
type CustomMspPkg struct {
	defmsp.ProviderFactory
	CryptoPath string
}

// CreateIdentityManagerProvider returns a new custom implementation of msp provider
func (m *CustomMspPkg) CreateIdentityManagerProvider(config fabApi.EndpointConfig, cryptoProvider coreApi.CryptoSuite, userStore mspApi.UserStore) (msp.IdentityManagerProvider, error) {
	return &CustomMSPProvider{config: config, cryptoProvider: cryptoProvider, cryptoPath: m.CryptoPath}, nil
}
