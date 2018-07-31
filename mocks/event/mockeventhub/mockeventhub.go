/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockeventhub

import (
	"github.com/hyperledger/fabric/events/consumer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

var logger *logging.Logger

// MockEventHub mocks out the Event Hub
type MockEventHub struct {
	Adapter          consumer.EventAdapter
	Interests        []*pb.Interest
	NumStartFailures int
}

// New returns a new MockEventHub
func New(adapter consumer.EventAdapter) *MockEventHub {
	return &MockEventHub{Adapter: adapter}
}

// Start implements the Start() method on the relay.EventHub interface
func (m *MockEventHub) Start() error {
	if m.NumStartFailures > 0 {
		m.NumStartFailures--
		return errors.Errorf("purposefully failing to start mock event hub")
	}

	interests, err := m.Adapter.GetInterestedEvents()
	if err != nil {
		return errors.Wrap(err, "error getting interested events")
	}
	m.Interests = interests
	return nil
}

// ProduceEvent produces a new event, which is sent to the adapter
func (m *MockEventHub) ProduceEvent(event *pb.Event) {
	go func() {
		_, err := m.Adapter.Recv(event)
		if err != nil {
			logger.Error("Error occurred in producing new events to be sent to adaptor")
		}
	}()
}

// Disconnect simulates a disconnect
func (m *MockEventHub) Disconnect(err error) {
	go func() {
		m.Adapter.Disconnected(err)
	}()
}
