/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"fmt"
	"os"

	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/api/config"
	"github.com/securekey/fabric-snaps/pkg/snapdispatcher"
)

var logger = logging.MustGetLogger("snap2")

// Snap2 is a sample Snap that simply echos the supplied argument in the response
type Snap2 struct {
	name string
}

// NewSnap - create new instance of snap
func NewSnap() shim.Chaincode {
	return &Snap2{}
}

// Invoke snap
// arg[0] - Some message (optional)
func (s *Snap2) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("Snap[%s] - Invoked", s.name)

	args := stub.GetArgs()
	if len(args) > 0 {
		return shim.Success([]byte(args[0]))
	}
	return shim.Success([]byte("Example snap invoked!"))
}

// Init initializes the snap
// arg[0] - Snap name
func (s *Snap2) Init(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetStringArgs()
	if len(args) == 0 {
		return shim.Error("Expecting snap name as first arg")
	}

	s.name = args[0]

	logger.Infof("Snap[%s] - Initialized", s.name)

	return shim.Success(nil)
}

func main() {
	fmt.Println("***** Snap 2 is starting *****")
	snapsDaemon := snapdispatcher.NewSnapsDaemon()

	err := snapsDaemon.Initialize([]*config.SnapConfig{
		&config.SnapConfig{
			Name: "snap2",
			Snap: &Snap2{},
		}})
	if err != nil {
		logger.Errorf("Error initializing Snap2 Daemon: %s\n", err)
		os.Exit(2)
	}

	if err := snapsDaemon.Start(); err != nil {
		logger.Errorf("Error starting Snap2 Daemon: %s\n", err)
	}
}
