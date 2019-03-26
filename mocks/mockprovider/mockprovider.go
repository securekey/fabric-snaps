/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockprovider

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/staticselection"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	fcmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	discoveryService "github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

// Factory mocks out the channel provider
type Factory struct {
	defsvc.ProviderFactory
}

// CreateChannelProvider returns a new default implementation of channel provider
func (f *Factory) CreateChannelProvider(config fabApi.EndpointConfig) (fabApi.ChannelProvider, error) {
	return &mockChannelProvider{
		ChannelProvider: &mockChannelProvider{},
	}, nil
}

type mockChannelProvider struct {
	fabApi.ChannelProvider
}

func (cp *mockChannelProvider) Initialize(providers core.Providers) error {
	chProvider, err := fcmocks.NewMockChannelProvider(providers)
	if err != nil {
		return err
	}
	cp.ChannelProvider = chProvider
	return nil
}

func (cp *mockChannelProvider) ChannelService(ctx fabApi.ClientContext, channelID string) (fabApi.ChannelService, error) {
	memService := membership.NewServiceWithMocks([]byte(ctx.Identifier().MSPID), membership.NewLocalNetworkChannelMember("internalhost1:1000", 1000))
	discovery := discoveryService.New(channelID, ctx.EndpointConfig(), memService)

	selection, err := staticselection.NewService(discovery)
	if err != nil {
		return nil, err
	}

	mockChannelService := &fcmocks.MockChannelService{}
	mockChannelService.SetDiscovery(discovery)
	mockChannelService.SetSelection(selection)

	return mockChannelService, nil
}
