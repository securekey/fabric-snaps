/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"reflect"
	"time"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

const (
	// BLOCKEVENT is the Block event type
	BLOCKEVENT EventType = "BLOCKEVENT"

	// FILTEREDBLOCKEVENT is the Filtered Block event type
	FILTEREDBLOCKEVENT EventType = "FILTEREDBLOCKEVENT"
)

// Dispatcher is responsible for handling all events, including connection and registration events originating from the client,
// and events originating from the channel event service. All events are processed in a single Go routine
// in order to avoid any race conditions. This avoids the need for synchronization.
type Dispatcher struct {
	handlers                   map[reflect.Type]Handler
	eventch                    chan interface{}
	authorized                 map[EventType]bool
	blockRegistrations         []*blockRegistration
	filteredBlockRegistrations []*filteredBlockRegistration
	txRegistrations            map[string]*txRegistration
	ccRegistrations            map[string]*ccRegistration
	timeout                    time.Duration
}

// DispatcherOpts contains options for the dispatcher.
type DispatcherOpts struct {
	Opts

	// AuthorizedEventTypes is an array of event types that the client is allowed to subscribe to.
	AuthorizedEventTypes []EventType
}

// Handler is the handler for a given event type.
type Handler func(Event)

// NewDispatcher creates a new Dispatcher.
func NewDispatcher(opts *DispatcherOpts) *Dispatcher {
	logger.Debugf("Creating new dispatcher.\n")

	dispatcher := &Dispatcher{
		handlers:        make(map[reflect.Type]Handler),
		eventch:         make(chan interface{}, opts.EventConsumerBufferSize),
		txRegistrations: make(map[string]*txRegistration),
		ccRegistrations: make(map[string]*ccRegistration),
		authorized:      asMap(opts.AuthorizedEventTypes),
		timeout:         opts.EventConsumerTimeout,
	}
	dispatcher.RegisterHandlers()
	return dispatcher
}

// EventCh returns the channel to which events may be posted
func (ed *Dispatcher) EventCh() chan<- interface{} {
	return ed.eventch
}

// IsAuthorized returns true if the client is authorized to receive events of the given type.
func (ed *Dispatcher) IsAuthorized(eventType EventType) bool {
	return ed.authorized[eventType]
}

// SetAuthorizedEventTypes sets the event types that the client is authorized to receive.
func (ed *Dispatcher) SetAuthorizedEventTypes(eventTypes []EventType) {
	logger.Debugf("Setting authorized event types: %v ...\n", eventTypes)
	ed.authorized = make(map[EventType]bool)
	for _, eventType := range eventTypes {
		ed.authorized[eventType] = true
	}
}

// Start starts dispatching events as they arrive. All events are processed in
// a single Go routine in order to avoid any race conditions
func (ed *Dispatcher) Start() {
	logger.Debugf("Starting event dispatcher\n")

	go func() {
		for {
			logger.Debugf("Listening for events...\n")

			e, ok := <-ed.eventch
			if !ok {
				break
			}

			logger.Debugf("Received event: %v\n", reflect.TypeOf(e))

			if handler, ok := ed.handlers[reflect.TypeOf(e)]; ok {
				logger.Debugf("Dispatching event: %v\n", reflect.TypeOf(e))
				handler(e)
			} else {
				logger.Errorf("Handler not found for: %s", reflect.TypeOf(e))
			}
		}
		logger.Debugf("Exiting event dispatcher\n")
	}()
}

// Stop stops the dispatcher and unregisters all event registration.
func (ed *Dispatcher) Stop() {
	// Remove all registrations and close the associated event channels
	// so that the client is notified that the registration has been removed
	ed.ClearBlockRegistrations()
	ed.ClearFilteredBlockRegistrations()
	ed.ClearTxRegistrations()
	ed.ClearChaincodeRegistrations()

	logger.Debugf("Closing dispatcher event channel.\n")
	close(ed.eventch)
}

// ClearBlockRegistrations removes all block registrations and closes the corresponding event channels.
// The listener will receive a 'closed' event to indicate that the channel has been closed.
func (ed *Dispatcher) ClearBlockRegistrations() {
	for _, reg := range ed.blockRegistrations {
		close(reg.eventch)
	}
	ed.blockRegistrations = nil
}

