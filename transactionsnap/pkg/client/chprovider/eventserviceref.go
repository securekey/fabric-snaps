/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package chprovider

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"
)

type eventClientProvider func() (fab.EventClient, error)
type eventClientCloser func(client fab.EventClient)

// EventClientRef holds a reference to the event client and manages its lifecycle.
// The EventClientRef implements all of the functions of fab.EventService, so the
// EventClientRef may be used wherever an EventService is required.
type EventClientRef struct {
	ref      *lazyref.Reference
	provider eventClientProvider
}

// NewEventClientRef returns a new EventClientRef
func NewEventClientRef(evtClientProvider eventClientProvider) *EventClientRef {
	clientRef := &EventClientRef{
		provider: evtClientProvider,
	}

	clientRef.ref = lazyref.New(
		clientRef.initializer(),
		lazyref.WithFinalizer(clientRef.finalizer()),
	)

	return clientRef
}

// Close immediately closes the connection.
func (ref *EventClientRef) Close() {
	ref.ref.Close()
}

// RegisterBlockEvent registers for block events.
func (ref *EventClientRef) RegisterBlockEvent(filter ...fab.BlockFilter) (fab.Registration, <-chan *fab.BlockEvent, error) {
	service, err := ref.get()
	if err != nil {
		return nil, nil, err
	}
	return service.RegisterBlockEvent(filter...)
}

// RegisterFilteredBlockEvent registers for filtered block events.
func (ref *EventClientRef) RegisterFilteredBlockEvent() (fab.Registration, <-chan *fab.FilteredBlockEvent, error) {
	service, err := ref.get()
	if err != nil {
		return nil, nil, err
	}
	return service.RegisterFilteredBlockEvent()
}

// RegisterChaincodeEvent registers for chaincode events.
func (ref *EventClientRef) RegisterChaincodeEvent(ccID, eventFilter string) (fab.Registration, <-chan *fab.CCEvent, error) {
	service, err := ref.get()
	if err != nil {
		return nil, nil, err
	}
	return service.RegisterChaincodeEvent(ccID, eventFilter)
}

// RegisterTxStatusEvent registers for transaction status events.
func (ref *EventClientRef) RegisterTxStatusEvent(txID string) (fab.Registration, <-chan *fab.TxStatusEvent, error) {
	service, err := ref.get()
	if err != nil {
		return nil, nil, err
	}
	return service.RegisterTxStatusEvent(txID)
}

// Unregister removes the given registration and closes the event channel.
func (ref *EventClientRef) Unregister(reg fab.Registration) {
	if service, err := ref.get(); err != nil {
		logger.Warnf("Error unregistering event registration: %s", err)
	} else {
		service.Unregister(reg)
	}
}

func (ref *EventClientRef) get() (fab.EventService, error) {
	service, err := ref.ref.Get()
	if err != nil {
		return nil, err
	}
	return service.(fab.EventService), nil
}

func (ref *EventClientRef) initializer() lazyref.Initializer {
	return func() (interface{}, error) {
		eventClient, err := ref.provider()
		if err != nil {
			return nil, err
		}
		if err := eventClient.Connect(); err != nil {
			return nil, err
		}
		return eventClient, nil
	}
}

func (ref *EventClientRef) finalizer() lazyref.Finalizer {
	return func(client interface{}) {
		client.(fab.EventClient).Close()
	}
}
