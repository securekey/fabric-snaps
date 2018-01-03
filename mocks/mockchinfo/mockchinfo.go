/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockchinfo

import (
	pb "github.com/hyperledger/fabric/protos/peer"
)

// MockChannelsInfoProvider provides mock ChannelInfo
type MockChannelsInfoProvider struct {
	channelInfo []*pb.ChannelInfo
}

// NewProvider returns a new MockChannelsInfoProvider
func NewProvider(channelIDs ...string) *MockChannelsInfoProvider {
	var channelInfo []*pb.ChannelInfo
	for _, chID := range channelIDs {
		channelInfo = append(channelInfo, &pb.ChannelInfo{ChannelId: chID})
	}
	return &MockChannelsInfoProvider{channelInfo: channelInfo}
}

// GetChannelsInfo returns ChannelInfo for the given channel
func (p *MockChannelsInfoProvider) GetChannelsInfo() []*pb.ChannelInfo {
	return p.channelInfo
}
