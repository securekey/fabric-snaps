/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	"errors"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	peer "github.com/hyperledger/fabric/core/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("transaction-snap/peerfilter/minblockheight")

// New creates a new Min Block Height peer filter. This filter
// selects peers whose block height is at least the height
// of the local peer on which the TxSnap is being invoked.
func New(args []string) (api.PeerFilter, error) {
	if len(args) == 0 {
		return nil, errors.New("expecting channel ID arg")
	}
	return &peerFilter{channelID: args[0]}, nil
}

type peerFilter struct {
	channelID string
}

// Accept returns true if the given peer's block height is
// at least the height of the local peer.
func (f *peerFilter) Accept(p sdkApi.Peer) bool {
	chanPeer, ok := p.(api.ChannelPeer)
	if !ok {
		// This shouldn't happen since all peers should implement ChannelPeer
		logger.Errorf("Peer is not a ChannelPeer")
		return false
	}

	ledger := peer.GetLedger(f.channelID)
	bcInfo, err := ledger.GetBlockchainInfo()

	var height uint64
	if err != nil {
		logger.Errorf("Error getting ledger height for channel [%s] on local peer: %s.\n", f.channelID, err)
	} else {
		height = bcInfo.Height - 1
	}

	peerHeight := chanPeer.GetBlockHeight(f.channelID)
	accepted := peerHeight >= height
	if !accepted {
		logger.Debugf("Peer [%s] will NOT be accepted since its block height for channel [%s] is %d which is less than or equal to that of the local peer: %d.\n", chanPeer.URL(), f.channelID, peerHeight, height)
	} else {
		logger.Debugf("Peer [%s] will be accepted since its block height for channel [%s] is %d which is greater than or equal to that of the local peer: %d.\n", chanPeer.URL(), f.channelID, peerHeight, height)
	}

	return accepted
}
