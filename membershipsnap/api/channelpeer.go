/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
)

// ChannelPeer extends Peer and adds channel-specific information
type ChannelPeer interface {
	fabApi.Peer

	// ChannelID returns the channel ID
	ChannelID() string

	// BlockHeight returns the block height of the peer
	// for the current channel.
	BlockHeight() uint64

	// GetBlockHeight returns the block height of the peer for
	// the given channel. Returns 0 if the peer is not joined
	// to the channel or if the info is not available.
	GetBlockHeight(channelID string) uint64
}
