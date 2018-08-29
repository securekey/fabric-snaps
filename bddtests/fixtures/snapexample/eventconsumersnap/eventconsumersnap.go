/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/bddtests/fixtures/snapexample/eventconsumersnap/channelutil"
	"github.com/securekey/fabric-snaps/bddtests/fixtures/snapexample/eventconsumersnap/common"
	"github.com/securekey/fabric-snaps/mocks/event/mockservice/pkg/localservice"
)

var logger = shim.NewLogger("EventConsumerSnap")

const (
	// Available function names
	putFunc                       = "put"
	registerBlockFunc             = "registerblock"
	unregisterBlockFunc           = "unregisterblock"
	getBlockeventsFunc            = "getblockevents"
	deleteBlockEventsFunc         = "deleteblockevents"
	registerFilteredBlockFunc     = "registerfilteredblock"
	unregisterFilteredBlockFunc   = "unregisterfilteredblock"
	getFilteredBlockEventsFunc    = "getfilteredblockevents"
	deleteFilteredBlockEventsFunc = "deletefilteredblockevents"
	registerCCFunc                = "registercc"
	unregisterCCFunc              = "unregistercc"
	getCCEventsFunc               = "getccevents"
	deleteCCEventsFunc            = "deleteccevents"
	registerTxFunc                = "registertx"
	unregisterTxFunc              = "unregistertx"
	getTxEventsFunc               = "gettxevents"
	deleteTxEventsFunc            = "deletetxevents"
)

// funcMap is a map of functions by function name
type funcMap map[string]func(shim.ChaincodeStubInterface, []string) pb.Response

// eventConsumerSnap is used in the EventSnap BDD test to test the features of the EventSnap
type eventConsumerSnap struct {
	functions           funcMap
	regmutex            sync.RWMutex
	eventmutex          sync.RWMutex
	blockRegistrations  map[string]fab.Registration
	fblockRegistrations map[string]fab.Registration
	ccRegistrations     map[string]fab.Registration
	txRegistrations     map[string]fab.Registration
	blockEvents         map[string][]*fab.BlockEvent
	fblockEvents        map[string][]*fab.FilteredBlockEvent
	ccEvents            map[string][]*fab.CCEvent
	txEvents            map[string][]*fab.TxStatusEvent
}

// New chaincode implementation
func New() shim.Chaincode {
	s := &eventConsumerSnap{
		functions:           make(funcMap),
		blockRegistrations:  make(map[string]fab.Registration),
		fblockRegistrations: make(map[string]fab.Registration),
		ccRegistrations:     make(map[string]fab.Registration),
		txRegistrations:     make(map[string]fab.Registration),
		blockEvents:         make(map[string][]*fab.BlockEvent),
		fblockEvents:        make(map[string][]*fab.FilteredBlockEvent),
		ccEvents:            make(map[string][]*fab.CCEvent),
		txEvents:            make(map[string][]*fab.TxStatusEvent),
	}

	s.functions[registerBlockFunc] = s.registerBlockEvents
	s.functions[unregisterBlockFunc] = s.unregisterBlockEvents
	s.functions[getBlockeventsFunc] = s.getBlockEvents
	s.functions[deleteBlockEventsFunc] = s.deleteBlockEvents
	s.functions[registerFilteredBlockFunc] = s.registerFilteredBlockEvents
	s.functions[unregisterFilteredBlockFunc] = s.unregisterFilteredBlockEvents
	s.functions[getFilteredBlockEventsFunc] = s.getFilteredBlockEvents
	s.functions[deleteFilteredBlockEventsFunc] = s.deleteFilteredBlockEvents
	s.functions[registerCCFunc] = s.registerCCEvents
	s.functions[unregisterCCFunc] = s.unregisterCCEvents
	s.functions[getCCEventsFunc] = s.getCCEvents
	s.functions[deleteCCEventsFunc] = s.deleteCCEvents
	s.functions[registerTxFunc] = s.registerTxEvents
	s.functions[unregisterTxFunc] = s.unregisterTxEvents
	s.functions[getTxEventsFunc] = s.getTxEvents
	s.functions[deleteTxEventsFunc] = s.deleteTxEvents
	s.functions[putFunc] = s.put

	return s
}

