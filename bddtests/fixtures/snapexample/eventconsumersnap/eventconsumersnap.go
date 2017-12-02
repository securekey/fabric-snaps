/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/eventserver/pkg/channelutil"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
)

var logger = shim.NewLogger("EventConsumerSnap")

var initOnce sync.Once
var regmutex sync.RWMutex
var eventmutex sync.RWMutex

var blockRegistrations map[string]eventapi.Registration
var fblockRegistrations map[string]eventapi.Registration
var ccRegistrations map[string]eventapi.Registration
var txRegistrations map[string]eventapi.Registration

var blockEvents map[string][]*eventapi.BlockEvent
var fblockEvents map[string][]*eventapi.FilteredBlockEvent
var ccEvents map[string][]*eventapi.CCEvent
var txEvents map[string][]*eventapi.TxStatusEvent

// New chaincode implementation
func New() shim.Chaincode {
	return &eventConsumerSnap{}
}

// eventConsumerSnap is used in the EventSnap BDD test to test the features of the EventSnap
type eventConsumerSnap struct {
}

// Init registers for events
func (s *eventConsumerSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	initOnce.Do(func() {
		blockRegistrations = make(map[string]eventapi.Registration)
		fblockRegistrations = make(map[string]eventapi.Registration)
		ccRegistrations = make(map[string]eventapi.Registration)
		txRegistrations = make(map[string]eventapi.Registration)

		blockEvents = make(map[string][]*eventapi.BlockEvent)
		fblockEvents = make(map[string][]*eventapi.FilteredBlockEvent)
		ccEvents = make(map[string][]*eventapi.CCEvent)
		txEvents = make(map[string][]*eventapi.TxStatusEvent)
	})

	return shim.Success(nil)
}

// Invoke is not supported
func (s *eventConsumerSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	if function == "registerblock" {
		return registerBlockEvents(args)
	}
	if function == "unregisterblock" {
		return unregisterBlockEvents(args)
	}
	if function == "getblockevents" {
		return getBlockEvents(args)
	}
	if function == "deleteblockevents" {
		return deleteBlockEvents(args)
	}
	if function == "registerfilteredblock" {
		return registerFilteredBlockEvents(args)
	}
	if function == "unregisterfilteredblock" {
		return unregisterFilteredBlockEvents(args)
	}
	if function == "getfilteredblockevents" {
		return getFilteredBlockEvents(args)
	}
	if function == "deletefilteredblockevents" {
		return deleteFilteredBlockEvents(args)
	}
	if function == "registercc" {
		return registerCCEvents(args)
	}
	if function == "unregistercc" {
		return unregisterCCEvents(args)
	}
	if function == "getccevents" {
		return getCCEvents(args)
	}
	if function == "deleteccevents" {
		return deleteCCEvents(args)
	}
	if function == "registertx" {
		return registerTxEvents(args)
	}
	if function == "unregistertx" {
		return unregisterTxEvents(args)
	}
	if function == "gettxevents" {
		return getTxEvents(args)
	}
	if function == "deletetxevents" {
		return deleteTxEvents(args)
	}
	if function == "put" {
		return put(stub, args)
	}
	return shim.Error(fmt.Sprintf("Invoke not supported: %s", function))
}

func registerBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	regmutex.Lock()
	defer regmutex.Unlock()

	if _, ok := blockRegistrations[channelID]; ok {
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

	blockRegistrations[channelID] = reg

	go func() {
		logger.Infof("Listening for block events on channel: %s\n")
		for {
			bevent, ok := <-eventch
			if !ok {
				logger.Infof("Stopped listening for block events on channel %s\n", channelID)
				return
			}
			go func() {
				logger.Infof("Received block event: %v\n", bevent.Block)

				chID, err := channelutil.ChannelIDFromBlock(bevent.Block)
				if err != nil {
					logger.Errorf("Error extracting channel ID from block: %s\n", err)
				} else {
					eventmutex.Lock()
					defer eventmutex.Unlock()
					blockEvents[chID] = append(blockEvents[chID], bevent)
				}
			}()
		}
	}()

	return shim.Success(nil)
}

func unregisterBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	regmutex.Lock()
	defer regmutex.Unlock()

	reg, ok := blockRegistrations[channelID]
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

	delete(blockRegistrations, channelID)

	return shim.Success(nil)
}

func getBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.RLock()
	defer eventmutex.RUnlock()

	bytes, err := json.Marshal(blockEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling block events: %s", err))
	}

	return shim.Success(bytes)
}

func deleteBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.Lock()
	defer eventmutex.Unlock()

	delete(blockEvents, channelID)

	return shim.Success(nil)
}

func registerFilteredBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	regmutex.Lock()
	defer regmutex.Unlock()

	if _, ok := fblockRegistrations[channelID]; ok {
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

	fblockRegistrations[channelID] = reg

	go func() {
		logger.Infof("Listening for filtered block events on channel %s\n", channelID)
		for {
			fbevent, ok := <-eventch
			if !ok {
				logger.Infof("Stopped listening for filtered block events on channel %s\n", channelID)
				return
			}
			go func() {
				logger.Infof("Received filtered block event: %v\n", fbevent.FilteredBlock)

				chID, err := channelutil.ChannelIDFromFilteredBlock(fbevent.FilteredBlock)
				if err != nil {
					logger.Errorf("Error extracting channel ID from filtered block: %s\n", err)
				} else {
					eventmutex.Lock()
					defer eventmutex.Unlock()
					fblockEvents[chID] = append(fblockEvents[chID], fbevent)
				}
			}()
		}
	}()

	return shim.Success(nil)
}

func unregisterFilteredBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	reg, ok := fblockRegistrations[channelID]
	if !ok {
		// No registrations
		return shim.Success(nil)
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Unregistering filtered block events on channel %s\n", channelID)

	eventService.Unregister(reg)

	delete(fblockRegistrations, channelID)

	return shim.Success(nil)
}

func getFilteredBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.RLock()
	defer eventmutex.RUnlock()

	bytes, err := json.Marshal(fblockEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling filtered block events: %s", err))
	}

	return shim.Success(bytes)
}

func deleteFilteredBlockEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.Lock()
	defer eventmutex.Unlock()

	delete(fblockEvents, channelID)

	return shim.Success(nil)
}

func registerCCEvents(args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("Expecting channel ID, CC ID, and event filter")
	}

	channelID := args[0]
	ccID := args[1]
	eventFilter := args[2]

	regmutex.Lock()
	defer regmutex.Unlock()

	regKey := getCCRegKey(channelID, ccID, eventFilter)
	if _, ok := ccRegistrations[regKey]; ok {
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

	ccRegistrations[regKey] = reg

	go func() {
		logger.Infof("Listening for chaincode events on channel %s, CC ID: %s, Event filter: %s\n", channelID, ccID, eventFilter)
		for {
			ccevent, ok := <-eventch
			if !ok {
				logger.Infof("Stopped listening for chaincode events on channel %s, CC ID: %s, Event filter: %s\n", channelID, ccID, eventFilter)
				return
			}
			go func() {
				logger.Infof("Received CC event on channel %s, CC ID: %s, Event: %s, TxID: %s\n", channelID, ccID, ccevent.EventName, ccevent.TxID)
				eventmutex.Lock()
				defer eventmutex.Unlock()
				ccEvents[channelID] = append(ccEvents[channelID], ccevent)
			}()
		}
	}()

	return shim.Success(nil)
}

func unregisterCCEvents(args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("Expecting channel ID, CC ID, and event filter")
	}

	channelID := args[0]
	ccID := args[1]
	eventFilter := args[2]

	regKey := getCCRegKey(channelID, ccID, eventFilter)
	reg, ok := ccRegistrations[regKey]
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

	delete(ccRegistrations, regKey)

	return shim.Success(nil)
}

func getCCEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.RLock()
	defer eventmutex.RUnlock()

	bytes, err := json.Marshal(ccEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling CC events: %s", err))
	}

	return shim.Success(bytes)
}

func deleteCCEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.Lock()
	defer eventmutex.Unlock()

	delete(ccEvents, channelID)

	return shim.Success(nil)
}

func registerTxEvents(args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Expecting channel ID, and Tx ID")
	}

	channelID := args[0]
	txID := args[1]

	regmutex.Lock()
	defer regmutex.Unlock()

	regKey := getTxRegKey(channelID, txID)
	if _, ok := txRegistrations[regKey]; ok {
		return shim.Error(fmt.Sprintf("Tx Status registration already exists for channel %s and TxID %s", channelID, txID))
	}

	eventService := localservice.Get(channelID)
	if eventService == nil {
		return shim.Error(fmt.Sprintf("No local event service for channel: %s", channelID))
	}

	logger.Infof("Registering for Tx Status events on channel %s and TxID %s", channelID, txID)

	reg, eventch, err := eventService.RegisterTxStatusEvent(txID)
	if err != nil {
		return shim.Error(fmt.Sprintf("Error registering for Tx Status events on channel %s and TxID %s", channelID, txID))
	}

	txRegistrations[regKey] = reg

	go func() {
		logger.Infof("Listening for Tx Status events on channel %s for Tx: %s\n", channelID, txID)

		txevent, ok := <-eventch
		if !ok {
			logger.Infof("Stopped listening for Tx Status events for Tx: %s\n", txID)
			return
		}
		go func() {
			eventmutex.Lock()
			defer eventmutex.Unlock()
			logger.Infof("Received Tx Status event - TxID: %s, Status: %s\n", txevent.TxID, txevent.TxValidationCode)
			txEvents[channelID] = append(txEvents[channelID], txevent)
		}()
	}()

	return shim.Success(nil)
}

func unregisterTxEvents(args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Expecting channel ID and Tx ID")
	}

	channelID := args[0]
	txID := args[1]

	regKey := getTxRegKey(channelID, txID)
	reg, ok := txRegistrations[regKey]
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

	delete(txRegistrations, regKey)

	return shim.Success(nil)
}

func getTxEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	eventmutex.RLock()
	defer eventmutex.RUnlock()

	bytes, err := json.Marshal(txEvents[channelID])
	if err != nil {
		return shim.Error(fmt.Sprintf("Error marshalling Tx Status events: %s", err))
	}

	return shim.Success(bytes)
}

func deleteTxEvents(args []string) pb.Response {
	if len(args) < 1 {
		return shim.Error("Expecting channel ID")
	}

	channelID := args[0]

	delete(txEvents, channelID)

	return shim.Success(nil)
}

// put is called to generate events
func put(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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
