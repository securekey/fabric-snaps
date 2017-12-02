/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"regexp"
	"sync"
	"time"

	logging "github.com/op/go-logging"
	"github.com/pkg/errors"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

var logger = logging.MustGetLogger("eventservice")

// EventProducer produces events which are dispatched to clients
type EventProducer interface {
	// Register registers the given event channel with the event producer
	// and events are sent to this channel.
	Register(eventch chan<- interface{})
}

// EventDispatcher is responsible for processing registration requests and block/filtered block events.
type EventDispatcher interface {
	// Start starts the dispatcher, i.e. the dispatcher starts listening for requests/events
	Start()

	// Stop stops the dispatcher
	Stop()

	// EventCh is the event channel over which to communicate with the dispatcher
	EventCh() chan<- interface{}
}

// EventService allows clients to register for channel events, such as filtered block, chaincode, and transaction status events.
type EventService struct {
	eventTypes      []EventType
	opts            Opts
	dispatcher      EventDispatcher
	registerOnce    sync.Once
	eventBufferSize uint
}

// Opts provides options for the events client
type Opts struct {
	// EventConsumerBufferSize is the size of the registered consumer's event channel.
	EventConsumerBufferSize uint

	// EventConsumerTimeout is the timeout when sending events to a registered consumer.
	// If < 0, if buffer full, unblocks immediately and does not send.
	// If 0, if buffer full, will block and guarantee the event will be sent out.
	// If > 0, if buffer full, blocks util timeout.
	EventConsumerTimeout time.Duration
}

// DefaultOpts returns client options set to default values
func DefaultOpts() *Opts {
	return &Opts{
		EventConsumerBufferSize: 100,
		EventConsumerTimeout:    100 * time.Millisecond,
	}
}

// NewService returns a new event service
func NewService(opts *Opts, eventTypes []EventType) *EventService {
	return NewServiceWithDispatcher(
		NewDispatcher(
			&DispatcherOpts{
				Opts:                 *opts,
				AuthorizedEventTypes: eventTypes,
			},
		),
		opts,
		eventTypes,
	)
}

// NewServiceWithDispatcher returns a new event service initialized with the given Dispatcher
func NewServiceWithDispatcher(dispatcher EventDispatcher, opts *Opts, eventTypes []EventType) *EventService {
	return &EventService{
		eventTypes:      eventTypes,
		opts:            *opts,
		dispatcher:      dispatcher,
		eventBufferSize: opts.EventConsumerBufferSize,
	}
}

// Start starts the event service
func (s *EventService) Start(producer EventProducer) {
	s.dispatcher.Start()
	producer.Register(s.dispatcher.EventCh())
}

// Stop stops the event service
func (s *EventService) Stop() {
	s.dispatcher.Stop()
}

// Submit submits an event for processing
func (s *EventService) Submit(event interface{}) {
	defer func() {
		// During shutdown, events may still be produced and we may
		// get a 'send on closed channel' panic. Just log and ignore the error.
		if p := recover(); p != nil {
			logger.Warningf("panic while submitting event: %s", p)
		}
	}()

	s.dispatcher.EventCh() <- event
}

// Dispatcher returns the dispatcher
func (s *EventService) Dispatcher() EventDispatcher {
	return s.dispatcher
}

// EventChannelSize returns the event channel size
func (s *EventService) EventChannelSize() uint {
	return s.eventBufferSize
}

// EventTypes returns the event types that are to be handled
func (s *EventService) EventTypes() []EventType {
	return s.eventTypes
}

// RegisterBlockEvent registers for block events. If the client is not authorized to receive
// block events then an error is returned.
func (s *EventService) RegisterBlockEvent() (eventapi.Registration, <-chan *eventapi.BlockEvent, error) {
	eventch := make(chan *eventapi.BlockEvent, s.eventBufferSize)
	respch := make(chan *eventapi.RegistrationResponse)
	s.Submit(newRegisterBlockEvent(eventch, respch))
	response := <-respch

	return response.Reg, eventch, response.Err
}

// RegisterFilteredBlockEvent registers for filtered block events. If the client is not authorized to receive
// filtered block events then an error is returned.
func (s *EventService) RegisterFilteredBlockEvent() (eventapi.Registration, <-chan *eventapi.FilteredBlockEvent, error) {
	eventch := make(chan *eventapi.FilteredBlockEvent, s.eventBufferSize)
	respch := make(chan *eventapi.RegistrationResponse)
	s.Submit(newRegisterFilteredBlockEvent(eventch, respch))
	response := <-respch

	return response.Reg, eventch, response.Err
}

// RegisterChaincodeEvent registers for chaincode events. If the client is not authorized to receive
// chaincode events then an error is returned.
// - ccID is the chaincode ID for which events are to be received
// - eventFilter is the chaincode event name for which events are to be received
func (s *EventService) RegisterChaincodeEvent(ccID, eventFilter string) (eventapi.Registration, <-chan *eventapi.CCEvent, error) {
	if ccID == "" {
		return nil, nil, errors.New("chaincode ID is required")
	}
	if eventFilter == "" {
		return nil, nil, errors.New("event filter is required")
	}

	regExp, err := regexp.Compile(eventFilter)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid event filter [%s] for chaincode [%s]", eventFilter, ccID)
	}

	eventch := make(chan *eventapi.CCEvent, s.eventBufferSize)
	respch := make(chan *eventapi.RegistrationResponse)
	s.Submit(newRegisterCCEvent(ccID, eventFilter, regExp, eventch, respch))
	response := <-respch

	return response.Reg, eventch, response.Err
}

// RegisterTxStatusEvent registers for transaction status events. If the client is not authorized to receive
// transaction status events then an error is returned.
// - txID is the transaction ID for which events are to be received
func (s *EventService) RegisterTxStatusEvent(txID string) (eventapi.Registration, <-chan *eventapi.TxStatusEvent, error) {
	if txID == "" {
		return nil, nil, errors.New("txID must be provided")
	}

	eventch := make(chan *eventapi.TxStatusEvent, s.eventBufferSize)
	respch := make(chan *eventapi.RegistrationResponse)
	s.Submit(newRegisterTxStatusEvent(txID, eventch, respch))
	response := <-respch

	return response.Reg, eventch, response.Err
}

// Unregister unregisters the given registration.
// - reg is the registration handle that was returned from one of the RegisterXXX functions
func (s *EventService) Unregister(reg eventapi.Registration) {
	s.Submit(newUnregisterEvent(reg))
}
