/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package factories

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defcore"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/fabpvdr"

	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/securekey/fabric-snaps/util/errors"
)

// CustomCorePkg is will provide custom sdk core pkg
type CustomCorePkg struct {
	defcore.ProviderFactory
	ProviderName string
	CryptoPath   string
}

// CreateCryptoSuiteProvider returns a implementation of factory default bccsp cryptosuite
func (f *CustomCorePkg) CreateCryptoSuiteProvider(config core.Config) (core.CryptoSuite, error) {
	bccspSuite, err := factory.GetBCCSP(f.ProviderName)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error creating new cryptosuite provider")
	}
	return GetSuite(bccspSuite), nil
}

// CreateIdentityManager return new identity manager
func (f *CustomCorePkg) CreateIdentityManager(orgName string, stateStore core.KVStore, cryptoProvider core.CryptoSuite, config core.Config) (core.IdentityManager, error) {
	customIdenMgr, err := NewCustomIdentityManager(orgName, stateStore, cryptoProvider, config, f.CryptoPath)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "failed to create new credential manager")
	}
	return customIdenMgr, nil
}

// CreateInfraProvider returns a new custom implementation of fabric primitives
func (f *CustomCorePkg) CreateInfraProvider(config core.Config) (fab.InfraProvider, error) {
	return &CustomInfraProvider{InfraProvider: fabpvdr.New(config)}, nil
}
