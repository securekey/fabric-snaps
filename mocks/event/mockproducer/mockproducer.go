/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockproducer

import (
	"fmt"
	"sync"
	"time"
)

// MockProducer produces events for unit testing
type MockProducer struct {
	rcvch         chan interface{}
	eventChannels []chan<- interface{}
	mutex         sync.RWMutex
}

// New returns a new MockProducer
func New() *MockProducer {
	conn := &MockProducer{
		rcvch: make(chan interface{}, 100),
	}
	go conn.listen()
	return conn
}

// Close closes the event producer
func (c *MockProducer) Close() {
	if c.rcvch != nil {
		close(c.rcvch)
		c.rcvch = nil
	}
}

// Register registers an event channel with the event relay
func (c *MockProducer) Register(eventch chan<- interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.eventChannels = append(c.eventChannels, eventch)
}

// ProduceEvent allows a unit test to send an event
func (c *MockProducer) ProduceEvent(event interface{}) {
	go func() {
		c.rcvch <- event
	}()
}

func (c *MockProducer) listen() {
	for {
		event, ok := <-c.rcvch
		if !ok {
			return
		}

		c.mutex.RLock()
		defer c.mutex.RUnlock()

		for _, eventch := range c.eventChannels {
			select {
			case eventch <- event:
			case <-time.After(time.Second):
				fmt.Printf("***** Timed out sending event.")
			}
		}
	}
}
