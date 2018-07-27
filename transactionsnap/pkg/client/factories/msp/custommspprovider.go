/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/msppvdr"
	"github.com/securekey/fabric-snaps/util/errors"
)

// CustomMSPProvider  will provide custom msp provider
type CustomMSPProvider struct {
	msppvdr.MSPProvider
	config         fabApi.EndpointConfig
	cryptoProvider coreApi.CryptoSuite
	cryptoPath     string
}

// IdentityManager returns the organization's identity manager
func (p *CustomMSPProvider) IdentityManager(orgName string) (mspApi.IdentityManager, bool) {
	customIdenMgr, err := NewCustomIdentityManager(orgName, p.cryptoProvider, p.config, p.cryptoPath)
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, "failed to create NewCustomIdentityManager").GenerateLogMsg())
		return nil, false
	}
	return customIdenMgr, true
}
