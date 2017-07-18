/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"strings"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	config "github.com/securekey/fabric-snaps/pkg/snaps/examplesnap/config"
)

func TestInit(t *testing.T) {

	stub := newMockStub()

	res := stub.MockInit("txID", [][]byte{})
	if res.Status == shim.OK {
		t.Fatalf("Init should have failed due to missing config")
	}

	initConfig(t)

	res = stub.MockInit("txID", [][]byte{})
	if res.Status != shim.OK {
		t.Fatalf("Init failed: %v", res.Message)
	}

}

func TestInvoke(t *testing.T) {

	stub := newMockStub()

	initConfig(t)

	args := [][]byte{}
	verifyFailure(t, stub, args, "Missing function name")

	args = [][]byte{[]byte("invoke"), []byte("Test Hello")}
	verifySuccess(t, stub, args, "Test Hello")

	args = [][]byte{[]byte("invoke")}
	verifySuccess(t, stub, args, "Configured Hello")

}

func verifySuccess(t *testing.T, stub *shim.MockStub, args [][]byte, expected string) {
	res := stub.MockInvoke("txID", args)
	if res.Status != shim.OK {
		t.Fatalf("Invoke should have completed successfully args: %v", res.Message)
	}

	if !strings.Contains(string(res.Payload), expected) {
		t.Fatalf("Expecting response to contain %s, got %s", expected, string(res.Payload))
	}
}

func verifyFailure(t *testing.T, stub *shim.MockStub, args [][]byte, expected string) {
	res := stub.MockInvoke("txID", args)
	if res.Status == shim.OK {
		t.Fatalf("Expected shim to fail. Status=OK, Message: %s", res.Message)
	}

	if !strings.Contains(res.Message, expected) {
		t.Fatalf("Expecting error messasge to contain %s, got %s", expected, res.Message)
	}
}

func initConfig(t *testing.T) {
	err := config.Init("./sampleconfig")
	if err != nil {
		t.Fatalf("Error initializing config: %s", err)
	}
}

func newMockStub() *shim.MockStub {
	snap := new(ExampleSnap)
	return shim.NewMockStub("examplesnap", snap)
}
