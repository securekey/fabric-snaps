/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package channelutil

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	cb "github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/mocks/event/mockevent"
)

func TestChannelIDFromBlock(t *testing.T) {
	channelID1 := "ch1"

	channelID, err := ChannelIDFromBlock(mockevent.NewBlock(channelID1))
	if err != nil {
		t.Fatalf("error returned from ChannelIDFromBlock: %s", err)
	}
	if channelID != channelID1 {
		t.Fatalf("expecting channel ID %s but got %s", channelID1, channelID)
	}
}

func TestChannelIDFromFilteredBlock(t *testing.T) {
	channelID1 := "ch1"

	channelID, err := ChannelIDFromFilteredBlock(mockevent.NewFilteredBlock(channelID1))
	if err != nil {
		t.Fatalf("error returned from ChannelIDFromFilteredBlock: %s", err)
	}
	if channelID != channelID1 {
		t.Fatalf("expecting channel ID %s but got %s", channelID1, channelID)
	}
}

func TestChannelIDFromEvent(t *testing.T) {
	channelID1 := "ch1"

	channelID, err := ChannelIDFromEvent(mockevent.NewFilteredBlockEvent(channelID1))
	if err != nil {
		t.Fatalf("error returned from ChannelIDFromEvent with filtered block event: %s", err)
	}
	if channelID != channelID1 {
		t.Fatalf("expecting channel ID %s but got %s", channelID1, channelID)
	}

	channelID, err = ChannelIDFromEvent(mockevent.NewBlockEvent(channelID1))
	if err != nil {
		t.Fatalf("error returned from ChannelIDFromEvent with block event: %s", err)
	}
	if channelID != channelID1 {
		t.Fatalf("expecting channel ID %s but got %s", channelID1, channelID)
	}
}

func TestChannelIDFromInvalidEventType(t *testing.T) {
	event := &pb.Event{
		Creator:   []byte("some-id"),
		Timestamp: &timestamp.Timestamp{Seconds: 1000},
		Event:     &pb.Event_Rejection{},
	}

	_, err := ChannelIDFromEvent(event)
	if err == nil {
		t.Fatalf("expecting error from ChannelIDFromEvent for invalid event type but got none")
	}
}

func TestChannelIDFromInvalidBlockEvent(t *testing.T) {
	event := &pb.Event{
		Creator:   []byte("some-id"),
		Timestamp: &timestamp.Timestamp{Seconds: 1000},
		Event:     &pb.Event_Block{},
	}

	// No block
	_, err := ChannelIDFromEvent(event)
	if err == nil {
		t.Fatalf("expecting error from ChannelIDFromEvent for invalid block event but got none")
	}

	// Invalid block
	block := &cb.Block{}
	event.Event.(*pb.Event_Block).Block = block
	_, err = ChannelIDFromEvent(event)
	if err == nil {
		t.Fatalf("expecting error from ChannelIDFromEvent for invalid event type but got none")
	}

	// Invalid envelope in block
	block.Data = &cb.BlockData{}
	env := &cb.Envelope{}
	envBytes, _ := proto.Marshal(env)
	block.Data.Data = [][]byte{envBytes}

	_, err = ChannelIDFromEvent(event)
	if err == nil {
		t.Fatalf("expecting error from ChannelIDFromEvent for invalid event type but got none")
	}

	// Invalid ChannelHeader in envelope
	payload := &cb.Payload{
		Header: &cb.Header{
			ChannelHeader: []byte("invalid"),
		},
	}
	payloadBytes, _ := proto.Marshal(payload)
	env.Payload = payloadBytes
	envBytes, _ = proto.Marshal(env)
	block.Data.Data = [][]byte{envBytes}

	_, err = ChannelIDFromEvent(event)
	if err == nil {
		t.Fatalf("expecting error from ChannelIDFromEvent for invalid event type but got none")
	}
}
