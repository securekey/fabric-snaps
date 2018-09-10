/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package minblockheight

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	membershipMocks "github.com/securekey/fabric-snaps/membershipsnap/pkg/mocks"
	"github.com/securekey/fabric-snaps/mocks/mockmembership"
)

const (
	channelID = "testchannel"

	blockHeight1 = uint64(1000)
	blockHeight2 = uint64(1001)
	blockHeight3 = uint64(1002)

	org1MSP = "Org1MSP"
)

func TestPeerFilter(t *testing.T) {
	mockMembership := &mockmembership.Service{
		PeersOfChannel: map[string][]*memserviceapi.PeerEndpoint{
			channelID: {
				&memserviceapi.PeerEndpoint{
					Endpoint: "p0:7051",
				},
				&memserviceapi.PeerEndpoint{
					Endpoint:     "p1:7051",
					MSPid:        []byte(org1MSP),
					LedgerHeight: blockHeight1,
				},
				&memserviceapi.PeerEndpoint{
					Endpoint:     "p2:7051",
					MSPid:        []byte(org1MSP),
					LedgerHeight: blockHeight2,
				},
				&memserviceapi.PeerEndpoint{
					Endpoint:     "p3:7051",
					MSPid:        []byte(org1MSP),
					LedgerHeight: blockHeight3,
				},
			},
		},
	}

	memServiceProvider = func() (memserviceapi.Service, error) {
		return mockMembership, nil
	}

	_, err := New([]string{})
	if err == nil {
		t.Fatal("Expecting error when no channel ID provided but got none")
	}

	minBlockHeight := blockHeight2

	f, err := New([]string{channelID, fmt.Sprintf("%d", minBlockHeight)})
	require.NoErrorf(t, err, "Got error when creating peer filter")

	assert.Falsef(t, f.Accept(membershipMocks.New("p1", org1MSP, blockHeight1)), "Expecting that peer will NOT be accepted since its block height [%d] is less than the block height of the local peer [%d]", blockHeight1, minBlockHeight)

	assert.Truef(t, f.Accept(membershipMocks.New("p2", org1MSP, blockHeight2)), "Expecting that peer will be accepted since its block height [%d] is equal to the block height of the local peer [%d]", blockHeight2, minBlockHeight)

	assert.Truef(t, f.Accept(membershipMocks.New("p3", org1MSP, blockHeight3)), "Expecting that peer will be accepted since its block height [%d] is greater than the block height of the local peer [%d]", blockHeight3, minBlockHeight)
}

func TestPeerFilterError(t *testing.T) {
	mockMembership := &mockmembership.Service{
		Error: fmt.Errorf("simulated error"),
	}

	memServiceProvider = func() (memserviceapi.Service, error) {
		return mockMembership, nil
	}

	_, err := New([]string{})
	if err == nil {
		t.Fatal("Expecting error when no channel ID provided but got none")
	}

	minBlockHeight := blockHeight2

	f, err := New([]string{channelID, fmt.Sprintf("%d", minBlockHeight)})
	require.NoErrorf(t, err, "Got error when creating peer filter")

	assert.Falsef(t, f.Accept(membershipMocks.New("p2", org1MSP, blockHeight2)), "Expecting that peer will NOT be accepted since an error is returned when getting peers of channel")
}

func newPeer(name string) fabApi.Peer {
	peer, err := peer.New(mocks.NewMockEndpointConfig(), peer.WithURL("grpc://"+name+":7051"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create peer: %s)", err))
	}
	return peer
}
