/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"sync"
	"time"
)

// Operation is the operation being performed
type Operation string

// Result is the result to take for a given operation
type Result string

const (
	// SucceedResult indicates that the operation should succeed
	SucceedResult Result = "succeed"

	// FailResult indicates that the operation should fail
	FailResult Result = "fail"

	// NoOpResult indicates that the operation should be ignored (i.e. just do nothing)
	// This should result in the client timing out waiting for a response.
	NoOpResult Result = "no-op"
)

// MockProducer is a fake connection used for unit testing
type MockProducer struct {
	Operations       OperationMap
	AuthorizedEvents []EventType
	rcvch            chan interface{}
	eventChannels    []chan<- interface{}
	mutex            sync.RWMutex
}

// NewMockProducer returns a new MockProducer using the given options
func NewMockProducer(opts ...MockProducerOpt) *MockProducer {
	conn := &MockProducer{
		rcvch:      make(chan interface{}, 100),
		Operations: make(map[Operation]ResultDesc),
	}
	for _, opt := range opts {
		opt.Apply(conn)
	}
	go conn.listen()
	return conn
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
				logger.Errorf("Timed out sending event.")
			}
		}
	}
}

// Close closes the event producer
func (c *MockProducer) Close() {
	if c.rcvch != nil {
		close(c.rcvch)
		c.rcvch = nil
	}
}

// Register registers an event channel with the event relay. The event channel
// will be relayed events from the event hub.
func (c *MockProducer) Register(eventch chan<- interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.eventChannels = append(c.eventChannels, eventch)
}

// ProduceEvent allows a unit test to inject an event
func (c *MockProducer) ProduceEvent(event interface{}) {
	go func() {
		c.rcvch <- event
	}()
}

func asStrings(authEvents []EventType) []string {
	ret := make([]string, len(authEvents))
	for i, et := range authEvents {
		ret[i] = string(et)
	}
	return ret
}

// MockProducerProviderFactory creates various mock Connection Providers
type MockProducerProviderFactory struct {
	connection *MockProducer
	mtx        sync.RWMutex
}

// NewMockProducerProviderFactory returns a new producer-provider factory
func NewMockProducerProviderFactory() *MockProducerProviderFactory {
	return &MockProducerProviderFactory{}
}

// Connection returns a mock Connection
func (cp *MockProducerProviderFactory) Connection() *MockProducer {
	cp.mtx.RLock()
	defer cp.mtx.RUnlock()
	return cp.connection
}

// ResultDesc describes the result of an operation and optional error string
type ResultDesc struct {
	Result Result
	ErrMsg string
}

// OperationMap maps an Operation to a ResultDesc
type OperationMap map[Operation]ResultDesc

// MockProducerOpt applies an option to a MockProducer
type MockProducerOpt interface {
	// Apply applies the option to the MockProducer
	Apply(conn *MockProducer)
}

// OperationResult contains the result of an operation
type OperationResult struct {
	Operation  Operation
	Result     Result
	ErrMessage string
}

// NewResult returns a new OperationResult
func NewResult(operation Operation, result Result, errMsg ...string) *OperationResult {
	msg := ""
	if len(errMsg) > 0 {
		msg = errMsg[0]
	}
	return &OperationResult{
		Operation:  operation,
		Result:     result,
		ErrMessage: msg,
	}
}

// OperationResultsOpt is a connection option that indicates what to do for each operation
type OperationResultsOpt struct {
	Operations OperationMap
}

// Apply applies the option to the MockProducer
func (o *OperationResultsOpt) Apply(conn *MockProducer) {
	conn.Operations = o.Operations
}

// NewResultsOpt returns a new OperationResultsOpt
func NewResultsOpt(funcResults ...*OperationResult) *OperationResultsOpt {
	opt := &OperationResultsOpt{Operations: make(map[Operation]ResultDesc)}
	for _, fr := range funcResults {
		opt.Operations[fr.Operation] = ResultDesc{Result: fr.Result, ErrMsg: fr.ErrMessage}
	}
	return opt
}
