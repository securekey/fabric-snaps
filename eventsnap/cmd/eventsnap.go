/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	pb "github.com/hyperledger/fabric/protos/peer"
	eventrelay "github.com/securekey/fabric-snaps/eventrelay/pkg/relay"
	eventserverapi "github.com/securekey/fabric-snaps/eventserver/api"
	eventserver "github.com/securekey/fabric-snaps/eventserver/pkg/server"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/config"

	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
	localservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	eventservice "github.com/securekey/fabric-snaps/eventservice/pkg/service"
)

var logger = flogging.MustGetLogger("eventSnap")

const (
	channelConfigCheckDuration = 1 * time.Second
)

var chnlServer *eventserver.ChannelServer
var mutex sync.RWMutex

type configProvider interface {
	GetConfig(channelID string) (*config.EventSnapConfig, error)
}

// eventSnap starts the Channel Event Server which allows clients to register
// for channel events. It also registers a local event service on the peer so that other
// snaps may register for channel events directly.
type eventSnap struct {
	// pserver is only set during unit testing
	pserver *grpc.Server
	// eropts is only set during unit testing
	eropts *eventrelay.Opts
	// config is only set during unit testing
	configProvider configProvider
}

type cfgProvider struct {
}

// New returns a new Event Snap
func New() shim.Chaincode {
	return &eventSnap{configProvider: &cfgProvider{}}
}

func (cfgprovider *cfgProvider) GetConfig(channelID string) (*config.EventSnapConfig, error) {

	esconfig, err := config.New(channelID, "")
	if err != nil {
		logger.Warningf("Error initializing event snap: %s\n", err)
		return nil, errors.Wrap(err, "error initializing event snap ")
	}
	return esconfig, nil
}

// Init initializes the Event Snap.
// The Event Server is registered when Init is called without a channel and
// a new, channel-specific event service is registered each time Init is called with a channel.
func (s *eventSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Warningf("******** Init Event Snap on channel [%s]\n", stub.GetChannelID())

	channelID := stub.GetChannelID()

	esconfig, err := s.configProvider.GetConfig(channelID)
	if err != nil {
		return shim.Error(err.Error())
	}

	if channelID == "" {
		// The channel server must be started on the first call to Init with no channel ID,
		// since it needs to register with the peer server before the peer GRPC server starts
		// serving requests.
		if err := s.startChannelServer(esconfig); err != nil {
			logger.Error(err.Error())
			return shim.Error(err.Error())
		}
	} else {
		if esconfig.ChannelConfigLoaded {
			if err := s.startChannelEvents(stub.GetChannelID(), esconfig); err != nil {
				logger.Error(err.Error())
				return shim.Error(err.Error())
			}
		} else {
			// Check the config periodically and start
			// the event service when the config is available.
			logger.Warningf("EventSnap configuration is unavailable for channel [%s]. The event service will be started when configuration is available.\n", stub.GetChannelID())
			go s.delayStartChannelEvents(stub.GetChannelID())
		}
	}

	return shim.Success(nil)
}

// Invoke isn't implemented for this snap.
func (s *eventSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("not implemented")
}

// startChannelServer registers the Channel Event Server endpoint on the peer.
func (s *eventSnap) startChannelServer(config *config.EventSnapConfig) error {
	mutex.Lock()
	defer mutex.Unlock()

	if chnlServer != nil {
		logger.Infof("Channel Event Server already initialized\n")
		return nil
	}

	logger.Infof("Initializing Channel Event Server...\n")

	csConfig := &eventserver.ChannelServerConfig{
		BufferSize: config.EventServerBufferSize,
		Timeout:    config.EventServerTimeout,
		TimeWindow: config.EventServerTimeWindow,
	}
	chnlServer = eventserver.NewChannelServer(csConfig)
	eventserverapi.RegisterChannelServer(s.peerServer(), chnlServer)
	logger.Infof("... done initializing Channel Event Server.\n")

	return nil
}

