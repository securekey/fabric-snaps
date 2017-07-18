/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package main

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	logging "github.com/op/go-logging"
	config "github.com/securekey/fabric-snaps/pkg/snaps/examplesnap/config"
)

var logger = logging.MustGetLogger("examplesnap")

// ExampleSnap is a sample Snap that simply echos the supplied argument in the response
type ExampleSnap struct {
}

// Init - read configuration
func (t *ExampleSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	err := config.Init("")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to initialize config: %s", err)
		logger.Errorf(errMsg)
		return shim.Error(errMsg)
	}

	logger.Info("Example snap configuration loaded.")
	return shim.Success(nil)
}

// Invoke snap
// arg[0] - Function (currently not used)
// arg[1] - Some message (optional)
func (t *ExampleSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	fn, args := stub.GetFunctionAndParameters()

	if fn == "" {
		return shim.Error("Missing function name")
	}

	logger.Infof("Example snap invoked. Args=%v", args)
	if len(args) > 0 {
		return shim.Success([]byte(args[0]))
	}

	// message not provided - return default greeting
	configGreeting := config.GetGreeting()

	return shim.Success([]byte(configGreeting))
}

func main() {
	err := shim.Start(new(ExampleSnap))
	if err != nil {
		fmt.Printf("Error starting Example snap: %s", err)
	}
}
