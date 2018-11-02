/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/gossip/discovery"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	msp1 = []byte("Org1MSP")
	msp2 = []byte("Org2MSP")
	msp3 = []byte("Org3MSP")

	pkiID2 = []byte("pkiid2")
	pkiID3 = []byte("pkiid3")

	address1 = "host1:1000"
	address2 = "host2:1000"
	address3 = "host3:1000"

	blockHeight1 = uint64(1000)
	blockHeight2 = uint64(1100)
	blockHeight3 = uint64(1200)
)

// TestGetAllPeers tests Invoke with the "getAllPeers" function.
func TestGetAllPeers(t *testing.T) {
	localAddress := address1

	// First test with no members (except for self)
	memService := NewServiceWithMocks(msp1, discovery.NetworkMember{Endpoint: localAddress})

	endpoints := memService.GetAllPeers()
	expected := []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, msp1, 0),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("GetAllPeers - %s", err)
	}

	// Second test with two members plus self
	memService = NewServiceWithMocks(
		msp1, discovery.NetworkMember{Endpoint: localAddress},
		NewMSPNetworkMembers(
			msp2,
			NewNetworkMember(pkiID2, address2),
		),
		NewMSPNetworkMembers(
			msp3,
			NewNetworkMember(pkiID3, address3),
		),
	)

	endpoints = memService.GetAllPeers()
	expected = []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, msp1, 0),
		newEndpoint(address2, msp2, 0),
		newEndpoint(address3, msp3, 0),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("GetAllPeers - %s", err)
	}
}

// TestGetPeersOfChannel tests Invoke with the "getPeersOfChannel" function.
func TestGetPeersOfChannel(t *testing.T) {
	channelID := "testchannel"
	localAddress := "host3:1000"
	localBlockHeight := blockHeight1

	// Test on channel that peer hasn't joined
	memService := NewServiceWithMocks(
		msp1, NewNetworkChannelMember([]byte(""), localAddress, localBlockHeight),
		NewMSPNetworkMembers(
			msp2,
			NewNetworkChannelMember(pkiID2, address2, blockHeight2),
		),
		NewMSPNetworkMembers(
			msp3,
			NewNetworkChannelMember(pkiID3, address3, blockHeight3),
		),
	)

	endpoints, err := memService.GetPeersOfChannel("")
	if err == nil {
		t.Fatalf("getPeersOfChannel - Expecting error for empty channel ID but got none")
	}

	endpoints, err = memService.GetPeersOfChannel(channelID)
	if err != nil {
		t.Fatalf("getPeersOfChannel - unexpected error: %s", err)
	}

	expected := []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, msp1, localBlockHeight),
		newEndpoint(address2, msp2, blockHeight2),
		newEndpoint(address3, msp3, blockHeight3),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - %s", err)
	}
}

func TestGetLocalPeer(t *testing.T) {
	channelID := "testchannel"
	localAddress := "host3:1000"
	localBlockHeight := blockHeight1

	// Test on channel that peer hasn't joined
	memService := NewServiceWithMocks(
		msp1, NewNetworkChannelMember([]byte(""), localAddress, localBlockHeight),
	)

	endpoint, err := memService.GetLocalPeer("somechannel")
	assert.NoErrorf(t, err, "Expecting error since local peer is not joined to channel but got none")

	endpoint, err = memService.GetLocalPeer(channelID)
	require.NoError(t, err)

	assert.Equal(t, localAddress, endpoint.Endpoint)
	assert.Equal(t, localBlockHeight, endpoint.LedgerHeight)
}

func checkEndpoints(expected []*memserviceapi.PeerEndpoint, actual []*memserviceapi.PeerEndpoint) error {
	fmt.Printf("Expected: %v, Actual: %v\n", expected, actual)

	if len(expected) != len(actual) {
		return fmt.Errorf("expecting %d endpoints but received %d", len(expected), len(actual))
	}

	for _, endpoint := range expected {
		if err := validate(actual, endpoint); err != nil {
			return err
		}
	}

	return nil
}

func validate(actual []*memserviceapi.PeerEndpoint, expected *memserviceapi.PeerEndpoint) error {
	for _, endpoint := range actual {
		if endpoint.Endpoint == expected.Endpoint && bytes.Equal(endpoint.MSPid, expected.MSPid) {
			if endpoint.LedgerHeight != expected.LedgerHeight {
				return fmt.Errorf("the ledger height [%d] of the endpoint does not match the expected ledger height [%d]", endpoint.LedgerHeight, expected.LedgerHeight)
			}
			return nil
		}
	}
	return fmt.Errorf("endpoint %s not found in list of endpoints", expected)
}

func newEndpoint(endpoint string, mspID []byte, ledgerHeight uint64) *memserviceapi.PeerEndpoint {
	return &memserviceapi.PeerEndpoint{
		Endpoint:     endpoint,
		MSPid:        mspID,
		LedgerHeight: ledgerHeight,
	}
}
