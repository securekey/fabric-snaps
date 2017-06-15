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

	"github.com/securekey/fabric-snaps/config"
	snap_protos "github.com/securekey/fabric-snaps/snaps/protos"
	"google.golang.org/grpc"
)

const address = "localhost"

func TestInvokeOnRegisteredSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instatnitate client
	client := snap_protos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
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

func TestInvokeOnNonRegisteredSnap(t *testing.T) {

	conn, err := connectToSnapServer()
	if err != nil {
		t.Fatalf("Connect to SNAP server returned an error: %v", err)
	}
	defer conn.Close()
	//instatnitate client
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
	//instatnitate client
	client := snap_protos.NewSnapClient(conn)
	payload := [][]byte{[]byte("testChain"), []byte("example")}
	//registered sanp - does not have receiver interface
	irequest := snap_protos.Request{SnapName: "invalidConfig", Args: payload}
	//invoke snap server
	_, err = client.Invoke(context.Background(), &irequest)
	if err == nil {
		t.Fatalf("Expected error for non registered snap: ")
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
	//TODO add TLS here
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
