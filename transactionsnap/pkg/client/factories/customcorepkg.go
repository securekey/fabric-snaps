/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defcore"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/fabpvdr"
	"github.com/hyperledger/fabric/bccsp/factory"
)

// CustomCorePkg is will provide custom sdk core pkg
type CustomCorePkg struct {
	defcore.ProviderFactory
	ProviderName string
}

// CreateCryptoSuiteProvider returns a implementation of factory default bccsp cryptosuite
func (f *CustomCorePkg) CreateCryptoSuiteProvider(config core.CryptoSuiteConfig) (core.CryptoSuite, error) {
	return GetSuite(factory.GetDefault()), nil
}

// CreateInfraProvider returns a new custom implementation of fabric primitives
func (f *CustomCorePkg) CreateInfraProvider(config fab.EndpointConfig) (fab.InfraProvider, error) {
	return &CustomInfraProvider{InfraProvider: fabpvdr.New(config)}, nil
}