// startChannelEvents starts a new event relay and local event service for the given channel,
// and also starts a new Go routine that relays events to the channel server.
func (s *eventSnap) startChannelEvents(channelID string, config *config.EventSnapConfig) error {
	existingLocalEventService := localservice.Get(channelID)
	if existingLocalEventService != nil {
		logger.Errorf("Event service already initialized for channel [%s]\n", channelID)
		return errors.Errorf("Event service already initialized for channel [%s]\n", channelID)
	}

	// Create an event relay which gets events from the event hub
	eventRelay, err := s.startEventRelay(channelID, config)
	if err != nil {
		logger.Errorf("Error starting event relay: %s\n", err)
		return errors.Errorf("Error starting event relay: %s\n", err)
	}

	// Create a new channel event service which gets its events from the event relay
	service := s.startEventService(eventRelay, config)

	// Register the local event service for the channel
	if err := localservice.Register(channelID, service); err != nil {
		logger.Errorf("Error registering local event service: %s\n", err)
		return errors.Errorf("Error registering local event service: %s\n", err)
	}

	// Relay events to the channel event server
	s.startChannelServerEventRelay(eventRelay, config)

	return nil
}
func (s *eventSnap) delayStartChannelEvents(channelID string) {
	for {
		time.Sleep(channelConfigCheckDuration)

		logger.Infof("Checking if EventSnap configuration is available for channel [%s]...\n", channelID)
		if config, err := config.New(channelID, ""); err != nil {
			logger.Warningf("Error reading configuration: %s\n", err)
		} else if config.ChannelConfigLoaded {
			if err := s.startChannelEvents(channelID, config); err != nil {
				logger.Errorf("Error starting channel events for channel [%s]: %s. Aborting!!!\n", channelID, err.Error())
			} else {
				logger.Infof("Channel events successfully started for channel [%s].\n", channelID)
			}
			return
		}
		logger.Infof("... EventSnap configuration is not available yet for channel [%s]\n", channelID)
	}
}

// startEventRelay starts an event relay for the given channel. The event relay
// registers for block and filtered block events with the event hub and relays
// the events to all registered clients.
func (s *eventSnap) startEventRelay(channelID string, config *config.EventSnapConfig) (*eventrelay.EventRelay, error) {
	logger.Infof("Starting event relay for channel [%s]...\n", channelID)

	var opts *eventrelay.Opts
	if s.eropts != nil {
		opts = s.eropts
	} else {
		opts = eventrelay.DefaultOpts()
		opts.RegTimeout = config.EventHubRegTimeout
		opts.RelayTimeout = config.EventRelayTimeout
	}

	eventRelay, err := eventrelay.New(channelID, config.EventHubAddress, config.TransportCredentials, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating event relay")
	}

	eventRelay.Start()
	logger.Infof("... started event relay for channel [%s]...\n", channelID)
	return eventRelay, nil
}

// startEventService registers an Event Service which receives events from the given EventRelay.
func (s *eventSnap) startEventService(eventRelay *eventrelay.EventRelay, config *config.EventSnapConfig) eventapi.EventService {
	logger.Infof("Starting event service for channel [%s]...\n", eventRelay.ChannelID())

	opts := eventservice.Opts{
		EventConsumerBufferSize: config.EventConsumerBufferSize,
		EventConsumerTimeout:    config.EventConsumerTimeout,
	}
	eventTypes := []eventservice.EventType{eventservice.BLOCKEVENT, eventservice.FILTEREDBLOCKEVENT}

	service := eventservice.NewServiceWithDispatcher(
		eventservice.NewDispatcher(
			&eventservice.DispatcherOpts{
				Opts:                 opts,
				AuthorizedEventTypes: eventTypes,
			},
		),
		&opts, eventTypes,
	)
	service.Start(eventRelay)
	logger.Infof("... started event service for channel [%s]...\n", eventRelay.ChannelID())
	return service
}

// startChannelServerEventRelay starts a Go routine that relays the events
// from the given eventRelay to the Channel Event Server
func (s *eventSnap) startChannelServerEventRelay(eventRelay *eventrelay.EventRelay, config *config.EventSnapConfig) {
	eventch := make(chan interface{}, config.EventServerBufferSize)
	eventRelay.Register(eventch)

	go func() {
		logger.Debugf("Listening for events from the event relay.\n")
		for {
			event, ok := <-eventch
			if !ok {
				logger.Warningf("Event channel closed.\n")
				return
			}

			evt, ok := event.(*pb.Event)
			if ok {
				go func() {
					logger.Debugf("Sending event to channel event server: %s.\n", event)
					if err := channelServer().Send(evt); err != nil {
						logger.Errorf("Error sending event to channel server: %s\n", err)
					}
				}()
			} else {
				logger.Warningf("Unsupported event type: %s\n", reflect.TypeOf(event))
			}
		}
	}()
}

func (s *eventSnap) peerServer() *grpc.Server {
	if s.pserver != nil {
		return s.pserver
	}
	return peer.GetPeerServer().Server()
}

func channelServer() *eventserver.ChannelServer {
	mutex.RLock()
	defer mutex.RUnlock()
	return chnlServer
}

func main() {
}
