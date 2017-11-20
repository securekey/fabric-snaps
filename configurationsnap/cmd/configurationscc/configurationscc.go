/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	errors "github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	protosMSP "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/configdata"
	"github.com/securekey/fabric-snaps/healthcheck"

	"encoding/json"

	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	mgmtservice "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	configapi "github.com/securekey/fabric-snaps/configurationsnap/api"
)

// functionRegistry is a registry of the functions that are supported by configuration snap
var functionRegistry = map[string]func(shim.ChaincodeStubInterface, [][]byte) pb.Response{
	"getPublicKeyForLogging": getPublicKeyForLogging,
	"healthCheck":            healthCheck,
	"save":                   save,
	"get":                    get,
	"delete":                 delete,
}

var availableFunctions = functionSet()

var logger = shim.NewLogger("configuration-snap")

//default cache refresh interval is 5 seconds
var refreshInterval uint32 = 5

// ConfigurationSnap implementation
type ConfigurationSnap struct {
}

// Init snap
func (configSnap *ConfigurationSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	if err := setRefreshInterval(stub.GetArgs()); err != nil {
		return shim.Error(err.Error())
	}
	go refreshCache(stub)
	return shim.Success(nil)
}

// Invoke is the main entry point for invocations
func (configSnap *ConfigurationSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) == 0 {
		return shim.Error(fmt.Sprintf("Function not provided. Expecting one of (%s)", availableFunctions))
	}

	functionName := string(args[0])
	function, ok := functionRegistry[functionName]
	if !ok {
		return shim.Error(fmt.Sprintf("Invalid function: %s. Expecting one of (%s)", functionName, availableFunctions))
	}

	functionArgs := args[1:]

	logger.Infof("Invoking function [%s] with args: %s", functionName, functionArgs)
	return function(stub, functionArgs)
}

// functionSet returns a string enumerating all available functions
func functionSet() string {
	var functionNames string
	for name := range functionRegistry {
		functionNames = functionNames + " " + name
	}
	return functionNames
}

// getPublicKeyForLogging returns public key used for logging encryption
func getPublicKeyForLogging(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {

	configBytes, err := json.Marshal(&configapi.PublicKeyForLogging{PublicKey: configdata.PublicKeyForLogging, KeyID: configdata.KeyIDForLogging})

	if err != nil {
		shim.Error(fmt.Sprintf("failed to marshal public key logging config data. %v ", err))
	}

	return shim.Success(configBytes)
}

// healthCheck is the health check function of this ConfigurationSnap
func healthCheck(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	response := healthcheck.SmokeTest(healthcheck.ConfigurationScc, stub, args)
	if response.Status != shim.OK {
		es := fmt.Sprintf("%s healthcheck failed: %s", healthcheck.ConfigurationScc, response.Message)
		logger.Errorf("%s", es)
		return shim.Error(es)
	}
	return shim.Success(response.Payload)
}

//save - saves configuration passed in args
func save(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	config := args[0]
	if len(config) == 0 {
		return shim.Error("Config is empty-cannot be saved")
	}
	cmngr := mgmt.NewConfigManager(stub)
	err := cmngr.Save(config)
	if err != nil {
		logger.Errorf("Got error while saving cnfig %v", err)
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

//get - gets configuration using configkey as criteria
func get(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {

	configKey, err := getKey(args)
	if err != nil {
		return shim.Error(err.Error())
	}
	//valid key
	cmngr := mgmt.NewConfigManager(stub)
	config, err := cmngr.Get(*configKey)
	if err != nil {
		logger.Errorf("Get for key %v returns error: %v", configKey, err)
		return shim.Error(err.Error())
	}

	payload, err := json.Marshal(config)
	if err != nil {
		logger.Errorf("Got error while marshalling config: %v", err)
		return shim.Error(err.Error())
	}
	return shim.Success(payload)

}

//delete - deletes configuration using config key as criteria
func delete(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {

	configKey, err := getKey(args)
	if err != nil {
		return shim.Error(err.Error())
	}
	//valid key
	cmngr := mgmt.NewConfigManager(stub)
	if err := cmngr.Delete(*configKey); err != nil {
		logger.Errorf("Got error while deleting config: %v", err)
		return shim.Error(err.Error())

	}
	return shim.Success(nil)
}

//getKey gets config key from args
func getKey(args [][]byte) (*mgmtapi.ConfigKey, error) {
	configKey := &mgmtapi.ConfigKey{}
	if len(args) == 0 {
		logger.Error("Config is empty (no args)")
		return configKey, errors.New("Config is empty (no args)")
	}

	configBytes := args[0]
	if len(configBytes) == 0 {
		logger.Error("Config is empty (no key)")
		return configKey, errors.New("Config is empty (no key)")
	}
	if err := json.Unmarshal(configBytes, &configKey); err != nil {
		errStr := fmt.Sprintf("Got error %v unmarshalling config key %s", err, string(configBytes[:]))
		logger.Error(errStr)
		return configKey, errors.New(errStr)
	}
	err := mgmt.ValidateConfigKey(*configKey)
	if err != nil {
		if configKey.MspID != "" {
			return configKey, nil
		}
		return configKey, err
	}
	return configKey, nil
}

//getIdentity gets associated membership service provider
func getIdentity(stub shim.ChaincodeStubInterface) (string, error) {
	if stub == nil {
		return "", errors.New("Stub is nil")
	}
	creator, err := stub.GetCreator()
	if err != nil {
		logger.Errorf("Cannot get creatorBytes error %v", err)
		return "", err
	}
	sid := &protosMSP.SerializedIdentity{}
	if err := proto.Unmarshal(creator, sid); err != nil {
		logger.Errorf("Unmarshal creatorBytes error %v", err)
		return "", err
	}
	return sid.Mspid, nil
}

//setRefreshInterval sets cache refresh interval based on value
//pased in args list. If no value was passed the default refresh interval will be used.
//If args list is not empty expected value is int representing the cahce refresh interval
func setRefreshInterval(args [][]byte) error {
	if len(args) == 0 {
		return nil
	}
	cri, err := strconv.Atoi(string(args[0]))
	if err != nil {
		logger.Infof("refreshInterval was not set properly %v", err)
		return errors.Errorf("cache refresh interval was not set properly %v", err)
	}
	if cri < 0 {
		logger.Infof("refreshInterval cannot be less then zero.")
		return errors.New("cache refresh interval cannot be less then zero")
	}
	//reset
	atomic.StoreUint32(&refreshInterval, uint32(cri))
	logger.Infof("Using cache refresh interval of  %d seconds", cri)
	return nil

}

//refreshCache used to refresh cache on scheduled interval
func refreshCache(stub shim.ChaincodeStubInterface) error {
	adminService := mgmtservice.ConfigServiceImpl{}
	//get caller's mspid
	mspID, err := getIdentity(stub)
	if err != nil {
		return err
	}
	//refresh period
	ticker := time.NewTicker(time.Duration(atomic.LoadUint32(&refreshInterval)) * time.Second)
	for {
		select {
		case <-ticker.C:
			adminService.Refresh(stub, mspID)
		}
	}
}

// New chaincode implementation
func New() shim.Chaincode {
	return &ConfigurationSnap{}
}

func main() {
}
