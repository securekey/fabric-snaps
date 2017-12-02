/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"testing"
	"time"

	eventrelay "github.com/securekey/fabric-snaps/eventrelay/pkg/relay"
	localservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/mocks/event/mockevent"
	"github.com/securekey/fabric-snaps/mocks/event/mockeventhub"
	"google.golang.org/grpc"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/events/consumer"
)

func TestEventSnap(t *testing.T) {
	channelID1 := "ch1"
	channelID2 := "ch2"

	eventsnap := &eventSnap{
		pserver: grpc.NewServer(),
	}

	stub := shim.NewMockStub("eventsnap", eventsnap)

	// Invalid options
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid2", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in init since no event hub address was specified but got OK")
	}

	mockEventHubs := make(map[string]*mockeventhub.MockEventHub)

	eventsnap = &eventSnap{
		pserver: grpc.NewServer(),
		eropts: eventrelay.MockOpts(func(channelID string, address string, regTimeout time.Duration, adapter consumer.EventAdapter) (eventrelay.EventHub, error) {
			fmt.Printf("Creating mock event hub for channel %s\n", channelID)
			mockeh := mockeventhub.New(adapter)
			mockEventHubs[channelID] = mockeh
			return mockeh, nil
		}),
		configPath: "./sampleconfig",
	}

	stub = shim.NewMockStub("eventsnap", eventsnap)

	// Initialize with no channel
	if resp := stub.MockInit("txid1", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Initialize with channel
	stub.ChannelID = channelID1
	if resp := stub.MockInit("txid3", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Another channel
	stub.ChannelID = channelID2
	if resp := stub.MockInit("txid4", nil); resp.Status != shim.OK {
		t.Fatalf("Error in init: %s", resp.GetMessage())
	}

	// Init again on same channel
	if resp := stub.MockInit("txid5", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in init since init was already called for the same channel but got OK")
	}

	// Invoke should return error
	if resp := stub.MockInvoke("txid6", nil); resp.Status == shim.OK {
		t.Fatalf("Expecting error in invoke since invoke is not supported but got OK")
	}

	eventService1 := localservice.Get(channelID1)
	if eventService1 == nil {
		t.Fatalf("Expecting local event service for %s to be registered but got nil", channelID1)
	}
	eventService2 := localservice.Get(channelID2)
	if eventService2 == nil {
		t.Fatalf("Expecting local event service for %s to be registered but got nil", channelID2)
	}
	reg1, bEventCh1, err := eventService1.RegisterBlockEvent()
	if err != nil {
		t.Fatalf("Error in RegisterBlockEvent on event service channel %s: %s", channelID1, err)
	}
	defer eventService1.Unregister(reg1)

	reg2, bEventCh2, err := eventService2.RegisterBlockEvent()
	if err != nil {
		t.Fatalf("Error in RegisterBlockEvent on event service channel %s: %s", channelID2, err)
	}
	defer eventService2.Unregister(reg2)

	mockEventHubs[channelID1].ProduceEvent(mockevent.NewBlockEvent(channelID1))
	mockEventHubs[channelID2].ProduceEvent(mockevent.NewBlockEvent(channelID2))

	numExpected := 2
	numReceived := 0
	done := false

	for !done {
		select {
		case event, ok := <-bEventCh1:
			if !ok {
				t.Fatalf("event channel1 disconnected")
			}
			fmt.Printf("*** Received event on bEventCh1: %v\n", event)
			numReceived++
		case event, ok := <-bEventCh2:
			if !ok {
				t.Fatalf("event channel2 disconnected")
			}
			fmt.Printf("*** Received event on bEventCh2: %s\n", event)
			numReceived++
		case <-time.After(2 * time.Second):
			if numReceived != numExpected {
				t.Fatalf("Expecting %d events but received %d", numExpected, numReceived)
			} else {
				done = true
			}
		}
	}
}
