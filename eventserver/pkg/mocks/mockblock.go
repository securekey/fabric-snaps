/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// NewMockBlockEvent returns a new mock block event initialized with the given channel
func NewMockBlockEvent(channelID string) *pb.Event {
	return &pb.Event{
		Creator:   []byte("some-id"),
		Timestamp: &timestamp.Timestamp{Seconds: 1000},
		Event: &pb.Event_Block{
			Block: NewMockBlock(channelID),
		},
	}
}

// NewMockBlock returns a new mock block initialized with the given channel
func NewMockBlock(channelID string) *cb.Block {
	channelHeader := &cb.ChannelHeader{
		ChannelId: channelID,
	}
	channelHeaderBytes, _ := proto.Marshal(channelHeader)
	payload := &cb.Payload{
		Header: &cb.Header{
			ChannelHeader: channelHeaderBytes,
		},
	}
	payloadBytes, _ := proto.Marshal(payload)
	env := &cb.Envelope{
		Payload: payloadBytes,
	}
	envBytes, _ := proto.Marshal(env)

	return &cb.Block{
		Data: &cb.BlockData{
			Data: [][]byte{envBytes},
		},
	}
}

// NewMockFilteredBlockEvent returns a new mock filtered block event initialized with the given channel
// and filtered transactions
func NewMockFilteredBlockEvent(channelID string, filteredTx ...*pb.FilteredTransaction) *pb.Event {
	return &pb.Event{
		Creator:   []byte("some-id"),
		Timestamp: &timestamp.Timestamp{Seconds: 1000},
		Event: &pb.Event_FilteredBlock{
			FilteredBlock: NewMockFilteredBlock(channelID, filteredTx...),
		},
	}
}

// NewMockFilteredBlock returns a new mock filtered block initialized with the given channel
// and filtered transactions
func NewMockFilteredBlock(channelID string, filteredTx ...*pb.FilteredTransaction) *pb.FilteredBlock {
	return &pb.FilteredBlock{
		ChannelId:  channelID,
		FilteredTx: filteredTx,
	}
}

// NewMockFilteredTx returns a new mock filtered transaction
func NewMockFilteredTx(txID string, txValidationCode pb.TxValidationCode) *pb.FilteredTransaction {
	return &pb.FilteredTransaction{
		Txid:             txID,
		TxValidationCode: txValidationCode,
	}
}

// NewMockFilteredTxWithCCEvent returns a new mock filtered transaction
// with the given chaincode event
func NewMockFilteredTxWithCCEvent(txID, ccID, event string) *pb.FilteredTransaction {
	return &pb.FilteredTransaction{
		Txid: txID,
		FilteredAction: []*pb.FilteredAction{
			&pb.FilteredAction{
				CcEvent: &pb.ChaincodeEvent{
					ChaincodeId: ccID,
					EventName:   event,
					TxId:        txID,
				},
			},
		},
	}
}
