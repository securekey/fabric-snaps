/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// BlockEvent contains the data for the block event
type BlockEvent struct {
	Block *cb.Block
}

// FilteredBlockEvent contains the data for a filtered block event
type FilteredBlockEvent struct {
	FilteredBlock *pb.FilteredBlock
}

// TxStatusEvent contains the data for a transaction status event
type TxStatusEvent struct {
	TxID             string
	TxValidationCode pb.TxValidationCode
}

// CCEvent contains the data for a chaincode event
type CCEvent struct {
	TxID        string
	ChaincodeID string
	EventName   string
}

// Registration is a handle that is returned from a successful RegisterXXXEvent.
// This handle should be used in Unregister in order to unregister the event.
type Registration interface{}

// RegistrationResponse is the response that is returned for any register/unregister event.
// For a successful registration, the registration handle is set. This handle should be used
// in a subsequent Unregister request. If an error occurs then the error is set.
type RegistrationResponse struct {
	// Reg is a handle to the registration
	Reg Registration

	// Err contains the error if registration is unsuccessful
	Err error
}

// FilteredEventService is a service that receives events such as filtered block,
// chaincode, and transaction status events.
type FilteredEventService interface {
	// RegisterFilteredBlockEvent registers for filtered block events. If the client is not authorized to receive
	// filtered block events then an error is returned.
	// - Returns the registration and a channel that is used to receive events
	RegisterFilteredBlockEvent() (Registration, <-chan *FilteredBlockEvent, error)

	// RegisterChaincodeEvent registers for chaincode events. If the client is not authorized to receive
	// chaincode events then an error is returned.
	// - ccID is the chaincode ID for which events are to be received
	// - eventFilter is the chaincode event filter (regular expression) for which events are to be received
	RegisterChaincodeEvent(ccID, eventFilter string) (Registration, <-chan *CCEvent, error)

	// RegisterTxStatusEvent registers for transaction status events. If the client is not authorized to receive
	// transaction status events then an error is returned.
	// - txID is the transaction ID for which events are to be received
	RegisterTxStatusEvent(txID string) (Registration, <-chan *TxStatusEvent, error)

	// Unregister unregisters the given registration.
	// - reg is the registration handle that was returned from one of the RegisterXXX functions
	Unregister(reg Registration)
}

// EventService is a service that receives events such as block, filtered block,
// chaincode, and transaction status events.
type EventService interface {
	FilteredEventService

	// RegisterBlockEvent registers for block events. If the client is not authorized to receive
	// block events then an error is returned.
	RegisterBlockEvent() (Registration, <-chan *BlockEvent, error)
}
