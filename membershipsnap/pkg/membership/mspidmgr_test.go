/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"testing"
	"time"

	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	msppb "github.com/hyperledger/fabric/protos/msp"
)

// TestMSPIDMgr tests the MSP ID manager
func TestMSPIDMgr(t *testing.T) {
	mspID1 := "Org1MSP"
	mspID2 := "Org2MSP"
	mspID3 := "Org3MSP"

	pkiID1 := []byte("pki-id-1")
	pkiID2 := []byte("pki-id-2")
	pkiID3 := []byte("pki-id-3")

	gossipService := newMockGossipService(
		discovery.NetworkMember{
			Endpoint: "localhost:9999",
			PKIid:    []byte("pkiid1"),
		},
	)

	mspIDMgr := newMSPIDMgr(gossipService)

	time.Sleep(1 * time.Second)

	gossipService.sendMessage(newIdentityMsg(
		gcommon.PKIidType(pkiID1),
		&msppb.SerializedIdentity{
			Mspid:   mspID1,
			IdBytes: []byte("some-identity"),
		}))
	gossipService.sendMessage(newIdentityMsg(
		gcommon.PKIidType(pkiID2),
		&msppb.SerializedIdentity{
			Mspid:   mspID2,
			IdBytes: []byte("some-identity"),
		}))

	time.Sleep(1 * time.Second)

	if mspID := mspIDMgr.GetMSPID(pkiID1); mspID != mspID1 {
		t.Fatalf("Expecting MSP ID [%s] but got [%s]", mspID1, mspID)
	}
	if mspID := mspIDMgr.GetMSPID(pkiID2); mspID != mspID2 {
		t.Fatalf("Expecting MSP ID [%s] but got [%s]", mspID2, mspID)
	}

	// Send a message that we don't care about - should be ignored
	gossipService.sendMessage(newAliveMsg())

	time.Sleep(1 * time.Second)

	// Send a new MSP ID for an existing PKI ID and expect it to change
	gossipService.sendMessage(newIdentityMsg(
		gcommon.PKIidType(pkiID1),
		&msppb.SerializedIdentity{
			Mspid:   mspID3,
			IdBytes: []byte("some-identity"),
		}))

	time.Sleep(1 * time.Second)

	if mspID := mspIDMgr.GetMSPID(pkiID1); mspID != mspID3 {
		t.Fatalf("Expecting MSP ID [%s] but got [%s]", mspID3, mspID)
	}

	// Retrieve an unknown PKI ID
	if mspID := mspIDMgr.GetMSPID(pkiID3); mspID != "" {
		t.Fatalf("Expecting MSP ID [] but got [%s]", mspID)
	}

	gossipService.Stop()
	time.Sleep(1 * time.Second)
}
