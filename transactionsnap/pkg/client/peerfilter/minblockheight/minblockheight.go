/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	peer "github.com/hyperledger/fabric/core/peer"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/securekey/fabric-snaps/membershipsnap/api"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

// New creates a new Min Block Height peer filter. This filter
// selects peers whose block height is at least the height
// of the local peer on which the TxSnap is being invoked.
func New(args []string) (transactionsnapApi.PeerFilter, error) {
	return newWithOpts(args, &peerBlockchainInfoProvider{})
}

func newWithOpts(args []string, bcInfoProvider blockchainInfoProvider) (*peerFilter, error) {
	if len(args) == 0 {
		return nil, errors.New(errors.GeneralError, "expecting channel ID arg")
	}
	return &peerFilter{
		channelID:      args[0],
		bcInfoProvider: bcInfoProvider,
	}, nil
}

// blockchainInfoProvider provides block chain info for a given channel
type blockchainInfoProvider interface {
	GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error)
}

type peerFilter struct {
	channelID      string
	bcInfoProvider blockchainInfoProvider
}

// Accept returns true if the given peer's block height is
// at least the height of the local peer.
func (f *peerFilter) Accept(p fabApi.Peer) bool {
	logger.Debugf("minblockheight check if peer of type channel peer")
	chanPeer, ok := p.(api.ChannelPeer)
	if !ok {
		// This shouldn't happen since all peers should implement ChannelPeer
		logger.Errorf("Peer is not a ChannelPeer")
		return false
	}

	logger.Debugf("minblockheight GetBlockchainInfo for channel %s", f.channelID)

	bcInfo, err := f.bcInfoProvider.GetBlockchainInfo(f.channelID)

	var height uint64
	if err != nil {
		logger.Errorf("Error getting ledger height for channel [%s] on local peer: %s.\n", f.channelID, err)
	} else {
		// Need to subtract 1 from the block height since the block height (LedgerHeight) that's included
		// in the Gossip Network Member is really the block number (i.e. they subtract 1 also)
		height = bcInfo.Height - 1
	}

	logger.Debugf("minblockheight GetBlockHeight for channel %s", f.channelID)

	peerHeight := chanPeer.GetBlockHeight(f.channelID)
	accepted := peerHeight >= height
	if !accepted {
		logger.Debugf("Peer [%s] will NOT be accepted since its block height for channel [%s] is %d which is less than or equal to that of the local peer: %d.\n", chanPeer.URL(), f.channelID, peerHeight, height)
	} else {
		logger.Debugf("Peer [%s] will be accepted since its block height for channel [%s] is %d which is greater than or equal to that of the local peer: %d.\n", chanPeer.URL(), f.channelID, peerHeight, height)
	}

	return accepted
}

type peerBlockchainInfoProvider struct {
	bcInfo map[string]*cb.BlockchainInfo
}

// GetBlockchainInfo delegates to the peer to return basic info about the blockchain
func (l *peerBlockchainInfoProvider) GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error) {
	return peer.GetLedger(channelID).GetBlockchainInfo()
}
