/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/ledger/ledgermgmt"
	"github.com/hyperledger/fabric/core/peer"
	memserviceapi "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	memservice "github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
	"github.com/stretchr/testify/assert"
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

func TestErrorInInit(t *testing.T) {
	stub := shim.NewMockStub("MembershipSnap", New())
	initializer = func(mscc *MembershipSnap) error {
		return fmt.Errorf("some error")
	}

	resp := stub.MockInit("txid", nil)
	if resp.Status == shim.OK {
		t.Fatal("Expecting Init to return error but got success.")
	}
}

// TestInvokeInvalidFunction tests Invoke method with an invalid function name
func TestInvokeInvalidFunction(t *testing.T) {
	identity := newMockIdentity()
	sProp, identityDeserializer := newMockSignedProposal(identity)

	args := [][]byte{}
	stub := newMockStub(identity, identityDeserializer, msp1, address1, mockbcinfo.ChannelBCInfos())
	if res := stub.MockInvokeWithSignedProposal("txID", args, sProp); res.Status == shim.OK {
		t.Fatal("mscc invoke expecting error for invalid number of args")
	}

	args = [][]byte{[]byte("invalid")}
	if res := stub.MockInvokeWithSignedProposal("txID", args, sProp); res.Status == shim.OK {
		t.Fatal("mscc invoke expecting error for invalid function")
	}
}

// TestGetAllPeers tests Invoke with the "getAllPeers" function.
func TestGetAllPeers(t *testing.T) {
	localAddress := address1

	// First test with no members (except for self)

	identity := newMockIdentity()
	sProp, identityDeserializer := newMockSignedProposal(identity)
	stub := newMockStub(identity, identityDeserializer, msp1, localAddress, mockbcinfo.ChannelBCInfos())

	args := [][]byte{[]byte(getAllPeersFunction)}
	res := stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getAllPeers) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatal("mscc invoke(getAllPeers) - unexpected nil payload in response")
	}

	endpoints := &memserviceapi.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - error unmarshalling payload: %s", err)
	}

	expected := []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, msp1),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - %s", err)
	}

	// Second test with two members plus self

	args = [][]byte{[]byte(getAllPeersFunction)}

	stub = newMockStub(
		identity, identityDeserializer,
		msp1, localAddress, mockbcinfo.ChannelBCInfos(),
		memservice.NewMSPNetworkMembers(
			msp2,
			memservice.NewNetworkMember(pkiID2, address2, 0),
		),
		memservice.NewMSPNetworkMembers(
			msp3,
			memservice.NewNetworkMember(pkiID3, address3, 0),
		),
	)

	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getAllPeers) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatal("mscc invoke(getAllPeers) - unexpected nil payload in response")
	}

	endpoints = &memserviceapi.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - error unmarshalling payload: %s", err)
	}

	expected = []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, msp1),
		newEndpoint(address2, msp2),
		newEndpoint(address3, msp3),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - %s", err)
	}
}

// TestGetPeersOfChannel tests Invoke with the "getPeersOfChannel" function.
func TestGetPeersOfChannel(t *testing.T) {
	t.Skip("TestGetPeersOfChannel is skipped since MockCreateChain doesn't work with Viper 1.0.0.")

	channelID := "testchannel"
	localAddress := "host3:1000"
	localBlockHeight := blockHeight2

	peer.MockInitialize()
	defer ledgermgmt.CleanupTestEnv()

	// Test on channel that peer hasn't joined
	identity := newMockIdentity()
	sProp, identityDeserializer := newMockSignedProposal(identity)

	stub := newMockStub(
		identity, identityDeserializer,
		msp1, localAddress, mockbcinfo.ChannelBCInfos(mockbcinfo.NewChannelBCInfo(channelID, mockbcinfo.BCInfo(localBlockHeight))),
		memservice.NewMSPNetworkMembers(
			msp2,
			memservice.NewNetworkMember(pkiID2, address2, blockHeight1),
		),
		memservice.NewMSPNetworkMembers(
			msp3,
			memservice.NewNetworkMember(pkiID3, address3, blockHeight2),
		),
	)

	args := [][]byte{[]byte(getPeersOfChannelFunction)}
	res := stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status == shim.OK {
		t.Fatal("mscc invoke(getPeersOfChannel) - Expecting error for nil channel ID")
	}

	args = [][]byte{[]byte(getPeersOfChannelFunction), nil}
	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status == shim.OK {
		t.Fatal("mscc invoke(getPeersOfChannel) - Expecting error for nil channel ID")
	}

	args = [][]byte{[]byte(getPeersOfChannelFunction), []byte(channelID)}
	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getPeersOfChannel) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatal("mscc invoke(getPeersOfChannel) - unexpected nil payload in response")
	}

	endpoints := &memserviceapi.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - error unmarshalling payload: %s", err)
	}

	expected := []*memserviceapi.PeerEndpoint{
		newEndpoint(address2, msp2),
		newEndpoint(address3, msp3),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - %s", err)
	}

	// Join the peer to the channel
	if err := peer.MockCreateChain(channelID); err != nil {
		t.Fatalf("unexpected error when creating mock channel: %s", err)
	}

	args = [][]byte{[]byte(getPeersOfChannelFunction), []byte(channelID)}
	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getPeersOfChannel) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatal("mscc invoke(getPeersOfChannel) - unexpected nil payload in response")
	}

	endpoints = &memserviceapi.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - error unmarshalling payload: %s", err)
	}

	expected = []*memserviceapi.PeerEndpoint{
		newEndpoint(localAddress, msp1),
		newEndpoint(address2, msp2),
		newEndpoint(address3, msp3),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - %s", err)
	}
}

// TestAccessControl tests access control
func TestAccessControl(t *testing.T) {
	sProp, identityDeserializer := newMockSignedProposal([]byte("invalididentity"))

	// getAllPeers
	stub := newMockStub(newMockIdentity(), identityDeserializer, []byte("Org1MSP"), "localhost:1000", mockbcinfo.ChannelBCInfos())
	res := stub.MockInvokeWithSignedProposal("txID", [][]byte{[]byte(getAllPeersFunction), nil}, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "mscc invoke expected to fail with authorization error")
	assert.True(t, strings.Contains(res.Message, "\"getAllPeers\" request failed authorization check"), "Unexpected error message: %s", res.Message)

	// getPeersOfChannel
	stub = newMockStub(newMockIdentity(), identityDeserializer, []byte("Org1MSP"), "localhost:1000", mockbcinfo.ChannelBCInfos())
	res = stub.MockInvokeWithSignedProposal("txID", [][]byte{[]byte(getPeersOfChannelFunction), nil}, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "mscc invoke expected to fail with authorization error")
	assert.True(t, strings.Contains(res.Message, "\"getPeersOfChannel\" request failed authorization check"), "Unexpected error message: %s", res.Message)
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
		if endpoint.Endpoint == expected.Endpoint {
			if !bytes.Equal(endpoint.MSPid, expected.MSPid) {
				return fmt.Errorf("the MSP ID [%s] of the endpoint does not match the expected MSP ID [%s]", endpoint.MSPid, expected.MSPid)
			}
			return nil
		}
	}
	return fmt.Errorf("endpoint %s not found in list of endpoints", expected)
}

func newEndpoint(endpoint string, mspID []byte) *memserviceapi.PeerEndpoint {
	return &memserviceapi.PeerEndpoint{
		Endpoint: endpoint,
		MSPid:    mspID,
	}
}
