/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chprovider

import (
	reqContext "context"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/fabricselection"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	channelImpl "github.com/hyperledger/fabric-sdk-go/pkg/fab/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/channel/membership"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/chconfig"
	evtclient "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client"
	evtclientdisp "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client/dispatcher"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client/peerresolver/preferpeer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient/seek"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/dispatcher"
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

type params struct {
	localPeerURL    string
	initialBlockNum uint64
	eventSnapshots  map[string]fab.EventSnapshot
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

// Opt is a provider option
type Opt func(*params)

// WithEventSnapshots initializes the event service with the given event snapshots
func WithEventSnapshots(snapshots map[string]fab.EventSnapshot) Opt {
	return func(p *params) {
		p.eventSnapshots = snapshots
	}
}

// WithInitialBlockNum initializes the event service with the given event snapshots
func WithInitialBlockNum(blockNum uint64) Opt {
	return func(p *params) {
		p.initialBlockNum = blockNum
	}
}

// WithLocalPeerURL sets the URL of the local peer
func WithLocalPeerURL(url string) Opt {
	return func(p *params) {
		p.localPeerURL = url
	}
}

// New creates a new Provider
func New(config fab.EndpointConfig, opts ...Opt) (*Provider, error) {
	chConfigRefresh := config.Timeout(fab.ChannelConfigRefresh)
	membershipRefresh := config.Timeout(fab.ChannelMembershipRefresh)
	cp := Provider{
		chCfgCache:      chconfig.NewRefCache(chConfigRefresh),
		membershipCache: membership.NewRefCache(membershipRefresh),
	}

	// Apply options
	params := &params{eventSnapshots: make(map[string]fab.EventSnapshot)}
	for _, opt := range opts {
		opt(params)
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
			ck := key.(*cacheKey)
			return cp.newEventClientRef(params, ck.context, ck.channelConfig), nil
		},
	)

	return &cp, nil
}

func (cp *Provider) newEventClientRef(params *params, ctx fab.ClientContext, chConfig fab.ChannelCfg) *EventClientRef {
	preInitialize := false

	var opts []options.Opt

	// Keep retrying to connect to the event client forever
	opts = append(opts, evtclient.WithMaxConnectAttempts(0))

	if params.localPeerURL != "" {
		// Connect to the local peer if not too far behind in block height
		opts = append(opts, evtclientdisp.WithPeerResolver(preferpeer.NewResolver(params.localPeerURL)))
	}

	if snapshot, ok := params.eventSnapshots[chConfig.ID()]; ok {
		logger.Infof("Creating event client with snapshot for channel [%s]: %s", chConfig.ID(), snapshot)
		opts = append(opts, dispatcher.WithSnapshot(snapshot))

		// Must initialize the event client right away since there will be outstanding
		// registrations that are waiting for events
		preInitialize = true
	} else if params.initialBlockNum > 0 {
		logger.Debugf("Asking deliver client for all blocks from block %d for channel [%s]", params.initialBlockNum, chConfig.ID())
		opts = append(opts, deliverclient.WithSeekType(seek.FromBlock))
		opts = append(opts, deliverclient.WithBlockNum(params.initialBlockNum))
	}

	logger.Infof("Creating new event service ref for channel [%s]", chConfig.ID())

	ref := NewEventClientRef(
		func() (fab.EventClient, error) {
			return cp.createEventClient(ctx, chConfig, opts...)
		},
	)

	if preInitialize {
		go func() {
			// The membership cache needs to be pre-populated since we'll be connecting to other peers.
			logger.Debugf("Initializing membership cache for channel [%s]", chConfig.ID())
			err := cp.initMembership(chConfig.ID(), ctx)
			if err != nil {
				logger.Warnf("Error occurred while initializing membership cache for channel [%s]", chConfig.ID(), err)
			}

			logger.Debugf("Initializing event service for channel [%s]", chConfig.ID())
			ref.get() //nolint:gas
		}()

	}

	return ref
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
	logger.Debugf("Creating new deliver event client for channel [%s]", chConfig.ID())
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

	return fabricselection.New(ctx, channelID, discovery)
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

// EventService returns the Event Service.
func (cs *ChannelService) EventService(opts ...options.Opt) (fab.EventService, error) {
	chnlCfg, err := cs.ChannelConfig()
	if err != nil {
		return nil, err
	}

	eventService, err := cs.provider.eventServiceCache.Get(newEventCacheKey(cs.context, chnlCfg))
	if err != nil {
		return nil, err
	}
	return eventService.(fab.EventService), nil
}

// TransferEventRegistrations transfers all event registrations into the returned snapshot
func (cs *ChannelService) TransferEventRegistrations() (fab.EventSnapshot, error) {
	eventService, err := cs.EventService()
	if err != nil {
		return nil, err
	}

	eventRef := eventService.(*EventClientRef)
	service, err := eventRef.get()
	if err != nil {
		return nil, err
	}

	return service.(fab.EventClient).TransferRegistrations(false)
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

func (cp *Provider) initMembership(channelID string, ctx fab.ClientContext) error {
	chCfgRef, err := cp.loadChannelCfgRef(ctx, channelID)
	if err != nil {
		return err
	}

	key, err := membership.NewCacheKey(
		membership.Context{
			Providers:      cp.providerContext,
			EndpointConfig: ctx.EndpointConfig(),
		},
		chCfgRef.Reference, channelID,
	)
	if err != nil {
		return err
	}

	ref, err := cp.membershipCache.Get(key)
	if err != nil {
		return err
	}

	// Invoke any function so that the ref is initialized
	ref.(*membership.Ref).ContainsMSP("")

	return nil
}