// Init registers for events
func (s *eventConsumerSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke invokes various test functions
func (s *eventConsumerSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	functionName, args := stub.GetFunctionAndParameters()
	if functionName == "" {
		return shim.Error("Function name is required")
	}

	function, valid := s.functions[functionName]
	if !valid {
		fnNames := []string{}
		for k := range s.functions {
			fnNames = append(fnNames, k)
		}
		return shim.Error(fmt.Sprintf("Invalid invoke function [%s]. Expecting one of: %s", functionName, fnNames))
	}

	return function(stub, args)
}

func (s *eventConsumerSnap) registerBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.regmutex.Lock()
	defer s.regmutex.Unlock()

	if _, ok := s.blockRegistrations[channelID]; ok {
		return shim.Error(fmt.Sprintf("Block registration already exists for channel: %s", channelID))
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Registering for block events on channel %s ...\n", channelID)

	reg, eventch, err := eventService.RegisterBlockEvent()
	if err != nil {
		return shim.Error(fmt.Sprintf("Error registering for block events on channel: %s", channelID))
	}

	s.blockRegistrations[channelID] = reg

	go func() {
		logger.Info("Listening for block events on channel...")
		for {
			bevent, ok := <-eventch
			if !ok {
				logger.Infof("Stopped listening for block events on channel %s", channelID)
				return
			}
			go func() {
				logger.Infof("Received block event: %s", bevent.Block)

				chID, err := channelutil.ChannelIDFromBlock(bevent.Block)
				if err != nil {
					logger.Errorf("Error extracting channel ID from block: %s", err)
				} else {
					s.eventmutex.Lock()
					defer s.eventmutex.Unlock()
					s.blockEvents[chID] = append(s.blockEvents[chID], bevent)
				}
			}()
		}
	}()

	return shim.Success(nil)
}

func (s *eventConsumerSnap) unregisterBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.regmutex.Lock()
	defer s.regmutex.Unlock()

	reg, ok := s.blockRegistrations[channelID]
	if !ok {
		// No registrations
		return shim.Success(nil)
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Unregistering block events on channel %s\n", channelID)

	eventService.Unregister(reg)

	delete(s.blockRegistrations, channelID)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) getBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.RLock()
	defer s.eventmutex.RUnlock()

	bytes, err := json.Marshal(s.blockEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling block events: %s", err))
	}

	return shim.Success(bytes)
}

func (s *eventConsumerSnap) deleteBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.Lock()
	defer s.eventmutex.Unlock()

	delete(s.blockEvents, channelID)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) registerFilteredBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.regmutex.Lock()
	defer s.regmutex.Unlock()

	if _, ok := s.fblockRegistrations[channelID]; ok {
		return shim.Error(fmt.Sprintf("Filtered block registration already exists for channel: %s", channelID))
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Registering for filtered block events on channel %s ...\n", channelID)

	reg, eventch, err := eventService.RegisterFilteredBlockEvent()
	if err != nil {
		return shim.Error(fmt.Sprintf("Error registering for filtered block events on channel: %s", channelID))
	}

	s.fblockRegistrations[channelID] = reg

	go func() {
		logger.Infof("Listening for filtered block events on channel %s ...", channelID)
		for {
			fbevent, ok := <-eventch
			if !ok {
				logger.Infof("Stopped listening for filtered block events on channel %s", channelID)
				return
			}
			go func() {
				logger.Infof("Received filtered block event: %s", fbevent.FilteredBlock)

				chID, err := channelutil.ChannelIDFromFilteredBlock(fbevent.FilteredBlock)
				if err != nil {
					logger.Errorf("Error extracting channel ID from filtered block: %s", err)
				} else {
					s.eventmutex.Lock()
					defer s.eventmutex.Unlock()
					s.fblockEvents[chID] = append(s.fblockEvents[chID], fbevent)
				}
			}()
		}
	}()

	return shim.Success(nil)
}

func (s *eventConsumerSnap) unregisterFilteredBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	reg, ok := s.fblockRegistrations[channelID]
	if !ok {
		// No registrations
		return shim.Success(nil)
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Unregistering filtered block events on channel %s", channelID)

	eventService.Unregister(reg)

	delete(s.fblockRegistrations, channelID)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) getFilteredBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.RLock()
	defer s.eventmutex.RUnlock()

	bytes, err := protoToJSON(s.fblockEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling filtered block events for channel [%s], error: %s", channelID, err))
	}

	return shim.Success(bytes)
}

func (s *eventConsumerSnap) deleteFilteredBlockEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.Lock()
	defer s.eventmutex.Unlock()

	delete(s.fblockEvents, channelID)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) registerCCEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("Expecting channel ID, CC ID, and event filter")
	}

	channelID := args[0]
	ccID := args[1]
	eventFilter := args[2]

	s.regmutex.Lock()
	defer s.regmutex.Unlock()

	regKey := getCCRegKey(channelID, ccID, eventFilter)
	if _, ok := s.ccRegistrations[regKey]; ok {
		return shim.Error(fmt.Sprintf("CC registration already exists for channel %s, CC %s, and event filter %s", channelID, ccID, eventFilter))
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Registering for CC events on channel %s, CC %s, and event filter %s", channelID, ccID, eventFilter)

	reg, eventch, err := eventService.RegisterChaincodeEvent(ccID, eventFilter)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error registering for CC events on channel %s, CC %s, and event filter %s", channelID, ccID, eventFilter))
	}

	s.ccRegistrations[regKey] = reg

	go func() {
		logger.Infof("Listening for chaincode events on channel %s, CC ID: %s, Event filter: %s ...", channelID, ccID, eventFilter)
		for {
			ccevent, ok := <-eventch
			if !ok {
				logger.Infof("Stopped listening for chaincode events on channel %s, CC ID: %s, Event filter: %s", channelID, ccID, eventFilter)
				return
			}
			go func() {
				logger.Infof("Received CC event on channel %s, CC ID: %s, Event: %s, TxID: %s", channelID, ccID, ccevent.EventName, ccevent.TxID)
				s.eventmutex.Lock()
				defer s.eventmutex.Unlock()
				s.ccEvents[channelID] = append(s.ccEvents[channelID], ccevent)
			}()
		}
	}()

	return shim.Success(nil)
}

