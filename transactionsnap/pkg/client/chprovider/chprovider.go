/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chprovider

import (
	reqContext "context"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/dynamicselection"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	channelImpl "github.com/hyperledger/fabric-sdk-go/pkg/fab/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/channel/membership"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/chconfig"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/provider/chpvdr"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	"github.com/pkg/errors"
	dynamicDiscovery "github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
)

var logger = logging.NewLogger("txnsnap")

type cache interface {
	Get(lazycache.Key, ...interface{}) (interface{}, error)
	Close()
}

// Provider implements a ChannelProvider that uses a dynamic discovery provider based on
// the local Membership Snap, dynamic selection provider, and the local Event Snap
type Provider struct {
	providerContext       context.Providers
	discoveryServiceCache cache
	selectionServiceCache cache
	chCfgCache            cache
	membershipCache       cache
	eventServiceCache     cache
}

// New creates a new Provider
func New(config fab.EndpointConfig) (*Provider, error) {
	eventIdleTime := config.Timeout(fab.EventServiceIdle)
	chConfigRefresh := config.Timeout(fab.ChannelConfigRefresh)
	membershipRefresh := config.Timeout(fab.ChannelMembershipRefresh)
	cp := Provider{
		chCfgCache:      chconfig.NewRefCache(chConfigRefresh),
		membershipCache: membership.NewRefCache(membershipRefresh),
	}

	cp.discoveryServiceCache = lazycache.New(
		"TxSnap_Discovery_Service_Cache",
		func(key lazycache.Key) (interface{}, error) {
			ck := key.(*cacheKey)
			return cp.createDiscoveryService(ck.context, ck.channelID)
		},
	)

	cp.selectionServiceCache = lazycache.New(
		"TxSnap_Selection_Service_Cache",
		func(key lazycache.Key) (interface{}, error) {
			ck := key.(*cacheKey)
			return cp.createSelectionService(ck.context, ck.channelID)
		},
	)

	cp.eventServiceCache = lazycache.New(
		"TxSnap_Event_Service_Cache",
		func(key lazycache.Key) (interface{}, error) {
			ck := key.(*eventCacheKey)
			return chpvdr.NewEventClientRef(
				eventIdleTime,
				func() (fab.EventClient, error) {
					return cp.createEventClient(ck.context, ck.channelConfig, ck.opts...)
				},
			), nil
		},
	)

	return &cp, nil
}

// Initialize sets the provider context
func (cp *Provider) Initialize(providers context.Providers) error {
	cp.providerContext = providers
	return nil
}

// Close frees resources and caches.
func (cp *Provider) Close() {
	logger.Debug("Closing event service cache...")
	cp.eventServiceCache.Close()

	logger.Debug("Closing membership cache...")
	cp.membershipCache.Close()

	logger.Debug("Closing channel configuration cache...")
	cp.chCfgCache.Close()

	logger.Debug("Closing selection service cache...")
	cp.selectionServiceCache.Close()

	logger.Debug("Closing discovery service cache...")
	cp.discoveryServiceCache.Close()
}

// ChannelService creates a ChannelService for an identity
func (cp *Provider) ChannelService(ctx fab.ClientContext, channelID string) (fab.ChannelService, error) {
	return &ChannelService{
		provider:  cp,
		context:   ctx,
		channelID: channelID,
	}, nil
}

func (cp *Provider) createEventClient(ctx context.Client, chConfig fab.ChannelCfg, opts ...options.Opt) (fab.EventClient, error) {
	discovery, err := cp.getDiscoveryService(ctx, chConfig.ID())
	if err != nil {
		return nil, errors.WithMessage(err, "could not get discovery service")
	}
	logger.Debugf("Using deliver events for channel [%s]", chConfig.ID())
	return deliverclient.New(ctx, chConfig, discovery, opts...)
}

func (cp *Provider) createDiscoveryService(ctx context.Client, channelID string) (fab.DiscoveryService, error) {
	logger.Debugf("Creating discovery service for channel [%s]", channelID)
	service, err := memservice.Get()
	if err != nil {
		return nil, errors.WithMessage(err, "error getting membership service")
	}
	return dynamicDiscovery.New(channelID, ctx.EndpointConfig(), service), nil
}

func (cp *Provider) getDiscoveryService(context fab.ClientContext, channelID string) (fab.DiscoveryService, error) {
	key, err := newCacheKey(context, channelID)
	if err != nil {
		return nil, err
	}
	discoveryService, err := cp.discoveryServiceCache.Get(key)
	if err != nil {
		return nil, err
	}
	return discoveryService.(fab.DiscoveryService), nil
}