// ClearFilteredBlockRegistrations removes all filtered block registrations and closes the corresponding event channels.
// The listener will receive a 'closed' event to indicate that the channel has been closed.
func (ed *Dispatcher) ClearFilteredBlockRegistrations() {
	for _, reg := range ed.filteredBlockRegistrations {
		close(reg.eventch)
	}
	ed.filteredBlockRegistrations = nil
}

// ClearTxRegistrations removes all transaction registrations and closes the corresponding event channels.
// The listener will receive a 'closed' event to indicate that the channel has been closed.
func (ed *Dispatcher) ClearTxRegistrations() {
	for _, reg := range ed.txRegistrations {
		logger.Debugf("Closing TX registration event channel for TxID [%s].\n", reg.txID)
		close(reg.eventch)
	}
	ed.txRegistrations = make(map[string]*txRegistration)
}

// ClearChaincodeRegistrations removes all chaincode registrations and closes the corresponding event channels.
// The listener will receive a 'closed' event to indicate that the channel has been closed.
func (ed *Dispatcher) ClearChaincodeRegistrations() {
	for _, reg := range ed.ccRegistrations {
		logger.Debugf("Closing chaincode registration event channel for CC ID [%s] and event filter [%s].\n", reg.ccID, reg.eventFilter)
		close(reg.eventch)
	}
	ed.ccRegistrations = make(map[string]*ccRegistration)
}

// HandleEvent handles the event
func (ed *Dispatcher) HandleEvent(e Event) {
	event := e.(*pb.Event)

	switch evt := event.Event.(type) {
	case *pb.Event_Block:
		ed.handleBlockEvent(evt)
	case *pb.Event_FilteredBlock:
		ed.handleFilteredBlockEvent(evt)
	default:
		logger.Warningf("Unsupported event type: %v", reflect.TypeOf(event.Event))
	}
}

func (ed *Dispatcher) handleRegisterBlockEvent(e Event) {
	event := e.(*registerBlockEvent)

	if !ed.IsAuthorized(BLOCKEVENT) {
		event.RespCh <- ErrorResponse(errors.New("client not authorized to receive block events"))
	} else {
		ed.blockRegistrations = append(ed.blockRegistrations, event.reg)
		event.RespCh <- SuccessResponse(event.reg)
	}
}

func (ed *Dispatcher) handleRegisterFilteredBlockEvent(e Event) {
	event := e.(*registerFilteredBlockEvent)

	if !ed.IsAuthorized(FILTEREDBLOCKEVENT) {
		event.RespCh <- ErrorResponse(errors.New("client not authorized to receive filtered block events"))
	} else {
		ed.filteredBlockRegistrations = append(ed.filteredBlockRegistrations, event.reg)
		event.RespCh <- SuccessResponse(event.reg)
	}
}

func (ed *Dispatcher) handleRegisterCCEvent(e Event) {
	event := e.(*registerCCEvent)

	key := getCCKey(event.reg.ccID, event.reg.eventFilter)
	if !ed.IsAuthorized(FILTEREDBLOCKEVENT) {
		event.RespCh <- ErrorResponse(errors.New("client not authorized to receive chaincode events"))
	} else if _, exists := ed.ccRegistrations[key]; exists {
		event.RespCh <- ErrorResponse(errors.Errorf("registration already exists for chaincode [%s] and event [%s]", event.reg.ccID, event.reg.eventFilter))
	} else {
		ed.ccRegistrations[key] = event.reg
		event.RespCh <- SuccessResponse(event.reg)
	}
}

func (ed *Dispatcher) handleRegisterTxStatusEvent(e Event) {
	event := e.(*registerTxStatusEvent)

	if !ed.IsAuthorized(FILTEREDBLOCKEVENT) {
		event.RespCh <- ErrorResponse(errors.New("client not authorized to receive TX events"))
	} else if _, exists := ed.txRegistrations[event.reg.txID]; exists {
		event.RespCh <- ErrorResponse(errors.Errorf("registration already exists for TX ID [%s]", event.reg.txID))
	} else {
		ed.txRegistrations[event.reg.txID] = event.reg
		event.RespCh <- SuccessResponse(event.reg)
	}
}

