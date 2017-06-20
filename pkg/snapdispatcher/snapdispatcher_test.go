/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package snapdispatcher

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	snap_interfaces "github.com/securekey/fabric-snaps/api/interfaces"
	snap_protos "github.com/securekey/fabric-snaps/api/protos"
	"github.com/securekey/fabric-snaps/cmd/config"
	"google.golang.org/grpc"
)

const address = "localhost"

func TestInvokeOnRegisteredSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instantiate client
	client := snap_protos.NewSnapClient(conn)
	payload := [][]byte{[]byte("Hello from invoke"), []byte("example")}
	//use registered snap
	irequest := snap_protos.Request{SnapName: "example", Args: payload}
	//invoke snap server
	iresponse, err := client.Invoke(context.Background(), &irequest)

	select {
	case <-time.After(2 * time.Second):
	}

	if err != nil {
		t.Fatalf("Connect to SNAP return error: %v", err)
	}
	//for now assume that Invoke snap will return payload[0]
	responseMsg := string(iresponse.Payload[0])
	if responseMsg != "Hello from invoke" {
		t.Fatalf("Expected hello, received %s ", responseMsg)
	}
	fmt.Printf("Received response from snap invoke %s\n", responseMsg)
}

func TestImplementedSnapMethods(t *testing.T) {
	payload := [][]byte{[]byte("Hello from invoke"), []byte("example")}
	//use registered snap
	irequest := snap_protos.Request{SnapName: "example", Args: payload}

	//Create snap stub and pass it in
	snapStub := snap_interfaces.NewSnapStub(irequest.Args)
	args := snapStub.GetArgs()
	if len(args) == 0 {
		t.Fatalf("Function GetArgs was implemented. Expected length of 2; got  %v", len(args))
	}
	fmt.Println("Returned args ", args)

	function, parameters := snapStub.GetFunctionAndParameters()
	if function == "" {
		t.Fatalf("Expected function name, but it was not set")
	}
	if len(parameters) == 0 {
		t.Fatalf("GetFunctionAndParameters should return parameter")
	}

	//GetStringArgs
	stringArgs := snapStub.GetStringArgs()
	if len(stringArgs) == 0 {
		t.Fatalf("GetStringArgs shoud return value")
	}

}

func TestUnimplementedSnapMethods(t *testing.T) {

	//use registered snap
	payload := [][]byte{[]byte("Hello from invoke"), []byte("example")}
	//use example snap
	irequest := snap_protos.Request{SnapName: "example", Args: payload}
	//Create snap stub and pass it in
	snapStub := snap_interfaces.NewSnapStub(irequest.Args)
	//test snapStub methods
	//GetTxID ...
	txID := snapStub.GetTxID()
	if txID != "" {
		t.Fatalf("GetTxID was not implemented. Expected nil in return")
	}

	state, err := snapStub.GetState("abc")
	if state != nil || err != nil {
		t.Fatalf("GetState was not implemented. Expected nil in return")
	}

	err = snapStub.PutState("abc", nil)
	if err != nil {
		t.Fatalf("PutState was not implemented. Expected nil...")

	}

	err = snapStub.DelState("abc")
	if err != nil {
		t.Fatalf("DelState was not implemented. Expected nil...")

	}

	result, err := snapStub.GetStateByRange("abc", "def")
	if result != nil || err != nil {
		t.Fatalf("GetStateByRange was not implemented. Expected nil...")
	}

	result, err = snapStub.GetStateByPartialCompositeKey("abc", []string{"def"})
	if result != nil || err != nil {
		t.Fatalf("GetStateByPartialCompositeKey was not implemented. Expected nil...")
	}

	key, err := snapStub.CreateCompositeKey("abc", []string{"def"})
	if key != "" || err != nil {
		t.Fatalf("CreateCompositeKey was not implemented. Expected nil...")
	}

	part1, part2, err := snapStub.SplitCompositeKey("abc")
	if part1 != "" || part2 != nil || err != nil {
		t.Fatalf("SplitCompositeKey was not implemented. Expected nil...")
	}

	result, err = snapStub.GetQueryResult("abc")
	if result != nil || err != nil {
		t.Fatalf("GetQueryResult was not implemented. Expected nil.")
	}

	result, err = snapStub.GetHistoryForKey("abc")
	if result != nil || err != nil {
		t.Fatalf("GetHistoryForKey was not implemented. Expected nil.")
	}

	creator, err := snapStub.GetCreator()
	if creator != nil || err != nil {
		t.Fatalf("GetCreator was not implemented. Expected nil.")
	}

	transient, err := snapStub.GetTransient()
	if transient != nil || err != nil {
		t.Fatalf("GetTransient was not implemented. Expected nil.")
	}

	binding, err := snapStub.GetBinding()
	if binding != nil || err != nil {
		t.Fatalf("GetBinding was not implemented. Expected nil.")
	}

	slice, err := snapStub.GetArgsSlice()
	if slice != nil || err != nil {
		t.Fatalf("GetArgsSlice was not implemented. Expected nil.")
	}

	ts, err := snapStub.GetTxTimestamp()
	if ts != nil || err != nil {
		t.Fatalf("GetTxTimestamp was not implemented. Expected nil.")
	}

	err = snapStub.SetEvent("abc", nil)
	if err != nil {
		t.Fatalf("SetEvent was not implemented. Expected nil.")

	}
	response := snapStub.InvokeChaincode("abc", nil, "channel")
	if response.Payload != nil {
		t.Fatalf("InvokeChaincode was not implemented. Expected nil.")
	}

}

func TestInvokeOnNonRegisteredSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instantiate client
	client := snap_protos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//use non registered snap
	irequest := snap_protos.Request{SnapName: "thisSnapWasNotRegistered", Args: payload}
	//invoke snap server
	_, err = client.Invoke(context.Background(), &irequest)
	if err == nil {
		t.Fatalf("Expected error for non registered snap: ")
	}

}

func TestRequiredConfigFieldsOnSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instantiate client
	client := snap_protos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//registered snap - does not have receiver interface
	irequest := snap_protos.Request{SnapName: "invalidConfig", Args: payload}
	//invoke snap server
	_, err = client.Invoke(context.Background(), &irequest)
	if err == nil {
		t.Fatalf("Expected error for non registered snap ")
	}

}

func TestNoNameSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instantiate client
	client := snap_protos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//registered snap - does not have receiver interface
	irequest := snap_protos.Request{SnapName: "", Args: payload}
	//invoke snap server
	_, err = client.Invoke(context.Background(), &irequest)
	if err == nil {
		t.Fatalf("Expected error for name less snap ")
	}

}

//connectToSnapServer
func connectToSnapServer() (*grpc.ClientConn, error) {
	// Set up a connection to the server.
	var opts []grpc.DialOption

	//read snap server config
	snapServerPort := config.GetSnapServerPort()
	if snapServerPort == "" {
		logger.Infof("Snap server port was not set. ")
		return nil, fmt.Errorf("Error detecting snap server port")
	}
	opts = append(opts, grpc.WithInsecure())
	snapServerAddress := address + ":" + snapServerPort
	logger.Infof("Dialing snap server on: %s", snapServerAddress)
	//grpc to snap peer
	conn, err := grpc.Dial(snapServerAddress, opts...)
	select {
	case <-time.After(2 * time.Second):

	}
	if err != nil {
		return nil, err
	}
	return conn, err

}
func TestMain(m *testing.M) {
	err := config.Init("")
	if err != nil {
		panic(fmt.Sprintf("Error initializing config: %s", err))
	}
	//Add snap used for testing
	testSnaps := config.SnapConfig{
		Enabled:  true,
		Name:     "invalidConfig",
		InitArgs: [][]byte{[]byte("")},
	}
	config.Snaps = append(config.Snaps, &testSnaps)

	go StartSnapServer()

	os.Exit(m.Run())
}