func (cp *Provider) createSelectionService(ctx context.Client, channelID string) (fab.SelectionService, error) {
	logger.Debugf("Creating selection service for channel [%s]", channelID)
	discovery, err := cp.getDiscoveryService(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return dynamicselection.NewService(ctx, channelID, discovery)
}

func (cp *Provider) getSelectionService(context fab.ClientContext, channelID string) (fab.SelectionService, error) {
	key, err := newCacheKey(context, channelID)
	if err != nil {
		return nil, err
	}
	selectionService, err := cp.selectionServiceCache.Get(key)
	if err != nil {
		return nil, err
	}
	return selectionService.(fab.SelectionService), nil
}

func (cp *Provider) channelConfig(context fab.ClientContext, channelID string) (fab.ChannelCfg, error) {
	if channelID == "" {
		// System channel
		return chconfig.NewChannelCfg(""), nil
	}

	chCfgRef, err := cp.loadChannelCfgRef(context, channelID)
	if err != nil {
		return nil, err
	}
	chCfg, err := chCfgRef.Get()
	if err != nil {
		return nil, errors.WithMessage(err, "could not get chConfig cache reference")
	}
	return chCfg.(fab.ChannelCfg), nil
}

func (cp *Provider) loadChannelCfgRef(context fab.ClientContext, channelID string) (*chconfig.Ref, error) {
	logger.Debugf("Loading channel config ref for channel [%s]", channelID)

	key, err := chconfig.NewCacheKey(context, func(string) (fab.ChannelConfig, error) { return chconfig.New(channelID) }, channelID)
	if err != nil {
		return nil, err
	}
	c, err := cp.chCfgCache.Get(key)
	if err != nil {
		return nil, err
	}

	return c.(*chconfig.Ref), nil
}

// ChannelService provides Channel clients and maintains contexts for them.
// the identity context is used
type ChannelService struct {
	provider  *Provider
	context   context.Client
	channelID string
}

// Config returns the Config for the named channel
func (cs *ChannelService) Config() (fab.ChannelConfig, error) {
	return chconfig.New(cs.channelID)
}

// EventService returns the local Event Service.
func (cs *ChannelService) EventService(opts ...options.Opt) (fab.EventService, error) {
	chnlCfg, err := cs.ChannelConfig()
	if err != nil {
		return nil, err
	}
	key, err := newEventCacheKey(cs.context, chnlCfg, opts...)
	if err != nil {
		return nil, err
	}
	eventService, err := cs.provider.eventServiceCache.Get(key)
	if err != nil {
		return nil, err
	}
	return eventService.(fab.EventService), nil
}

// Membership returns and caches a channel member identifier
// A membership reference is returned that refreshes with the configured interval
func (cs *ChannelService) Membership() (fab.ChannelMembership, error) {
	chCfgRef, err := cs.loadChannelCfgRef()
	if err != nil {
		return nil, err
	}
	key, err := membership.NewCacheKey(membership.Context{Providers: cs.provider.providerContext, EndpointConfig: cs.context.EndpointConfig()},
		chCfgRef.Reference, cs.channelID)
	if err != nil {
		return nil, err
	}
	ref, err := cs.provider.membershipCache.Get(key)
	if err != nil {
		return nil, err
	}

	return ref.(*membership.Ref), nil
}

// ChannelConfig returns the channel config for this channel
func (cs *ChannelService) ChannelConfig() (fab.ChannelCfg, error) {
	return cs.provider.channelConfig(cs.context, cs.channelID)
}

// Transactor returns the transactor
func (cs *ChannelService) Transactor(reqCtx reqContext.Context) (fab.Transactor, error) {
	cfg, err := cs.ChannelConfig()
	if err != nil {
		return nil, err
	}
	return channelImpl.NewTransactor(reqCtx, cfg)
}

// Discovery returns a DiscoveryService for the given channel
func (cs *ChannelService) Discovery() (fab.DiscoveryService, error) {
	return cs.provider.getDiscoveryService(cs.context, cs.channelID)
}

// Selection returns a SelectionService for the given channel
func (cs *ChannelService) Selection() (fab.SelectionService, error) {
	return cs.provider.getSelectionService(cs.context, cs.channelID)
}

func (cs *ChannelService) loadChannelCfgRef() (*chconfig.Ref, error) {
	return cs.provider.loadChannelCfgRef(cs.context, cs.channelID)
}