func (ed *Dispatcher) handleUnregisterEvent(e Event) {
	event := e.(*unregisterEvent)

	var err error
	switch registration := event.reg.(type) {
	case *blockRegistration:
		err = ed.unregisterBlockEvents(registration)
	case *filteredBlockRegistration:
		err = ed.unregisterFilteredBlockEvents(registration)
	case *ccRegistration:
		err = ed.unregisterCCEvents(registration)
	case *txRegistration:
		err = ed.unregisterTXEvents(registration)
	default:
		err = errors.Errorf("Unsupported registration type: %v", reflect.TypeOf(registration))
	}
	if err != nil {
		logger.Warningf("Error in unregister: %s\n", err)
	}
}

func (ed *Dispatcher) handleFilteredBlockEvent(event *pb.Event_FilteredBlock) {
	logger.Debugf("Handling filtered block event: %v\n", event)

	if event.FilteredBlock == nil || event.FilteredBlock.FilteredTx == nil {
		logger.Errorf("Received invalid filtered block event: %s", event)
		return
	}

	for _, reg := range ed.filteredBlockRegistrations {
		if ed.timeout < 0 {
			select {
			case reg.eventch <- &eventapi.FilteredBlockEvent{FilteredBlock: event.FilteredBlock}:
			default:
				logger.Warningf("Unable to send to filtered block event channel.")
			}
		} else if ed.timeout == 0 {
			reg.eventch <- &eventapi.FilteredBlockEvent{FilteredBlock: event.FilteredBlock}
		} else {
			select {
			case reg.eventch <- &eventapi.FilteredBlockEvent{FilteredBlock: event.FilteredBlock}:
			case <-time.After(ed.timeout):
				logger.Warningf("Timed out sending filtered block event.")
			}
		}
	}

	for _, tx := range event.FilteredBlock.FilteredTx {
		ed.triggerTxStatusEvent(tx)

		// Only send a chaincode event if the transaction has committed
		if tx.TxValidationCode == pb.TxValidationCode_VALID {
			txActions := tx.GetTransactionActions()
			if txActions == nil {
				continue
			}
			for _, action := range txActions.ChaincodeActions {
				if action.CcEvent != nil {
					ed.triggerCCEvent(action.CcEvent)
				}
			}
		}
	}
}

func (ed *Dispatcher) handleBlockEvent(event *pb.Event_Block) {
	logger.Debugf("Handling block event %v\n", event)

	for _, reg := range ed.blockRegistrations {
		if ed.timeout < 0 {
			select {
			case reg.eventch <- &eventapi.BlockEvent{Block: event.Block}:
			default:
				logger.Warningf("Unable to send to block event channel.")
			}
		} else if ed.timeout == 0 {
			reg.eventch <- &eventapi.BlockEvent{Block: event.Block}
		} else {
			select {
			case reg.eventch <- &eventapi.BlockEvent{Block: event.Block}:
			case <-time.After(ed.timeout):
				logger.Warningf("Timed out sending block event.")
			}
		}
	}
}

func (ed *Dispatcher) unregisterBlockEvents(registration *blockRegistration) error {
	for i, reg := range ed.blockRegistrations {
		if reg == registration {
			// Move the 0'th item to i and then delete the 0'th item
			ed.blockRegistrations[i] = ed.blockRegistrations[0]
			ed.blockRegistrations = ed.blockRegistrations[1:]
			close(reg.eventch)
			return nil
		}
	}
	return errors.New("the provided registration is invalid")
}

func (ed *Dispatcher) unregisterFilteredBlockEvents(registration *filteredBlockRegistration) error {
	for i, reg := range ed.filteredBlockRegistrations {
		if reg == registration {
			// Move the 0'th item to i and then delete the 0'th item
			ed.filteredBlockRegistrations[i] = ed.filteredBlockRegistrations[0]
			ed.filteredBlockRegistrations = ed.filteredBlockRegistrations[1:]
			close(reg.eventch)
			return nil
		}
	}
	return errors.New("the provided registration is invalid")
}

func (ed *Dispatcher) unregisterCCEvents(registration *ccRegistration) error {
	key := getCCKey(registration.ccID, registration.eventFilter)
	reg, ok := ed.ccRegistrations[key]
	if !ok {
		return errors.New("the provided registration is invalid")
	}

	logger.Debugf("Unregistering CC event for CC ID [%s] and event filter [%s]...\n", registration.ccID, registration.eventFilter)
	close(reg.eventch)
	delete(ed.ccRegistrations, key)
	return nil
}

func (ed *Dispatcher) unregisterTXEvents(registration *txRegistration) error {
	reg, ok := ed.txRegistrations[registration.txID]
	if !ok {
		return errors.New("the provided registration is invalid")
	}

	logger.Debugf("Unregistering Tx Status event for TxID [%s]...\n", registration.txID)
	close(reg.eventch)
	delete(ed.txRegistrations, registration.txID)
	return nil
}

