/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	"fmt"
	"testing"

	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	membershipMocks "github.com/securekey/fabric-snaps/membershipsnap/pkg/mocks"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
)

const (
	blockHeight1 = uint64(1000)
	blockHeight2 = uint64(1001)
	blockHeight3 = uint64(1002)
)

func TestPeerFilter(t *testing.T) {
	_, err := New([]string{})
	if err == nil {
		t.Fatal("Expecting error when no channel ID provided but got none")
	}

	channelID := "testchannel"
	localBlockHeight := blockHeight2

	f, err := newWithOpts([]string{channelID}, mockbcinfo.NewProvider(mockbcinfo.NewChannelBCInfo(channelID, mockbcinfo.BCInfo(localBlockHeight))))
	if err != nil {
		t.Fatal("Got error when creating peer filter")
	}

	if f.Accept(newPeer("p0")) {
		t.Fatal("Expecting that peer will NOT be accepted since the given peer is not a ChannelPeer so it doesn't contain the block height")
	}
	if f.Accept(membershipMocks.New("p1", "Org1MSP", channelID, blockHeight1)) {
		t.Fatalf("Expecting that peer will NOT be accepted since its block height [%d] is less than the block height of the local peer [%d]", blockHeight1, localBlockHeight)
	}
	if !f.Accept(membershipMocks.New("p2", "Org1MSP", channelID, blockHeight2)) {
		t.Fatalf("Expecting that peer will be accepted since its block height [%d] is equal to the block height of the local peer [%d]", blockHeight2, localBlockHeight)
	}
	if !f.Accept(membershipMocks.New("p3", "Org1MSP", channelID, blockHeight3)) {
		t.Fatalf("Expecting that peer will be accepted since its block height [%d] is greater than the block height of the local peer [%d]", blockHeight3, localBlockHeight)
	}
}

func newPeer(name string) fabApi.Peer {
	peer, err := peer.New(mocks.NewMockEndpointConfig(), peer.WithURL("grpc://"+name+":7051"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %s)", err))
	}
	return peer
}
