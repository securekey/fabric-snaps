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
	"github.com/securekey/fabric-snaps/membershipsnap/cmd/api"
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

	internalAddress1 = "internalhost1:1000"
	internalAddress2 = "internalhost2:1000"
	internalAddress3 = "internalhost3:1000"
)

func TestErrorInInit(t *testing.T) {
	stub := shim.NewMockStub("MembershipSnap", New())
	initializer = func(mscc *MembershipSnap, stub shim.ChaincodeStubInterface) error {
		return fmt.Errorf("some error")
	}

	resp := stub.MockInit("txid", nil)
	if resp.Status == shim.OK {
		t.Fatalf("Expecting Init to return error but got success.")
	}
}

// TestInvokeInvalidFunction tests Invoke method with an invalid function name
func TestInvokeInvalidFunction(t *testing.T) {
	identity := newMockIdentity()
	sProp, identityDeserializer := newMockSignedProposal(identity)

	args := [][]byte{}
	stub := newMockStub(identity, identityDeserializer, msp1, address1)
	if res := stub.MockInvokeWithSignedProposal("txID", args, sProp); res.Status == shim.OK {
		t.Fatalf("mscc invoke expecting error for invalid number of args")
	}

	args = [][]byte{[]byte("invalid")}
	if res := stub.MockInvokeWithSignedProposal("txID", args, sProp); res.Status == shim.OK {
		t.Fatalf("mscc invoke expecting error for invalid function")
	}
}

// TestGetAllPeers tests Invoke with the "getAllPeers" function.
func TestGetAllPeers(t *testing.T) {
	localAddress := address1

	// First test with no members (except for self)

	identity := newMockIdentity()
	sProp, identityDeserializer := newMockSignedProposal(identity)
	stub := newMockStub(identity, identityDeserializer, msp1, localAddress)

	args := [][]byte{[]byte(getAllPeersFunction)}
	res := stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getAllPeers) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatalf("mscc invoke(getAllPeers) - unexpected nil payload in response")
	}

	endpoints := &api.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - error unmarshalling payload: %s", err)
	}

	expected := []*api.PeerEndpoint{
		newEndpoint(localAddress, localAddress, msp1),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - %s", err)
	}

	// Second test with two members plus self

	args = [][]byte{[]byte(getAllPeersFunction)}

	stub = newMockStub(
		identity, identityDeserializer,
		msp1, localAddress,
		newMSPNetworkMembers(
			msp2,
			newNetworkMember(pkiID2, address2, internalAddress2),
		),
		newMSPNetworkMembers(
			msp3,
			newNetworkMember(pkiID3, address3, internalAddress3),
		),
	)

	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getAllPeers) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatalf("mscc invoke(getAllPeers) - unexpected nil payload in response")
	}

	endpoints = &api.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - error unmarshalling payload: %s", err)
	}

	expected = []*api.PeerEndpoint{
		newEndpoint(localAddress, localAddress, msp1),
		newEndpoint(address2, internalAddress2, msp2),
		newEndpoint(address3, internalAddress3, msp3),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getAllPeers) - %s", err)
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
	identity := newMockIdentity()
	sProp, identityDeserializer := newMockSignedProposal(identity)

	stub := newMockStub(
		identity, identityDeserializer,
		msp1, localAddress,
		newMSPNetworkMembers(
			msp2,
			newNetworkMember(pkiID2, address2, internalAddress2),
		),
		newMSPNetworkMembers(
			msp3,
			newNetworkMember(pkiID3, address3, internalAddress3),
		),
	)

	args := [][]byte{[]byte(getPeersOfChannelFunction)}
	res := stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status == shim.OK {
		t.Fatalf("mscc invoke(getPeersOfChannel) - Expecting error for nil channel ID")
	}

	args = [][]byte{[]byte(getPeersOfChannelFunction), nil}
	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status == shim.OK {
		t.Fatalf("mscc invoke(getPeersOfChannel) - Expecting error for nil channel ID")
	}

	args = [][]byte{[]byte(getPeersOfChannelFunction), []byte(channelID)}
	res = stub.MockInvokeWithSignedProposal("txID", args, sProp)
	if res.Status != shim.OK {
		t.Fatalf("mscc invoke(getPeersOfChannel) - unexpected status: %d, Message: %s", res.Status, res.Message)
	}

	if len(res.Payload) == 0 {
		t.Fatalf("mscc invoke(getPeersOfChannel) - unexpected nil payload in response")
	}

	endpoints := &api.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - error unmarshalling payload: %s", err)
	}

	expected := []*api.PeerEndpoint{
		newEndpoint(address2, internalAddress2, msp2),
		newEndpoint(address3, internalAddress3, msp3),
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
		t.Fatalf("mscc invoke(getPeersOfChannel) - unexpected nil payload in response")
	}

	endpoints = &api.PeerEndpoints{}
	if err := proto.Unmarshal(res.Payload, endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - error unmarshalling payload: %s", err)
	}

	expected = []*api.PeerEndpoint{
		newEndpoint(localAddress, localAddress, msp1),
		newEndpoint(address2, internalAddress2, msp2),
		newEndpoint(address3, internalAddress3, msp3),
	}

	if err := checkEndpoints(expected, endpoints.Endpoints); err != nil {
		t.Fatalf("mscc invoke(getPeersOfChannel) - %s", err)
	}
}

// TestAccessControl tests access control
func TestAccessControl(t *testing.T) {
	sProp, identityDeserializer := newMockSignedProposal([]byte("invalididentity"))

	// getAllPeers
	stub := newMockStub(newMockIdentity(), identityDeserializer, []byte("Org1MSP"), "localhost:1000")
	res := stub.MockInvokeWithSignedProposal("txID", [][]byte{[]byte(getAllPeersFunction), nil}, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "mscc invoke expected to fail with authorization error")
	assert.True(t, strings.HasPrefix(res.Message, "\"getAllPeers\" request failed authorization check"), "Unexpected error message: %s", res.Message)

	// getPeersOfChannel
	stub = newMockStub(newMockIdentity(), identityDeserializer, []byte("Org1MSP"), "localhost:1000")
	res = stub.MockInvokeWithSignedProposal("txID", [][]byte{[]byte(getPeersOfChannelFunction), nil}, sProp)
	assert.Equal(t, int32(shim.ERROR), res.Status, "mscc invoke expected to fail with authorization error")
	assert.True(t, strings.HasPrefix(res.Message, "\"getPeersOfChannel\" request failed authorization check"), "Unexpected error message: %s", res.Message)
}

func checkEndpoints(expected []*api.PeerEndpoint, actual []*api.PeerEndpoint) error {
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

func validate(actual []*api.PeerEndpoint, expected *api.PeerEndpoint) error {
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

func newEndpoint(endpoint string, internalEndpoint string, mspID []byte) *api.PeerEndpoint {
	return &api.PeerEndpoint{
		Endpoint:         endpoint,
		InternalEndpoint: internalEndpoint,
		MSPid:            mspID,
	}
}
