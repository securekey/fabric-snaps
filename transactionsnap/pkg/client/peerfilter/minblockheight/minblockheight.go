/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	"strconv"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/securekey/fabric-snaps/membershipsnap/api"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// New creates a new Min Block Height peer filter. This filter
// selects peers whose block height is at least that of the
// provided value.
// - arg[0] - Channel ID
// - arg[1] - Minimum block height
func New(args []string) (transactionsnapApi.PeerFilter, error) {
	if len(args) < 2 {
		return nil, errors.New(errors.SystemError, "expecting channel ID and block height args")
	}

	height, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "invalid block height arg "+args[1])
	}

	return &peerFilter{
		channelID: args[0],
		height:    height,
	}, nil
}

type peerFilter struct {
	channelID string
	height    uint64
}

// Accept returns true if the given peer's block height is
// at least the height of the local peer.
func (f *peerFilter) Accept(p fabApi.Peer) bool {
	logger.Debug("minblockheight check if peer of type channel peer")
	chanPeer, ok := p.(api.ChannelPeer)
	if !ok {
		// This shouldn't happen since all peers should implement ChannelPeer
		logger.Error(errors.New(errors.SystemError, "Peer is not a ChannelPeer").GenerateLogMsg())
		return false
	}

	peerHeight := chanPeer.GetBlockHeight(f.channelID)
	accepted := peerHeight >= f.height
	if !accepted {
		logger.Debugf("Peer [%s] will NOT be accepted since its block height for channel [%s] is %d which is less than or equal to that of the local peer: %d.", chanPeer.URL(), f.channelID, peerHeight, f.height)
	} else {
		logger.Debugf("Peer [%s] will be accepted since its block height for channel [%s] is %d which is greater than or equal to that of the local peer: %d.", chanPeer.URL(), f.channelID, peerHeight, f.height)
	}

	return accepted
}
