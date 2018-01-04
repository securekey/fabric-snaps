/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/mocks"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/mocks/mockchpeer"
)

const (
	blockHeight1 = uint64(1000)
	blockHeight2 = uint64(1001)
	blockHeight3 = uint64(1002)
)

func TestPeerFilter(t *testing.T) {
	f, err := New([]string{})
	if err == nil {
		t.Fatalf("Expecting error when no channel ID provided but got none")
	}

	channelID := "testchannel"
	localBlockHeight := blockHeight2

	f, err = newWithOpts([]string{channelID}, mockbcinfo.NewProvider(mockbcinfo.NewChannelBCInfo(channelID, mockbcinfo.BCInfo(localBlockHeight))))
	if err != nil {
		t.Fatalf("Got error when creating peer filter")
	}

	if f.Accept(newPeer("p0")) {
		t.Fatalf("Expecting that peer will NOT be accepted since the given peer is not a ChannelPeer so it doesn't contain the block height")
	}
	if f.Accept(mockchpeer.New("p1", "Org1MSP", channelID, blockHeight1)) {
		t.Fatalf("Expecting that peer will NOT be accepted since its block height [%d] is less than the block height of the local peer [%d]", blockHeight1, localBlockHeight)
	}
	if !f.Accept(mockchpeer.New("p2", "Org1MSP", channelID, blockHeight2)) {
		t.Fatalf("Expecting that peer will be accepted since its block height [%d] is equal to the block height of the local peer [%d]", blockHeight2, localBlockHeight)
	}
	if !f.Accept(mockchpeer.New("p3", "Org1MSP", channelID, blockHeight3)) {
		t.Fatalf("Expecting that peer will be accepted since its block height [%d] is greater than the block height of the local peer [%d]", blockHeight3, localBlockHeight)
	}
}

func newPeer(name string) apifabclient.Peer {
	peer, err := fabapi.NewPeer("grpc://"+name+":7051", "", "", mocks.NewMockConfig())
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %v)", err))
	}
	return peer
}
