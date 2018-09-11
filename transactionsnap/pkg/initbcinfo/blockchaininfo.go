/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package initbcinfo

import (
	"sync"

	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/securekey/fabric-snaps/util/errors"
)

var initialBCInfoByChannel sync.Map

// Set stores the initial blockchain info for the peer when the channel is joined or on restart
func Set(channelID string, bcInfo *cb.BlockchainInfo) error {
	if _, loaded := initialBCInfoByChannel.LoadOrStore(channelID, bcInfo); loaded {
		return errors.Errorf(errors.SystemError, "initial blockchain info already set for channel [%s]", channelID)
	}
	return nil
}

// Get returns the initial blockchain info for the peer when the channel is joined or on restart
func Get(channelID string) (*cb.BlockchainInfo, bool) {
	bcInfo, ok := initialBCInfoByChannel.Load(channelID)
	if !ok {
		return nil, false
	}
	return bcInfo.(*cb.BlockchainInfo), true
}
