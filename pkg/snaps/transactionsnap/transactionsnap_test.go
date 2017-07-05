/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package transactionsnap

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func TestTransactionSnapInit(t *testing.T) {
	snap := &TxSnapImpl{}
	stub := shim.NewMockStub("transactionsnap", snap)

	snapName := "transactionsnap"
	snapURL := ""

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

}
func TestTransactionSnapInvoke(t *testing.T) {
	snap := NewSnap()

	transientMap := make(map[string][]byte)
	transientMap["key"] = []byte("transientvalue")
	endorserArgs := make([][]byte, 5)
	endorserArgs[0] = []byte("invoke")
	endorserArgs[1] = []byte("move")
	endorserArgs[2] = []byte("a")
	endorserArgs[3] = []byte("b")
	endorserArgs[4] = []byte("1")
	additionalCCIDs := []string{"additionalccid"}

	snapTxReq := SnapTransactionRequest{ChannelID: "testChannel",
		ChaincodeID:     "ccid",
		TransientMap:    transientMap,
		EndorserArgs:    endorserArgs,
		AdditionalCCIDs: additionalCCIDs}
	fmt.Printf("snapTxReq %v", snapTxReq)

	snapTxReqB, err := json.Marshal(snapTxReq)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("snapTxReqB %v ", snapTxReqB)

	var args [][]byte
	args = append(args, []byte("endorseTransaction"))
	args = append(args, snapTxReqB)
	stub := shim.NewMockStub("transactionsnap", snap)
	//initialize fabric-client
	response := stub.MockInit("TxID", args)
	//invoke transaction snap
	response = stub.MockInvoke("TxID", args)

	if response.Status != shim.ERROR {
		t.Fatalf("TestTransactionSnapInvoke failed. Expected response status %d but got %d", shim.ERROR, response.Status)
	}
}
