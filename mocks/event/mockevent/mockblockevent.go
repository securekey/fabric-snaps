/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mockevent

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	cb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/op/go-logging"
)

var logger *logging.Logger

// NewBlockEvent returns a new mock block event initialized with the given channel
func NewBlockEvent(channelID string) *pb.Event {
	return &pb.Event{
		Creator:   []byte("some-id"),
		Timestamp: &timestamp.Timestamp{Seconds: 1000},
		Event: &pb.Event_Block{
			Block: NewBlock(channelID),
		},
	}
}

// NewBlock returns a new mock block initialized with the given channel
func NewBlock(channelID string) *cb.Block {
	channelHeader := &cb.ChannelHeader{
		ChannelId: channelID,
	}
	channelHeaderBytes, err := proto.Marshal(channelHeader)
	if err != nil {
		logger.Panicf("Error creating new mock block: %s", err)
	}
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

// NewFilteredBlockEvent returns a new mock filtered block event initialized with the given channel
// and filtered transactions
func NewFilteredBlockEvent(channelID string, filteredTx ...*pb.FilteredTransaction) *pb.Event {
	return &pb.Event{
		Creator:   []byte("some-id"),
		Timestamp: &timestamp.Timestamp{Seconds: 1000},
		Event: &pb.Event_FilteredBlock{
			FilteredBlock: NewFilteredBlock(channelID, filteredTx...),
		},
	}
}

// NewFilteredBlock returns a new mock filtered block initialized with the given channel
// and filtered transactions
func NewFilteredBlock(channelID string, filteredTx ...*pb.FilteredTransaction) *pb.FilteredBlock {
	return &pb.FilteredBlock{
		ChannelId:            channelID,
		FilteredTransactions: filteredTx,
	}
}

// NewFilteredTx returns a new mock filtered transaction
func NewFilteredTx(txID string, txValidationCode pb.TxValidationCode) *pb.FilteredTransaction {
	return &pb.FilteredTransaction{
		Txid:             txID,
		TxValidationCode: txValidationCode,
	}
}

// NewFilteredTxWithCCEvent returns a new mock filtered transaction
// with the given chaincode event
func NewFilteredTxWithCCEvent(txID, ccID, event string) *pb.FilteredTransaction {
	return &pb.FilteredTransaction{
		Txid: txID,
		Data: &pb.FilteredTransaction_TransactionActions{
			TransactionActions: &pb.FilteredTransactionActions{
				ChaincodeActions: []*pb.FilteredChaincodeAction{
					{
						ChaincodeEvent: &pb.ChaincodeEvent{
							ChaincodeId: ccID,
							EventName:   event,
							TxId:        txID,
						},
					},
				},
			},
		},
	}
}
