/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"fmt"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/staticselection"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

// MockProviderFactory event out the CHannel Provider
type MockProviderFactory struct {
	defsvc.ProviderFactory
	EventService fabApi.EventService
}

// CreateChannelProvider creates a mock ChannelProvider
func (f *MockProviderFactory) CreateChannelProvider(config fabApi.EndpointConfig) (fabApi.ChannelProvider, error) {
	provider, err := f.ProviderFactory.CreateChannelProvider(config)
	if err != nil {
		return nil, err
	}
	return &mockChannelProvider{
		ChannelProvider: provider,
		eventService:    f.EventService,
	}, nil
}

type mockChannelProvider struct {
	fabApi.ChannelProvider
	eventService fabApi.EventService
}

type providerInit interface {
	Initialize(providers contextApi.Providers) error
}

func (cp *mockChannelProvider) Initialize(providers contextApi.Providers) error {
	if pi, ok := cp.ChannelProvider.(providerInit); ok {
		err := pi.Initialize(providers)
		if err != nil {
			return fmt.Errorf("failed to initialize channel provider: %s", err)
		}
	}
	return nil
}

func (cp *mockChannelProvider) ChannelService(ctx fabApi.ClientContext, channelID string) (fabApi.ChannelService, error) {
	chService, err := cp.ChannelProvider.ChannelService(ctx, channelID)
	if err != nil {
		return nil, err
	}

	memService := membership.NewServiceWithMocks(
		[]byte(ctx.Identifier().MSPID),
		membership.NewLocalNetworkChannelMember("internalhost1:1000", 0),
	)

	discovery := service.New(channelID, ctx.EndpointConfig(), memService)
	selection, err := staticselection.NewService(discovery)
	if err != nil {
		return nil, err
	}

	return &mockChannelService{
		ChannelService: chService,
		discovery:      discovery,
		selection:      selection,
		eventService:   cp.eventService,
	}, nil
}

type mockChannelService struct {
	fabApi.ChannelService
	discovery    fabApi.DiscoveryService
	selection    fabApi.SelectionService
	eventService fabApi.EventService
}

func (cs *mockChannelService) Discovery() (fabApi.DiscoveryService, error) {
	return cs.discovery, nil
}

func (cs *mockChannelService) Selection() (fabApi.SelectionService, error) {
	return cs.selection, nil
}

// EventService returns the local Event Service.
func (cs *mockChannelService) EventService(opts ...options.Opt) (fabApi.EventService, error) {
	return cs.eventService, nil
}
