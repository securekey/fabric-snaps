/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"regexp"

	pb "github.com/hyperledger/fabric/protos/peer"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

// EventType is the type of the event for which to register (e.g. BLOCK or FILTEREDBLOCK)
type EventType string

// Event is an event that's sent to the dispatcher. This includes client registration
// requests or events that come from an event producer.
type Event interface{}

// RegisterEvent is the base for all registration events.
type RegisterEvent struct {
	RespCh chan<- *eventapi.RegistrationResponse
}

type registerBlockEvent struct {
	RegisterEvent
	reg *blockRegistration
}

type registerFilteredBlockEvent struct {
	RegisterEvent
	reg *filteredBlockRegistration
}

type registerCCEvent struct {
	RegisterEvent
	reg *ccRegistration
}

type registerTxStatusEvent struct {
	RegisterEvent
	reg *txRegistration
}

type unregisterEvent struct {
	reg eventapi.Registration
}

func newRegisterBlockEvent(eventch chan<- *eventapi.BlockEvent, respch chan<- *eventapi.RegistrationResponse) *registerBlockEvent {
	return &registerBlockEvent{
		reg:           &blockRegistration{eventch: eventch},
		RegisterEvent: RegisterEvent{RespCh: respch},
	}
}

func newRegisterFilteredBlockEvent(eventch chan<- *eventapi.FilteredBlockEvent, respch chan<- *eventapi.RegistrationResponse) *registerFilteredBlockEvent {
	return &registerFilteredBlockEvent{
		reg:           &filteredBlockRegistration{eventch: eventch},
		RegisterEvent: RegisterEvent{RespCh: respch},
	}
}

func newUnregisterEvent(reg eventapi.Registration) *unregisterEvent {
	return &unregisterEvent{
		reg: reg,
	}
}

func newRegisterCCEvent(ccID, eventFilter string, eventRegExp *regexp.Regexp, eventch chan<- *eventapi.CCEvent, respch chan<- *eventapi.RegistrationResponse) *registerCCEvent {
	return &registerCCEvent{
		reg: &ccRegistration{
			ccID:        ccID,
			eventFilter: eventFilter,
			eventRegExp: eventRegExp,
			eventch:     eventch,
		},
		RegisterEvent: RegisterEvent{RespCh: respch},
	}
}

func newRegisterTxStatusEvent(txID string, eventch chan<- *eventapi.TxStatusEvent, respch chan<- *eventapi.RegistrationResponse) *registerTxStatusEvent {
	return &registerTxStatusEvent{
		reg:           &txRegistration{txID: txID, eventch: eventch},
		RegisterEvent: RegisterEvent{RespCh: respch},
	}
}

func newCCEvent(chaincodeID, eventName, txID string) *eventapi.CCEvent {
	return &eventapi.CCEvent{
		ChaincodeID: chaincodeID,
		EventName:   eventName,
		TxID:        txID,
	}
}

func newTxStatusEvent(txID string, txValidationCode pb.TxValidationCode) *eventapi.TxStatusEvent {
	return &eventapi.TxStatusEvent{
		TxID:             txID,
		TxValidationCode: txValidationCode,
	}
}
