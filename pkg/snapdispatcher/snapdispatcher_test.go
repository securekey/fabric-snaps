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

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/api/config"
	snapInterfaces "github.com/securekey/fabric-snaps/api/interfaces"
	snapProtos "github.com/securekey/fabric-snaps/api/protos"
	"github.com/securekey/fabric-snaps/pkg/snapdispatcher/registry"
	"github.com/securekey/fabric-snaps/pkg/snaps/examplesnap"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const address = "localhost"

var notImplemented = "Required functionality was not implemented"

func TestInvokeOnRegisteredSnap(t *testing.T) {
	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instantiate client
	client := snapProtos.NewSnapClient(conn)
	arg := "Hello from invoke"
	payload := [][]byte{[]byte(arg), []byte("example")}
	//use registered snap
	irequest := snapProtos.Request{SnapName: "examplesnap", Args: payload}
	//invoke snap server
	iresponse, err := client.Invoke(context.Background(), &irequest)

	select {
	case <-time.After(2 * time.Second):
	}

	if err != nil {
		t.Fatalf("Connect to SNAP return error: %v", err)
	}

	if iresponse.Status != shim.OK {
		t.Fatalf("Expected status of OK, received %d ", iresponse.Status)
	}

	//for now assume that Invoke snap will return payload[0]
	responseMsg := string(iresponse.Payload[0])
	if responseMsg != arg {
		t.Fatalf("Expected %s, received %s ", arg, responseMsg)
	}
	fmt.Printf("Received response from snap invoke %s\n", responseMsg)
}

func TestImplementedSnapMethods(t *testing.T) {
	payload := [][]byte{[]byte("Hello from invoke"), []byte("example")}
	//use registered snap
	irequest := snapProtos.Request{SnapName: "example", Args: payload}

	//Create snap stub and pass it in
	snapStub := snapInterfaces.NewSnapStub(irequest.Args)
	args := snapStub.GetArgs()
	if len(args) == 0 {
		t.Fatalf("Function GetArgs was implemented. Expected length of 2; got  %v", len(args))
	}

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
	irequest := snapProtos.Request{SnapName: "example", Args: payload}
	//Create snap stub and pass it in
	snapStub := snapInterfaces.NewSnapStub(irequest.Args)
	//test snapStub methods
	//GetTxID ...

	txID := snapStub.GetTxID()
	if txID != notImplemented {
		t.Fatalf("GetTxID was not implemented.")
	}

	_, err := snapStub.GetState("abc")
	if err.Error() != notImplemented {
		t.Fatalf("GetState was not implemented.")
	}

	err = snapStub.PutState("abc", nil)
	if err.Error() != notImplemented {
		t.Fatalf("PutState was not implemented.")

	}

	err = snapStub.DelState("abc")
	if err.Error() != notImplemented {
		t.Fatalf("DelState was not implemented.")

	}

	_, err = snapStub.GetStateByRange("abc", "def")
	if err.Error() != notImplemented {
		t.Fatalf("GetStateByRange was not implemented.")
	}

	_, err = snapStub.GetStateByPartialCompositeKey("abc", []string{"def"})
	if err.Error() != notImplemented {
		t.Fatalf("GetStateByPartialCompositeKey was not implemented.")
	}

	_, err = snapStub.CreateCompositeKey("abc", []string{"def"})
	if err.Error() != notImplemented {
		t.Fatalf("CreateCompositeKey was not implemented.")
	}

	_, _, err = snapStub.SplitCompositeKey("abc")
	if err.Error() != notImplemented {
		t.Fatalf("SplitCompositeKey was not implemented.")
	}

	_, err = snapStub.GetQueryResult("abc")
	if err.Error() != notImplemented {
		t.Fatalf("GetQueryResult was not implemented.")
	}

	_, err = snapStub.GetHistoryForKey("abc")
	if err.Error() != notImplemented {
		t.Fatalf("GetHistoryForKey was not implemented.")
	}

	_, err = snapStub.GetCreator()
	if err.Error() != notImplemented {
		t.Fatalf("GetCreator was not implemented.")
	}

	_, err = snapStub.GetTransient()
	if err.Error() != notImplemented {
		t.Fatalf("GetTransient was not implemented.")
	}

	_, err = snapStub.GetBinding()
	if err.Error() != notImplemented {
		t.Fatalf("GetBinding was not implemented.")
	}

	_, err = snapStub.GetArgsSlice()
	if err.Error() != notImplemented {
		t.Fatalf("GetArgsSlice was not implemented.")
	}

	_, err = snapStub.GetTxTimestamp()
	if err.Error() != notImplemented {
		t.Fatalf("GetTxTimestamp was not implemented.")
	}

	err = snapStub.SetEvent("abc", nil)
	if err.Error() != notImplemented {
		t.Fatalf("SetEvent was not implemented. ")

	}

}

func TestInvokeOnNonRegisteredSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instantiate client
	client := snapProtos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//use non registered snap
	irequest := snapProtos.Request{SnapName: "thisSnapWasNotRegistered", Args: payload}
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
	client := snapProtos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//registered snap - does not have receiver interface
	irequest := snapProtos.Request{SnapName: "invalidConfig", Args: payload}
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
	client := snapProtos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//registered snap - does not have receiver interface
	irequest := snapProtos.Request{SnapName: "", Args: payload}
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

	// override tls enabled to false in config.yaml for unit testing
	viper.Set("snap.server.tls.enabled", false)
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
	var snaps []*config.SnapConfig

	snaps = append(snaps, &config.SnapConfig{
		Name: "examplesnap",
		Snap: &examplesnap.ExampleSnap{},
	})
	snaps = append(snaps, &config.SnapConfig{
		Name: "invalidConfig",
	})

	snapsRegistry := registry.NewSnapsRegistry(snaps)
	if err := snapsRegistry.Initialize(); err != nil {
		panic(fmt.Sprintf("Error initializing Snaps Registry: %s", err))
	}

	go startSnapServer(snapsRegistry)

	os.Exit(m.Run())
}
