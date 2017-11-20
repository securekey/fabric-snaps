/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package healthcheck

import (
	"os"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func TestMain(m *testing.M) {
	setup()
	r := m.Run()
	teardown()
	os.Exit(r)
}
func setup() {
	// do any test setup for all tests here...
}
func teardown() {
	// do any teardown activities here ..
}

func TestDefaultSmokeTestReturnEmptyResult(t *testing.T) {
	stub := shim.NewMockStub("", nil)
	resp := SmokeTest("", stub, [][]byte{})
	if resp.Status != shim.OK {
		t.Fatalf("Default smokeTest returned abnormal message '%s'. Payload is: '%s'", resp.GetMessage(), resp.GetPayload())
	}
}