func (ed *Dispatcher) triggerTxStatusEvent(tx *pb.FilteredTransaction) {
	logger.Debugf("Triggering Tx Status event for TxID [%s]...\n", tx.Txid)
	if reg, ok := ed.txRegistrations[tx.Txid]; ok {
		logger.Debugf("Sending Tx Status event for TxID [%s] to registrant...\n", tx.Txid)

		if ed.timeout < 0 {
			select {
			case reg.eventch <- newTxStatusEvent(tx.Txid, tx.TxValidationCode):
			default:
				logger.Warningf("Unable to send to Tx Status event channel.")
			}
		} else if ed.timeout == 0 {
			reg.eventch <- newTxStatusEvent(tx.Txid, tx.TxValidationCode)
		} else {
			select {
			case reg.eventch <- newTxStatusEvent(tx.Txid, tx.TxValidationCode):
			case <-time.After(ed.timeout):
				logger.Warningf("Timed out sending Tx Status event.")
			}
		}
	}
}

func (ed *Dispatcher) triggerCCEvent(ccEvent *pb.ChaincodeEvent) {
	for _, reg := range ed.ccRegistrations {
		logger.Debugf("Matching CCEvent[%s,%s] against Reg[%s,%s] ...\n", ccEvent.ChaincodeId, ccEvent.EventName, reg.ccID, reg.eventFilter)
		if reg.ccID == ccEvent.ChaincodeId && reg.eventRegExp.MatchString(ccEvent.EventName) {
			logger.Debugf("... matched CCEvent[%s,%s] against Reg[%s,%s]\n", ccEvent.ChaincodeId, ccEvent.EventName, reg.ccID, reg.eventFilter)

			if ed.timeout < 0 {
				select {
				case reg.eventch <- newCCEvent(ccEvent.ChaincodeId, ccEvent.EventName, ccEvent.TxId):
				default:
					logger.Warningf("Unable to send to CC event channel.")
				}
			} else if ed.timeout == 0 {
				reg.eventch <- newCCEvent(ccEvent.ChaincodeId, ccEvent.EventName, ccEvent.TxId)
			} else {
				select {
				case reg.eventch <- newCCEvent(ccEvent.ChaincodeId, ccEvent.EventName, ccEvent.TxId):
				case <-time.After(ed.timeout):
					logger.Warningf("Timed out sending CC event.")
				}
			}
		}
	}
}

// RegisterHandlers registers all of the event handlers
func (ed *Dispatcher) RegisterHandlers() {
	ed.RegisterHandler(&registerCCEvent{}, ed.handleRegisterCCEvent)
	ed.RegisterHandler(&registerTxStatusEvent{}, ed.handleRegisterTxStatusEvent)
	ed.RegisterHandler(&registerBlockEvent{}, ed.handleRegisterBlockEvent)
	ed.RegisterHandler(&registerFilteredBlockEvent{}, ed.handleRegisterFilteredBlockEvent)
	ed.RegisterHandler(&unregisterEvent{}, ed.handleUnregisterEvent)
	ed.RegisterHandler(&pb.Event{}, ed.HandleEvent)
}

// RegisterHandler registers an event handler
func (ed *Dispatcher) RegisterHandler(t interface{}, h Handler) {
	logger.Debugf("Registering handler for %s\n", reflect.TypeOf(t))
	ed.handlers[reflect.TypeOf(t)] = h
}

// SuccessResponse returns a success response
func SuccessResponse(reg eventapi.Registration) *eventapi.RegistrationResponse {
	return &eventapi.RegistrationResponse{Reg: reg}
}

// ErrorResponse returns an error response
func ErrorResponse(err error) *eventapi.RegistrationResponse {
	return &eventapi.RegistrationResponse{Err: err}
}

// UnregResponse returns an unregister response
func UnregResponse(err error) *eventapi.RegistrationResponse {
	return &eventapi.RegistrationResponse{Err: err}
}

func getCCKey(ccID, eventFilter string) string {
	return ccID + "/" + eventFilter
}

func asMap(eventTypes []EventType) map[EventType]bool {
	m := make(map[EventType]bool)
	for _, eventType := range eventTypes {
		m[eventType] = true
	}
	return m
}
