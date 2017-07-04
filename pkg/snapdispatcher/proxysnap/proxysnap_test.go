/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package proxysnap

import (
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/api/protos"
)

func TestInit(t *testing.T) {
	snap := &proxySnap{}
	stub := shim.NewMockStub("proxysnap", snap)

	snapName := "someremotesnap"
	snapURL := "remotehost:9999"

	var args [][]byte
	args = append(args, []byte(snapName))
	args = append(args, []byte(snapURL))

	response := stub.MockInit("TxID", args)

	if response.Status != shim.OK {
		t.Fatalf("Expecting response status %d but got %d", shim.OK, response.Status)
	}
	if response.Message != "" {
		t.Fatalf("Expecting no response message but got %s", response.Message)
	}
	if snap.name != snapName {
		t.Fatalf("Expecting snap name to be %s but got %s", snapName, snap.name)
	}
	if snap.url != snapURL {
		t.Fatalf("Expecting snap URL to be %s but got %s", snapURL, snap.url)
	}
}
func TestInvoke(t *testing.T) {
	expectedResponse := "some response"

	snap := newSnap(func(url string) SnapsClient {
		return &mockSnapsClient{
			url:     url,
			status:  shim.OK,
			payload: [][]byte{[]byte(expectedResponse)},
		}
	})

	stub := shim.NewMockStub("proxysnap", snap)

	snapName := "someremotesnap"
	snapURL := "remotehost:9999"

	var args [][]byte
	args = append(args, []byte(snapName))
	args = append(args, []byte(snapURL))

	response := stub.MockInvoke("TxID", args)

	if response.Status != shim.OK {
		t.Fatalf("Expecting response status %d but got %d", shim.OK, response.Status)
	}
	if len(response.Payload) == 0 {
		t.Fatalf("Expecting one response payload but got none")
	}
	if string(response.Payload) != expectedResponse {
		t.Fatalf("Expecting %s in response payload but got %s", expectedResponse, string(response.Payload))
	}
}

type mockSnapsClient struct {
	url        string
	status     protos.Status
	errMessage string
	payload    [][]byte
}

func (c *mockSnapsClient) Send(request *protos.Request) protos.Response {
	return protos.Response{
		Status:  c.status,
		Payload: c.payload,
		Error:   c.errMessage}
}

func (c *mockSnapsClient) Disconnect() {
}
