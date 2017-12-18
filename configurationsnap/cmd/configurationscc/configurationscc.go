/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudflare/cfssl/log"
	errors "github.com/pkg/errors"

	"encoding/json"

	"github.com/gogo/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	protosMSP "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	configapi "github.com/securekey/fabric-snaps/configurationsnap/api"
	config "github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/config"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/configdata"

	"github.com/securekey/fabric-snaps/healthcheck"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/txsnapservice"
)

// functionRegistry is a registry of the functions that are supported by configuration snap
var functionRegistry = map[string]func(shim.ChaincodeStubInterface, [][]byte) pb.Response{
	"getPublicKeyForLogging": getPublicKeyForLogging,
	"healthCheck":            healthCheck,
	"save":                   save,
	"get":                    get,
	"delete":                 delete,
	"refresh":                refresh,
}

var availableFunctions = functionSet()

var logger = shim.NewLogger("configuration-snap")

//default cache refresh interval is 10 seconds
var refreshInterval uint32 = 10

// ConfigurationSnap implementation
type ConfigurationSnap struct {
}

var peerConfigPath = ""

// Init snap
func (configSnap *ConfigurationSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Warningf("******** Init Config Snap on channel [%s]\n", stub.GetChannelID())

	if stub.GetChannelID() != "" {
		config, err := config.New(stub.GetChannelID(), peerConfigPath)
		if err != nil {
			return shim.Error(fmt.Sprintf("error getting config for channel %s %v", stub.GetChannelID(), err))
		}
		configmgmtService.Initialize(stub, config.PeerMspID)
		interval := time.Duration(config.RefreshInterval) * time.Second
		periodicRefresh(stub.GetChannelID(), interval)
	}
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
	//TODO the Initialize should delete from here when DEV-4253 done
	configmgmtService.Initialize(stub, configKey.MspID)
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

func refresh(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {

	config, err := config.New(stub.GetChannelID(), peerConfigPath)
	if err != nil {
		return shim.Error(fmt.Sprintf("error getting config for channel %s %v", stub.GetChannelID(), err))
	}
	logger.Warningf("******** refresh invoked [%s]\n", config.RefreshInterval)
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)
	instance.Refresh(stub, config.PeerMspID)
	msg := fmt.Sprintf("Refresh has started on %d refresh interval", config.RefreshInterval)
	return shim.Success([]byte(msg))
}

func periodicRefresh(channelID string, refreshInterval time.Duration) {
	logger.Warningf("***Periodic refresh called on  [%d]\n", refreshInterval)

	go func() {
		for {
			time.Sleep(refreshInterval)
			sendRefreshRequest(channelID)
			config, err := config.New(channelID, peerConfigPath)
			if err != nil {
				log.Warning("Got error while creating config for channel %v\n", channelID)
			}
			refreshInterval = time.Duration(config.RefreshInterval) * time.Second
		}
	}()
}

func sendRefreshRequest(channelID string) {
	logger.Warningf("***Sending request refresh for channel %s", channelID)

	txService, err := txsnapservice.Get(channelID)
	if err != nil {
		fmt.Printf("Cannot get txService: %v", err)
	}
	if txService.Config != nil {
		sendEndorseRequest(channelID, txService)
	}

}

func sendEndorseRequest(channelID string, txService *txsnapservice.TxServiceImpl) {
	peerConfig, err := txService.Config.GetLocalPeer()
	if err != nil {
		fmt.Printf("Cannot get local peer: %v", err)
	}
	s := []string{peerConfig.Host, strconv.Itoa(peerConfig.Port)}
	peerURL := strings.Join(s, ":")

	targetPeer, err := fabapi.NewPeer(peerURL, txService.Config.GetTLSRootCertPath(), "", txService.ClientConfig())
	if err != nil {
		fmt.Printf("Error creating target peer: %v", err)
	}
	args := [][]byte{[]byte("refresh")}
	txSnapReq := createTransactionSnapRequest("configurationsnap", channelID, args, nil, nil, false)
	txService.EndorseTransaction(txSnapReq, []sdkApi.Peer{targetPeer})
}

func createTransactionSnapRequest(chaincodeID string, chnlID string,
	endorserArgs [][]byte, transientMap map[string][]byte,
	ccIDsForEndorsement []string, registerTxEvent bool) *api.SnapTransactionRequest {

	return &api.SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        transientMap,
		EndorserArgs:        endorserArgs,
		CCIDsForEndorsement: ccIDsForEndorsement,
		RegisterTxEvent:     registerTxEvent}

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

// New chaincode implementation
func New() shim.Chaincode {
	return &ConfigurationSnap{}
}

func main() {
}
