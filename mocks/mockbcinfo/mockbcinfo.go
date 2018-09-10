/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockbcinfo

import (
	cb "github.com/hyperledger/fabric/protos/common"
)

// MockBlockchainInfoProvider provides mock BlockchainInfo
type MockBlockchainInfoProvider struct {
	bcInfo map[string]*cb.BlockchainInfo
}

// ChannelBCInfo contains a BlockchainInfo for a given channel
type ChannelBCInfo struct {
	ChannelID string
	BCInfo    *cb.BlockchainInfo
}

// NewChannelBCInfo returns a new ChannelBCInfo
func NewChannelBCInfo(channelID string, bcInfo *cb.BlockchainInfo) *ChannelBCInfo {
	return &ChannelBCInfo{ChannelID: channelID, BCInfo: bcInfo}
}

// NewProvider returns a new MockBlockchainInfoProvider
func NewProvider(bcInfo ...*ChannelBCInfo) *MockBlockchainInfoProvider {
	bcInfoMap := make(map[string]*cb.BlockchainInfo)
	for _, info := range bcInfo {
		bcInfoMap[info.ChannelID] = info.BCInfo
	}
	return &MockBlockchainInfoProvider{bcInfo: bcInfoMap}
}

// GetBlockchainInfo returns basic info about blockchain
func (l *MockBlockchainInfoProvider) GetBlockchainInfo(channelID string) (*cb.BlockchainInfo, error) {
	if channelID == "" {
		return &cb.BlockchainInfo{}, nil
	}
	return l.bcInfo[channelID], nil
}

// ChannelBCInfos returns an array of ChannelBCInfo
func ChannelBCInfos(bcInfo ...*ChannelBCInfo) []*ChannelBCInfo {
	infos := make([]*ChannelBCInfo, len(bcInfo))
	for i, info := range bcInfo {
		infos[i] = info
	}
	return infos
}

// BCInfo returns a BlockchainInfo with the given block height
func BCInfo(blockHeight uint64) *cb.BlockchainInfo {
	return &cb.BlockchainInfo{Height: blockHeight}
}
