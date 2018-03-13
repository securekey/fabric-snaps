/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/msppvdr"
)

// CustomMSPProvider  will provide custom msp provider
type CustomMSPProvider struct {
	msppvdr.MSPProvider
	config         core.Config
	cryptoProvider core.CryptoSuite
	cryptoPath     string
}

// IdentityManager returns the organization's identity manager
func (p *CustomMSPProvider) IdentityManager(orgName string) (mspApi.IdentityManager, bool) {
	customIdenMgr, err := NewCustomIdentityManager(orgName, p.cryptoProvider, p.config, p.cryptoPath)
	if err != nil {
		logger.Errorf("NewCustomIdentityManager return error %v", err)
		return nil, false
	}
	return customIdenMgr, true
}
