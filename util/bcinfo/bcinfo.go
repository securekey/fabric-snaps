/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bcinfo

import (
	"github.com/hyperledger/fabric/core/peer"
	cb "github.com/hyperledger/fabric/protos/common"
)

// Provider is a Blockchain Info Provider
type Provider struct {
	bcInfo map[string]*cb.BlockchainInfo
}

// NewProvider returns a new Blockchain Info Provider
func NewProvider() *Provider {
	return &Provider{}
}

// GetBlockchainInfo delegates to the peer to return basic info about the blockchain
func (l *Provider) GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error) {
	return peer.GetLedger(channelID).GetBlockchainInfo()
}
