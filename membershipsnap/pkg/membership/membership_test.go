/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
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

	internalAddress1 = "internalhost1:1000"
	internalAddress2 = "internalhost2:1000"
	internalAddress3 = "internalhost3:1000"
)

// TestGetAllPeers tests Invoke with the "getAllPeers" function.
func TestGetAllPeers(t *testing.T) {
	localAddress := address1

	// First test with no members (except for self)
	memService := NewServiceWithMocks(
		msp1, localAddress,
	)

	endpoints := memService.GetAllPeers()
	expected := []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, localAddress, msp1),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("GetAllPeers - %s", err)
	}

	// Second test with two members plus self
	memService = NewServiceWithMocks(
		msp1, localAddress,
		NewMSPNetworkMembers(
			msp2,
			NewNetworkMember(pkiID2, address2, internalAddress2),
		),
		NewMSPNetworkMembers(
			msp3,
			NewNetworkMember(pkiID3, address3, internalAddress3),
		),
	)

	endpoints = memService.GetAllPeers()
	expected = []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, localAddress, msp1),
		newEndpoint(address2, internalAddress2, msp2),
		newEndpoint(address3, internalAddress3, msp3),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("GetAllPeers - %s", err)
	}
}

// TestGetPeersOfChannel tests Invoke with the "getPeersOfChannel" function.
func TestGetPeersOfChannel(t *testing.T) {
	t.Skipf("TestGetPeersOfChannel is skipped since MockCreateChain doesn't work with Viper 1.0.0.")

	channelID := "testchannel"
	localAddress := "host3:1000"

	peer.MockInitialize()
	defer ledgermgmt.CleanupTestEnv()

	// Test on channel that peer hasn't joined
	memService := NewServiceWithMocks(
		msp1, localAddress,
		NewMSPNetworkMembers(
			msp2,
			NewNetworkMember(pkiID2, address2, internalAddress2),
		),
		NewMSPNetworkMembers(
			msp3,
			NewNetworkMember(pkiID3, address3, internalAddress3),
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
		newEndpoint(address2, internalAddress2, msp2),
		newEndpoint(address3, internalAddress3, msp3),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - %s", err)
	}

	// Join the peer to the channel
	if err := peer.MockCreateChain(channelID); err != nil {
		t.Fatalf("unexpected error when creating mock channel: %s", err)
	}

	endpoints, err = memService.GetPeersOfChannel(channelID)
	if err != nil {
		t.Fatalf("getPeersOfChannel - unexpected error: %s", err)
	}

	expected = []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, localAddress, msp1),
		newEndpoint(address2, internalAddress2, msp2),
		newEndpoint(address3, internalAddress3, msp3),
	}

	if err := checkEndpoints(expected, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - %s", err)
	}
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
		if endpoint.Endpoint == expected.Endpoint && endpoint.InternalEndpoint == expected.InternalEndpoint {
			if !bytes.Equal(endpoint.MSPid, expected.MSPid) {
				return fmt.Errorf("the MSP ID [%s] of the endpoint does not match the expected MSP ID [%s]", endpoint.MSPid, expected.MSPid)
			}
			return nil
		}
	}
	return fmt.Errorf("endpoint %s not found in list of endpoints", expected)
}

func newEndpoint(endpoint string, internalEndpoint string, mspID []byte) *memserviceapi.PeerEndpoint {
	return &memserviceapi.PeerEndpoint{
		Endpoint:         endpoint,
		InternalEndpoint: internalEndpoint,
		MSPid:            mspID,
	}
}
