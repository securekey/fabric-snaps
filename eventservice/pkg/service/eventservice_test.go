/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/eventserver/pkg/mocks"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

type Outcome string
type State int32
type NumBlockEvents uint
type NumCCEvents uint

type EventsReceived struct {
	BlockEvents NumBlockEvents
	CCEvents    NumCCEvents
}

const (
	initialState State = -1

	reconnectedOutcome Outcome = "reconnected"
	terminatedOutcome  Outcome = "terminated"
	timedOutOutcome    Outcome = "timeout"
	connectedOutcome   Outcome = "connected"
	errorOutcome       Outcome = "error"
)

func TestInvalidUnregister(t *testing.T) {
	channelID := "mychannel"
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{BLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	// Make sure the client doesn't panic with invalid registration
	eventService.Unregister("invalid registration")
}

func TestBlockEvents(t *testing.T) {
	channelID := "mychannel"
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{BLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	registration, eventch, err := eventService.RegisterBlockEvent()
	if err != nil {
		t.Fatalf("error registering for block events: %s", err)
	}
	defer eventService.Unregister(registration)

	eventProducer.ProduceEvent(mocks.NewMockBlockEvent(channelID))

	select {
	case _, ok := <-eventch:
		if !ok {
			t.Fatalf("unexpected closed channel")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for block event")
	}
}

func TestBlockEventsUnauthorized(t *testing.T) {
	eventService, eventProducer, err := newServiceWithMockConn("mychannel", []EventType{FILTEREDBLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	if _, _, err := eventService.RegisterBlockEvent(); err == nil {
		t.Fatalf("expecting authorization error since client is not authorized to receive block events")
	}
}

func TestFilteredBlockEvents(t *testing.T) {
	channelID := "mychannel"
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{FILTEREDBLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	registration, eventch, err := eventService.RegisterFilteredBlockEvent()
	if err != nil {
		t.Fatalf("error registering for filtered block events: %s", err)
	}
	defer eventService.Unregister(registration)

	txID1 := "1234"
	txCode1 := pb.TxValidationCode_VALID
	txID2 := "5678"
	txCode2 := pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE

	eventProducer.ProduceEvent(mocks.NewMockFilteredBlockEvent(
		channelID,
		mocks.NewMockFilteredTx(txID1, txCode1),
		mocks.NewMockFilteredTx(txID2, txCode2),
	))

	select {
	case fbevent, ok := <-eventch:
		if !ok {
			t.Fatalf("unexpected closed channel")
		}
		if fbevent.FilteredBlock == nil {
			t.Fatalf("Expecting filtered block but got nil")
		}
		if fbevent.FilteredBlock.ChannelId != channelID {
			t.Fatalf("Expecting channel [%s] but got [%s]", channelID, fbevent.FilteredBlock.ChannelId)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for filtered block event")
	}
}

func TestFilteredBlockEventsUnauthorized(t *testing.T) {
	eventService, eventProducer, err := newServiceWithMockConn("mychannel", []EventType{}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	if _, _, err := eventService.RegisterFilteredBlockEvent(); err == nil {
		t.Fatalf("expecting authorization error since client is not authorized to receive filtered block events")
	}
}

func TestBlockAndFilteredBlockEvents(t *testing.T) {
	channelID := "mychannel"
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{BLOCKEVENT, FILTEREDBLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	// First register for filtered block events
	fbreg, fbeventch, err := eventService.RegisterFilteredBlockEvent()
	if err != nil {
		t.Fatalf("error registering for filtered block events: %s", err)
	}
	defer eventService.Unregister(fbreg)

	txID1 := "1234"
	txCode1 := pb.TxValidationCode_VALID
	txID2 := "5678"
	txCode2 := pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE

	eventProducer.ProduceEvent(mocks.NewMockFilteredBlockEvent(
		channelID,
		mocks.NewMockFilteredTx(txID1, txCode1),
		mocks.NewMockFilteredTx(txID2, txCode2),
	))

	select {
	case fbevent, ok := <-fbeventch:
		if !ok {
			t.Fatalf("unexpected closed channel")
		}
		if fbevent.FilteredBlock == nil {
			t.Fatalf("Expecting filtered block but got nil")
		}
		if fbevent.FilteredBlock.ChannelId != channelID {
			t.Fatalf("Expecting channel [%s] but got [%s]", channelID, fbevent.FilteredBlock.ChannelId)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for filtered block event")
	}

	// Now register for block events
	breg, beventch, err := eventService.RegisterBlockEvent()
	if err != nil {
		t.Fatalf("error registering for block events: %s", err)
	}
	defer eventService.Unregister(breg)

	eventProducer.ProduceEvent(mocks.NewMockBlockEvent(channelID))
	eventProducer.ProduceEvent(mocks.NewMockFilteredBlockEvent(
		channelID,
		mocks.NewMockFilteredTx(txID1, txCode1),
		mocks.NewMockFilteredTx(txID2, txCode2),
	))
	numEventsReceived := 0

	expectedEvents := 2

	for {
		select {
		case _, ok := <-fbeventch:
			if !ok {
				t.Fatalf("unexpected closed channel")
			}
			numEventsReceived++
		case _, ok := <-beventch:
			if !ok {
				t.Fatalf("unexpected closed channel")
			}
			numEventsReceived++
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for events")
		}
		if numEventsReceived == expectedEvents {
			break
		}
	}
}

func TestTxStatusEvents(t *testing.T) {
	channelID := "mychannel"
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{FILTEREDBLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	txID1 := "1234"
	txCode1 := pb.TxValidationCode_VALID
	txID2 := "5678"
	txCode2 := pb.TxValidationCode_ENDORSEMENT_POLICY_FAILURE

	if _, _, err := eventService.RegisterTxStatusEvent(""); err == nil {
		t.Fatalf("expecting error registering for TxStatus event without a TX ID but got none")
	}
	reg1, _, err := eventService.RegisterTxStatusEvent(txID1)
	if err != nil {
		t.Fatalf("error registering for TxStatus events: %s", err)
	}
	_, _, err = eventService.RegisterTxStatusEvent(txID1)
	if err == nil {
		t.Fatalf("expecting error registering multiple times for TxStatus events: %s", err)
	}
	eventService.Unregister(reg1)

	reg1, eventch1, err := eventService.RegisterTxStatusEvent(txID1)
	if err != nil {
		t.Fatalf("error registering for TxStatus events: %s", err)
	}
	defer eventService.Unregister(reg1)

	reg2, eventch2, err := eventService.RegisterTxStatusEvent(txID2)
	if err != nil {
		t.Fatalf("error registering for TxStatus events: %s", err)
	}
	defer eventService.Unregister(reg2)

	eventProducer.ProduceEvent(
		mocks.NewMockFilteredBlockEvent(
			channelID,
			mocks.NewMockFilteredTx(txID1, txCode1),
			mocks.NewMockFilteredTx(txID2, txCode2),
		),
	)

	numExpected := 2
	numReceived := 0
	done := false
	for !done {
		select {
		case event, ok := <-eventch1:
			if !ok {
				t.Fatalf("unexpected closed channel")
			} else {
				checkTxStatusEvent(t, event, txID1, txCode1)
				numReceived++
			}
		case event, ok := <-eventch2:
			if !ok {
				t.Fatalf("unexpected closed channel")
			} else {
				checkTxStatusEvent(t, event, txID2, txCode2)
				numReceived++
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for [%d] TxStatus events. Only received [%d]", numExpected, numReceived)
		}

		if numReceived == numExpected {
			break
		}
	}
}

func TestTxStatusEventsUnauthorized(t *testing.T) {
	eventService, eventProducer, err := newServiceWithMockConn("mychannel", []EventType{}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	if _, _, err := eventService.RegisterTxStatusEvent("txid"); err == nil {
		t.Fatalf("expecting authorization error since client is not authorized to receive filtered events")
	}
}

func TestCCEvents(t *testing.T) {
	channelID := "mychannel"
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{FILTEREDBLOCKEVENT}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	ccID1 := "mycc1"
	ccID2 := "mycc2"
	ccFilter1 := "event1"
	ccFilter2 := "event.*"
	event1 := "event1"
	event2 := "event2"
	event3 := "event3"

	if _, _, err := eventService.RegisterChaincodeEvent("", ccFilter1); err == nil {
		t.Fatalf("expecting error registering for chaincode events without CC ID but got none")
	}
	if _, _, err := eventService.RegisterChaincodeEvent(ccID1, ""); err == nil {
		t.Fatalf("expecting error registering for chaincode events without event filter but got none")
	}
	if _, _, err := eventService.RegisterChaincodeEvent(ccID1, ".(xxx"); err == nil {
		t.Fatalf("expecting error registering for chaincode events with invalid (regular expression) event filter but got none")
	}
	reg1, _, err := eventService.RegisterChaincodeEvent(ccID1, ccFilter1)
	if err != nil {
		t.Fatalf("error registering for chaincode events: %s", err)
	}
	_, _, err = eventService.RegisterChaincodeEvent(ccID1, ccFilter1)
	if err == nil {
		t.Fatalf("expecting error registering multiple times for chaincode events: %s", err)
	}
	eventService.Unregister(reg1)

	reg1, eventch1, err := eventService.RegisterChaincodeEvent(ccID1, ccFilter1)
	if err != nil {
		t.Fatalf("error registering for block events: %s", err)
	}
	defer eventService.Unregister(reg1)

	reg2, eventch2, err := eventService.RegisterChaincodeEvent(ccID2, ccFilter2)
	if err != nil {
		t.Fatalf("error registering for chaincode events: %s", err)
	}
	defer eventService.Unregister(reg2)

	eventProducer.ProduceEvent(
		mocks.NewMockFilteredBlockEvent(
			channelID,
			mocks.NewMockFilteredTxWithCCEvent("txid1", ccID1, event1),
			mocks.NewMockFilteredTxWithCCEvent("txid2", ccID2, event2),
			mocks.NewMockFilteredTxWithCCEvent("txid3", ccID2, event3),
		),
	)

	numExpected := 3
	numReceived := 0
	done := false
	for !done {
		select {
		case event, ok := <-eventch1:
			if !ok {
				t.Fatalf("unexpected closed channel")
			} else {
				checkCCEvent(t, event, ccID1, event1)
				numReceived++
			}
		case event, ok := <-eventch2:
			if !ok {
				t.Fatalf("unexpected closed channel")
			} else {
				checkCCEvent(t, event, ccID2, event2, event3)
				numReceived++
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for [%d] CC events. Only received [%d]", numExpected, numReceived)
		}

		if numReceived == numExpected {
			break
		}
	}
}

func TestCCEventsUnauthorized(t *testing.T) {
	eventService, eventProducer, err := newServiceWithMockConn("mychannel", []EventType{}, DefaultOpts())
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}
	defer eventProducer.Close()
	defer eventService.Stop()

	if _, _, err := eventService.RegisterChaincodeEvent("ccid", ".*"); err == nil {
		t.Fatalf("expecting authorization error since client is not authorized to receive filtered events")
	}
}

// TestConcurrentEvents ensures that the channel event client is thread-safe
func TestConcurrentEvents(t *testing.T) {
	var numEvents uint = 1000
	channelID := "mychannel"
	opts := DefaultOpts()
	opts.EventConsumerBufferSize = numEvents * 4
	eventService, eventProducer, err := newServiceWithMockConn(channelID, []EventType{BLOCKEVENT, FILTEREDBLOCKEVENT}, opts)
	if err != nil {
		t.Fatalf("error creating channel event client: %s", err)
	}

	t.Run("Block Events", func(t *testing.T) {
		t.Parallel()
		if err := testConcurrentBlockEvents(channelID, numEvents, eventService, eventProducer); err != nil {
			t.Fatalf("error in testConcurrentBlockEvents: %s", err)
		}
	})
	t.Run("Filtered Block Events", func(t *testing.T) {
		t.Parallel()
		if err := testConcurrentFilteredBlockEvents(channelID, numEvents, eventService, eventProducer); err != nil {
			t.Fatalf("error in testConcurrentBlockEvents: %s", err)
		}
	})
	t.Run("Chaincode Events", func(t *testing.T) {
		t.Parallel()
		if err := testConcurrentCCEvents(channelID, numEvents, eventService, eventProducer); err != nil {
			t.Fatalf("error in testConcurrentBlockEvents: %s", err)
		}
	})
	t.Run("Tx Status Events", func(t *testing.T) {
		t.Parallel()
		if err := testConcurrentTxStatusEvents(channelID, numEvents, eventService, eventProducer); err != nil {
			t.Fatalf("error in testConcurrentBlockEvents: %s", err)
		}
	})
}

func testConcurrentBlockEvents(channelID string, numEvents uint, eventService eventapi.EventService, eventProducer *MockProducer) error {
	registration, eventch, err := eventService.RegisterBlockEvent()
	if err != nil {
		return errors.Errorf("error registering for block events: %s", err)
	}

	go func() {
		var i uint
		for i = 0; i < numEvents+10; i++ {
			eventProducer.ProduceEvent(mocks.NewMockBlockEvent(channelID))
		}
	}()

	var numReceived uint
	done := false

	for !done {
		select {
		case _, ok := <-eventch:
			if !ok {
				fmt.Printf("Block events channel was closed \n")
				done = true
			} else {
				numReceived++
				if numReceived == numEvents {
					// Unregister will close the event channel
					// and done will be set to true
					eventService.Unregister(registration)
				}
			}
		case <-time.After(5 * time.Second):
			if numReceived < numEvents {
				return errors.Errorf("Expected [%d] events but received [%d]", numEvents, numReceived)
			}
		}
	}

	return nil
}

func testConcurrentFilteredBlockEvents(channelID string, numEvents uint, eventService eventapi.EventService, conn *MockProducer) error {
	registration, eventch, err := eventService.RegisterFilteredBlockEvent()
	if err != nil {
		return errors.Errorf("error registering for filtered block events: %s", err)
	}
	defer eventService.Unregister(registration)

	var i uint
	for i = 0; i < numEvents; i++ {
		txID := fmt.Sprintf("txid_fb_%d", i)
		conn.ProduceEvent(mocks.NewMockFilteredBlockEvent(
			channelID,
			mocks.NewMockFilteredTx(txID, pb.TxValidationCode_VALID),
		))
	}

	var numReceived uint
	done := false

	for !done {
		select {
		case fbevent, ok := <-eventch:
			if !ok {
				fmt.Printf("Filtered block events channel was closed \n")
				done = true
			} else {
				if fbevent.FilteredBlock == nil {
					return errors.New("Expecting filtered block but got nil")
				}
				if fbevent.FilteredBlock.ChannelId != channelID {
					return errors.Errorf("Expecting channel [%s] but got [%s]", channelID, fbevent.FilteredBlock.ChannelId)
				}
				numReceived++
				if numReceived == numEvents {
					// Unregister will close the event channel and done will be set to true
					return nil
					// eventService.Unregister(registration)
				}
			}
		case <-time.After(5 * time.Second):
			if numReceived < numEvents {
				return errors.Errorf("Expected [%d] events but received [%d]", numEvents, numReceived)
			}
		}
	}

	return nil
}

func testConcurrentCCEvents(channelID string, numEvents uint, eventService eventapi.EventService, conn *MockProducer) error {
	ccID := "mycc1"
	ccFilter := "event.*"
	event1 := "event1"

	reg, eventch, err := eventService.RegisterChaincodeEvent(ccID, ccFilter)
	if err != nil {
		return errors.New("error registering for chaincode events")
	}

	var i uint
	for i = 0; i < numEvents+10; i++ {
		txID := fmt.Sprintf("txid_cc_%d", i)
		conn.ProduceEvent(
			mocks.NewMockFilteredBlockEvent(
				channelID,
				mocks.NewMockFilteredTxWithCCEvent(txID, ccID, event1),
			),
		)
	}

	var numReceived uint
	done := false
	for !done {
		select {
		case _, ok := <-eventch:
			if !ok {
				fmt.Printf("CC events channel was closed \n")
				done = true
			} else {
				numReceived++
			}
		case <-time.After(5 * time.Second):
			if numReceived < numEvents {
				return errors.Errorf("timed out waiting for [%d] CC events but received [%d]", numEvents, numReceived)
			}
		}

		if numReceived == numEvents {
			// Unregister will close the event channel and done will be set to true
			eventService.Unregister(reg)
		}
	}

	return nil
}

func testConcurrentTxStatusEvents(channelID string, numEvents uint, eventService eventapi.EventService, conn *MockProducer) error {
	var wg sync.WaitGroup

	wg.Add(int(numEvents))

	var errs []error
	var mutex sync.Mutex

	var receivedEvents uint
	for i := 0; i < int(numEvents); i++ {
		txID := fmt.Sprintf("txid_tx_%d", i)
		go func() {
			defer wg.Done()

			reg, eventch, err := eventService.RegisterTxStatusEvent(txID)
			if err != nil {
				mutex.Lock()
				errs = append(errs, errors.New("Error registering for TxStatus event"))
				mutex.Unlock()
				return
			}
			defer eventService.Unregister(reg)

			conn.ProduceEvent(
				mocks.NewMockFilteredBlockEvent(
					channelID,
					mocks.NewMockFilteredTx(txID, pb.TxValidationCode_VALID),
				),
			)

			select {
			case _, ok := <-eventch:
				mutex.Lock()
				if !ok {
					errs = append(errs, errors.New("unexpected closed channel"))
				} else {
					receivedEvents++
				}
				mutex.Unlock()
			case <-time.After(5 * time.Second):
				mutex.Lock()
				errs = append(errs, errors.New("timed out waiting for TxStatus event"))
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return errors.Errorf("Received %d events and %d errors. First error %s\n", receivedEvents, len(errs), errs[0])
	}
	return nil
}

func listenEvents(blockch <-chan *eventapi.BlockEvent, ccch <-chan *eventapi.CCEvent, waitDuration time.Duration, numEventsCh chan EventsReceived, expectedBlockEvents NumBlockEvents, expectedCCEvents NumCCEvents) {
	var numBlockEventsReceived NumBlockEvents
	var numCCEventsReceived NumCCEvents

	for {
		select {
		case _, ok := <-blockch:
			if ok {
				numBlockEventsReceived++
			} else {
				// The channel was closed by the event client. Make a new channel so
				// that we don't get into a tight loop
				blockch = make(chan *eventapi.BlockEvent)
			}
		case _, ok := <-ccch:
			if ok {
				numCCEventsReceived++
			} else {
				// The channel was closed by the event client. Make a new channel so
				// that we don't get into a tight loop
				ccch = make(chan *eventapi.CCEvent)
			}
		case <-time.After(waitDuration):
			numEventsCh <- EventsReceived{BlockEvents: numBlockEventsReceived, CCEvents: numCCEventsReceived}
			return
		}
		if numBlockEventsReceived >= expectedBlockEvents && numCCEventsReceived >= expectedCCEvents {
			numEventsCh <- EventsReceived{BlockEvents: numBlockEventsReceived, CCEvents: numCCEventsReceived}
			return
		}
	}
}

func newServiceWithMockConn(channelID string, eventTypes []EventType, opts *Opts, prodOpts ...MockProducerOpt) (*EventService, *MockProducer, error) {
	service := NewService(opts, eventTypes)
	eventProducer := NewMockProducer(prodOpts...)
	service.Start(eventProducer)
	return service, eventProducer, nil
}

func checkTxStatusEvent(t *testing.T, event *eventapi.TxStatusEvent, expectedTxID string, expectedCode pb.TxValidationCode) {
	if event.TxID != expectedTxID {
		t.Fatalf("expecting event for TxID [%s] but received event for TxID [%s]", expectedTxID, event.TxID)
	}
	if event.TxValidationCode != expectedCode {
		t.Fatalf("expecting TxValidationCode [%s] but received [%s]", expectedCode, event.TxValidationCode)
	}
}

func checkCCEvent(t *testing.T, event *eventapi.CCEvent, expectedCCID string, expectedEventNames ...string) {
	if event.ChaincodeID != expectedCCID {
		t.Fatalf("expecting event for CC [%s] but received event for CC [%s]", expectedCCID, event.ChaincodeID)
	}
	found := false
	for _, eventName := range expectedEventNames {
		if event.EventName == eventName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expecting one of [%v] but received [%s]", expectedEventNames, event.EventName)
	}
}
