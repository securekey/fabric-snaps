/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package relay

import (
	"crypto/tls"
	"sync"
	"time"

	"github.com/securekey/fabric-snaps/eventserver/pkg/channelutil"
	"github.com/securekey/fabric-snaps/util/errors"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = logging.NewLogger("eventsnap")

// EventHub is an abstraction of the Event Hub
type EventHub interface {
	Start() error
}

// EventHubProvider is a function that creates a new EventHub implementation
type EventHubProvider func(channelID string, address string, regTimeout time.Duration, adapter EventAdapter, tlsConfig *tls.Config) (EventHub, error)

// Opts provides the event relay options
type Opts struct {
	// RegTimeout is the timeout when registering for events with the event hub
	RegTimeout time.Duration

	// EventHubRetryInterval is the time between retries when connecting to the event hub
	EventHubRetryInterval time.Duration

	// RelayTimeout is the timeout when relaying events to the registered event channel.
	// If < 0, if buffer full, unblocks immediately and does not send.
	// If 0, if buffer full, will block and guarantee the event will be sent out.
	// If > 0, if buffer full, blocks util timeout.
	RelayTimeout time.Duration

	// eventHubProvider is the event hub provider (only used in unit tests)
	eventHubProvider EventHubProvider
}

// DefaultOpts returns the default event relay options
func DefaultOpts() *Opts {
	return &Opts{
		RegTimeout:            3 * time.Second,
		RelayTimeout:          100 * time.Millisecond,
		eventHubProvider:      defaultEHProvider,
		EventHubRetryInterval: 2 * time.Second,
	}
}

// EventRelay registers for block and filtered block events with the
// Event Hub, and relays the events to all registered clients.
type EventRelay struct {
	channelID             string
	eventHubAddress       string
	regTimeout            time.Duration
	relayTimeout          time.Duration
	eventHubRetryInterval time.Duration
	eventHub              EventHub
	ehProvider            EventHubProvider
	mutex                 sync.RWMutex
	ehmutex               sync.RWMutex
	eventChannels         []chan<- interface{}
	tlsConfig             *tls.Config
}

// defaultEHProvider creates a new EventHub client
var defaultEHProvider EventHubProvider = func(channelID string, address string, regTimeout time.Duration, adapter EventAdapter, tlsConfig *tls.Config) (EventHub, error) {
	return NewEventsClient(address, regTimeout, adapter, tlsConfig)
}

// New creates a new event relay on the given channel.
func New(channelID string, eventHubAddress string, tlsConfig *tls.Config, opts *Opts) (*EventRelay, error) {
	if channelID == "" {
		return nil, errors.New(errors.GeneralError, "channelID is required")
	}

	if eventHubAddress == "" {
		return nil, errors.New(errors.GeneralError, "eventHubAddress is required")
	}

	// tlsCredentials are not required if insecure

	return &EventRelay{
		channelID:             channelID,
		eventHubAddress:       eventHubAddress,
		regTimeout:            opts.RegTimeout,
		relayTimeout:          opts.RelayTimeout,
		ehProvider:            opts.eventHubProvider,
		eventHubRetryInterval: opts.EventHubRetryInterval,
		tlsConfig:             tlsConfig,
	}, nil
}

// Start starts the event relay
func (er *EventRelay) Start() {
	// Start in the background since we don't want to hold up the caller
	go er.connectEventHub()
}

// ChannelID returns the channel ID
func (er *EventRelay) ChannelID() string {
	return er.channelID
}

// Register registers an event channel with the event relay. The event channel
// will receive events that are relayed from the event hub.
func (er *EventRelay) Register(eventch chan<- interface{}) {
	logger.Infof("Registering event channel with event relay.\n")
	er.mutex.Lock()
	defer er.mutex.Unlock()
	er.eventChannels = append(er.eventChannels, eventch)
}

// ---- Implementation of consumer.EventAdapter

// GetInterestedEvents implements EventAdapter.GetInterestedEvents
// The event relay registers for all Block and Filtered Block events
func (er *EventRelay) GetInterestedEvents() ([]*pb.Interest, error) {
	logger.Infof("Returning InterestedEvents - Block & FilteredBlock.\n")
	return []*pb.Interest{
		&pb.Interest{EventType: pb.EventType_BLOCK},
		&pb.Interest{EventType: pb.EventType_FILTEREDBLOCK},
	}, nil
}

// Recv implements EventAdapter.Recv
// Here the event is relayed to all subscribers.
func (er *EventRelay) Recv(event *pb.Event) (bool, error) {
	logger.Debugf("Received event: %s\n", event)

	if channelID, err := channelutil.ChannelIDFromEvent(event); err != nil {
		logger.Warnf("Unable to extract channel ID from the event: %s.\n", err)
		return true, nil
	} else if channelID != er.channelID {
		logger.Debugf("Received event from inapplicable channel [%s].\n", channelID)
		return true, nil
	}

	er.mutex.RLock()
	defer er.mutex.RUnlock()

	for _, eventch := range er.eventChannels {
		if er.relayTimeout < 0 {
			// Send will fail immediately if the channel buffer is full.
			select {
			case eventch <- event:
			default:
				logger.Warnf("Unable to relay event over channel since buffer is full.")
			}
		} else if er.relayTimeout == 0 {
			// Send will block.
			eventch <- event
		} else {
			// Send will fail after the relay timeout if the channel buffer is full.
			select {
			case eventch <- event:
			case <-time.After(er.relayTimeout):
				logger.Warnf("Timed out relaying event over channel.")
			}
		}
	}

	return true, nil
}

// Disconnected implements EventAdapter.Disconnected
// This function handles the disconnect by attempting to reconnect to a new event hub.
func (er *EventRelay) Disconnected(err error) {
	logger.Warnf("Disconnected: %s. Attempting to reconnect...\n", err)

	er.ehmutex.Lock()
	defer er.ehmutex.Unlock()

	er.eventHub = nil

	go er.connectEventHub()
}

func (er *EventRelay) connectEventHub() {
	logger.Infof("Starting event hub ...\n")

	er.ehmutex.Lock()
	defer er.ehmutex.Unlock()

	if er.eventHub != nil {
		logger.Errorf("Event hub already started\n")
		return
	}

	client, err := er.ehProvider(er.channelID, er.eventHubAddress, er.regTimeout, er, er.tlsConfig)
	if err != nil {
		logger.Errorf("Error creating new events client: %s\n", err)
		return
	}

	er.eventHub = client

	for {
		logger.Infof("... connecting to event hub at address %s ...\n", er.eventHubAddress)
		if err := er.eventHub.Start(); err != nil {
			logger.Errorf("Error starting event hub: %s. Will retry later.\n", err)
			time.Sleep(er.eventHubRetryInterval)
		} else {
			logger.Infof("... successfully connected to event hub.\n")
			return
		}
	}
}
