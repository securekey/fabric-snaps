/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package relay

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hyperledger/fabric/events/consumer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/eventserver/pkg/channelutil"
	"github.com/securekey/fabric-snaps/eventserver/pkg/mocks"
)

func TestEventRelayInvalidOpts(t *testing.T) {
	if _, err := New("", "localhost:7053", DefaultOpts()); err == nil {
		t.Fatalf("expecting error for empty channel ID but got none")
	}
	if _, err := New("ch1", "", DefaultOpts()); err == nil {
		t.Fatalf("expecting error for empty event hub address but got none")
	}
}

func TestEventRelay(t *testing.T) {
	channelID1 := "ch1"

	var mockeh *mocks.MockEventHub
	opts := MockOpts(func(channelID string, address string, regTimeout time.Duration, adapter consumer.EventAdapter) (EventHub, error) {
		mockeh = &mocks.MockEventHub{
			Adapter:          adapter,
			NumStartFailures: 1, // Simulate a failed startup the first time
		}
		return mockeh, nil
	})

	eventRelay, err := New(channelID1, "localhost:7053", opts)
	if err != nil {
		t.Fatalf("error starting event relay: %s", err)
	}

	eventch1 := make(chan interface{}, 10)
	eventRelay.Register(eventch1)
	eventch2 := make(chan interface{}, 10)
	eventRelay.Register(eventch2)

	eventRelay.Start()

	// Wait for event hub to connect
	time.Sleep(200 * time.Millisecond)

	// Attempt to send an invalid BlockEvent (should be ignored)
	mockeh.ProduceEvent(mocks.NewMockBlockEvent(
		"", // No channel ID
	))

	// Attempt to send an invalid FilteredBlockEvent (should be ignored)
	mockeh.ProduceEvent(mocks.NewMockFilteredBlockEvent(
		"", // No channel ID
		mocks.NewMockFilteredTx("txid", pb.TxValidationCode_VALID),
	))

	// Send a valid BlockEvent (should be relayed)
	mockeh.ProduceEvent(mocks.NewMockBlockEvent(channelID1))

	numExpected := 2
	numReceived := 0

	for {
		select {
		case event, ok := <-eventch1:
			if !ok {
				t.Fatalf("event channel1 disconnected")
			}
			fmt.Printf("*** Received event on eventch1: %s\n", event)
			checkEvent(t, event, channelID1)
			numReceived++
		case event, ok := <-eventch2:
			if !ok {
				t.Fatalf("event channel2 disconnected")
			}
			fmt.Printf("*** Received event on eventch2: %s\n", event)
			checkEvent(t, event, channelID1)
			numReceived++
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event")
		}
		if numReceived == numExpected {
			break
		}
	}

	// Disconnect the event hub. Expecting that the event relay will reconnect to a new event hub.
	mockeh.Disconnect(errors.New("testing disconnect"))

	time.Sleep(2 * time.Second)

	mockeh.ProduceEvent(mocks.NewMockFilteredBlockEvent(
		channelID1,
		mocks.NewMockFilteredTx("txid", pb.TxValidationCode_VALID),
	))

	numReceived = 0

	for {
		select {
		case event, ok := <-eventch1:
			if !ok {
				t.Fatalf("event channel1 disconnected")
			}
			fmt.Printf("*** Received event on eventch1: %s\n", event)
			checkEvent(t, event, channelID1)
			numReceived++
		case event, ok := <-eventch2:
			if !ok {
				t.Fatalf("event channel2 disconnected")
			}
			fmt.Printf("*** Received event on eventch2: %s\n", event)
			checkEvent(t, event, channelID1)
			numReceived++
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event")
		}
		if numReceived == numExpected {
			break
		}
	}
}

func TestEventRelayBufferFull(t *testing.T) {
	channelID1 := "ch1"

	var mockeh *mocks.MockEventHub
	opts := MockOpts(func(channelID string, address string, regTimeout time.Duration, adapter consumer.EventAdapter) (EventHub, error) {
		mockeh = &mocks.MockEventHub{
			Adapter: adapter,
		}
		return mockeh, nil
	})

	eventRelay, err := New(channelID1, "localhost:7053", opts)
	if err != nil {
		t.Fatalf("error starting event relay: %s", err)
	}

	eventch1 := make(chan interface{})
	eventRelay.Register(eventch1)

	eventRelay.Start()

	// Wait for event hub to connect
	time.Sleep(200 * time.Millisecond)

	// Send two events - only the first should be processed and the second should be immediately rejected
	// since the channel size is set to 1
	mockeh.ProduceEvent(mocks.NewMockBlockEvent(channelID1))
	mockeh.ProduceEvent(mocks.NewMockBlockEvent(channelID1))

	numExpected := 1
	numReceived := 0

	done := false

	for !done {
		select {
		case event, ok := <-eventch1:
			if !ok {
				t.Fatalf("event channel1 disconnected")
			}
			fmt.Printf("*** Received event on eventch1: %s\n", event)
			numReceived++
			time.Sleep(300 * time.Millisecond)
		case <-time.After(2 * time.Second):
			if numReceived > numExpected {
				t.Fatalf("expected %d events but received %d", numExpected, numReceived)
			}
			done = true
		}
	}
}

func TestEventRelayTimeout(t *testing.T) {
	channelID1 := "ch1"

	var mockeh *mocks.MockEventHub
	opts := MockOpts(func(channelID string, address string, regTimeout time.Duration, adapter consumer.EventAdapter) (EventHub, error) {
		mockeh = &mocks.MockEventHub{
			Adapter: adapter,
		}
		return mockeh, nil
	})
	opts.RelayTimeout = 250 * time.Millisecond

	eventRelay, err := New(channelID1, "localhost:7053", opts)
	if err != nil {
		t.Fatalf("error starting event relay: %s", err)
	}

	eventch1 := make(chan interface{})
	eventRelay.Register(eventch1)

	eventRelay.Start()

	// Wait for event hub to connect
	time.Sleep(200 * time.Millisecond)

	// Send two events - only the first should be processed and the second should
	// time out since the event channel size is set to 1 and we're artificially be
	// delaying the processing
	mockeh.ProduceEvent(mocks.NewMockBlockEvent(channelID1))
	mockeh.ProduceEvent(mocks.NewMockBlockEvent(channelID1))

	numExpected := 1
	numReceived := 0

	done := false

	for !done {
		select {
		case event, ok := <-eventch1:
			if !ok {
				t.Fatalf("event channel1 disconnected")
			}
			fmt.Printf("*** Received event on eventch1: %s\n", event)
			numReceived++
			time.Sleep(300 * time.Millisecond)
		case <-time.After(2 * time.Second):
			if numReceived > numExpected {
				t.Fatalf("expected %d events but received %d", numExpected, numReceived)
			}
			done = true
		}
	}
}

func checkEvent(t *testing.T, event interface{}, channelID string) {
	switch evt := event.(type) {
	case *pb.Event:
		if chID, _ := channelutil.ChannelIDFromEvent(evt); chID != channelID {
			t.Fatalf("expecting channel %s but got %s", channelID, chID)
		}
	default:
		t.Fatalf("expecting Event but got %s", reflect.TypeOf(event))
	}
}

func MockOpts(mockEventHubProvider EventHubProvider) *Opts {
	opts := DefaultOpts()
	opts.eventHubProvider = mockEventHubProvider
	return opts
}
