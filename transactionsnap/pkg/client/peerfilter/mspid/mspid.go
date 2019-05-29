/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mspid

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// New creates a new msp peer filter
// - arg[0] - Msp ID
func New(args []string) (transactionsnapApi.PeerFilter, error) {
	if len(args) < 1 {
		return nil, errors.New(errors.SystemError, "expecting msp ID")
	}
	return &peerFilter{
		mspID: args[0],
	}, nil
}

type peerFilter struct {
	mspID string
}

// Accept returns true if the given peer's msp id is
// equal specific msp id.
func (f *peerFilter) Accept(p fabApi.Peer) bool {
	accepted := p.MSPID() == f.mspID
	if !accepted {
		logger.Debugf("Peer [%s] will NOT be accepted since its msp id %s not equal %s", p.MSPID(), f.mspID)
	} else {
		logger.Debugf("Peer [%s] will be accepted is msp id %s equal %s", p.MSPID(), f.mspID)
	}

	return accepted
}