func (s *eventConsumerSnap) unregisterCCEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("Expecting channel ID, CC ID, and event filter")
	}

	channelID := args[0]
	ccID := args[1]
	eventFilter := args[2]

	regKey := getCCRegKey(channelID, ccID, eventFilter)
	reg, ok := s.ccRegistrations[regKey]
	if !ok {
		// No registrations
		return shim.Success(nil)
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Unregistering CC events for channel %s, CC %s, and event filter %s", channelID, ccID, eventFilter)

	eventService.Unregister(reg)

	delete(s.ccRegistrations, regKey)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) getCCEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.RLock()
	defer s.eventmutex.RUnlock()

	bytes, err := json.Marshal(s.ccEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling CC events: %s", err))
	}

	return shim.Success(bytes)
}

func (s *eventConsumerSnap) deleteCCEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.Lock()
	defer s.eventmutex.Unlock()

	delete(s.ccEvents, channelID)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) registerTxEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Expecting channel ID, and Tx ID")
	}

	channelID := args[0]
	txID := args[1]

	s.regmutex.Lock()
	defer s.regmutex.Unlock()

	regKey := getTxRegKey(channelID, txID)
	if _, ok := s.txRegistrations[regKey]; ok {
		return shim.Error(fmt.Sprintf("Tx Status registration already exists for channel %s and TxID %s", channelID, txID))
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Registering for Tx Status events on channel %s and TxID %s", channelID, txID)

	reg, eventch, err := eventService.RegisterTxStatusEvent(txID)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error registering for Tx Status events on channel %s and TxID %s err:%s", channelID, txID, err))
	}

	s.txRegistrations[regKey] = reg

	go func() {
		logger.Infof("Listening for Tx Status events on channel %s for Tx: %s", channelID, txID)

		txevent, ok := <-eventch
		if !ok {
			logger.Infof("Stopped listening for Tx Status events for Tx: %s", txID)
			return
		}
		go func() {
			s.eventmutex.Lock()
			defer s.eventmutex.Unlock()
			logger.Infof("Received Tx Status event - TxID: %s, Status: %s", txevent.TxID, txevent.TxValidationCode)
			s.txEvents[channelID] = append(s.txEvents[channelID], txevent)
		}()
	}()

	return shim.Success(nil)
}

func (s *eventConsumerSnap) unregisterTxEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Expecting channel ID and Tx ID")
	}

	channelID := args[0]
	txID := args[1]

	regKey := getTxRegKey(channelID, txID)
	reg, ok := s.txRegistrations[regKey]
	if !ok {
		// No registrations
		return shim.Success(nil)
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Unregistering Tx Status events for channel %s and Tx ID %s", channelID, txID)

	eventService.Unregister(reg)

	delete(s.txRegistrations, regKey)

	return shim.Success(nil)
}

func (s *eventConsumerSnap) getTxEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	s.eventmutex.RLock()
	defer s.eventmutex.RUnlock()

	bytes, err := json.Marshal(s.txEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling Tx Status events: %s", err))
	}

	return shim.Success(bytes)
}

func (s *eventConsumerSnap) deleteTxEvents(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	delete(s.txEvents, channelID)

	return shim.Success(nil)
}

// put is called to generate events
func (s *eventConsumerSnap) put(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Expecting key, value, and optional event")
	}

	key := args[0]
	value := args[1]

	var eventName string
	if len(args) > 2 {
		eventName = args[2]
	}

	if err := stub.PutState(key, []byte(value)); err != nil {
		return shim.Error(fmt.Sprintf("Error putting state: %s", err))
	}

	if eventName != "" {
		stub.SetEvent(eventName, nil)
	}

	return shim.Success(nil)
}

func getCCRegKey(channelID, ccID, eventFilter string) string {
	return "cc_" + channelID + "_" + ccID + "_" + eventFilter
}

func getTxRegKey(channelID, txID string) string {
	return "tx_" + channelID + "_" + txID
}

func main() {
}

// protoToJSON is a simple shortcut wrapper around the proto JSON marshaler
func protoToJSON(msg []*fab.FilteredBlockEvent) ([]byte, error) {
	// step 1, marshall each FilteredBlock in the FilteredBlockEvent
	// array and set them to the Payload of ByteFilteredBlockEvent
	var marshalledFbe []*common.ByteFilteredBlockEvent
	m := jsonpb.Marshaler{
		EnumsAsInts:  true,
		EmitDefaults: true,
		Indent:       "  ",
		OrigName:     true,
	}

	for _, fbe := range msg {
		var b bytes.Buffer
		err := m.Marshal(&b, fbe.FilteredBlock)
		if err != nil {
			return nil, err
		}

		marshalledFbe = append(marshalledFbe, &common.ByteFilteredBlockEvent{Payload: b.Bytes(), SourceURL: fbe.SourceURL})
	}

	// step 2, marshall the full array
	bytes, err := json.Marshal(marshalledFbe)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
