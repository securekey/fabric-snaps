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

	"encoding/json"

	"github.com/cloudflare/cfssl/log"
	errors "github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric/bccsp"
	factory "github.com/hyperledger/fabric/bccsp/factory"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	protosMSP "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	config "github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/config"
	"github.com/securekey/fabric-snaps/healthcheck"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/txsnapservice"
)

// functionRegistry is a registry of the functions that are supported by configuration snap
var functionRegistry = map[string]func(shim.ChaincodeStubInterface, [][]byte) pb.Response{
	"healthCheck":     healthCheck,
	"save":            save,
	"get":             get,
	"delete":          delete,
	"refresh":         refresh,
	"generateKeyPair": generateKeyPair,
}
var supportedAlgs = []string{"ECDSA", "ECDSAP256", "ECDSAP384", "RSA", "RSA1024", "RSA2048", "RSA3072", "RSA4096"}
var availableFunctions = functionSet()

var logger = shim.NewLogger("configuration-snap")

// ConfigurationSnap implementation
type ConfigurationSnap struct {
}

var peerConfigPath = ""

// Init snap
func (configSnap *ConfigurationSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debugf("******** Init Config Snap on channel [%s]\n", stub.GetChannelID())
	if stub.GetChannelID() != "" {

		peerMspID, err := config.GetPeerMSPID(peerConfigPath)
		if err != nil {
			return shim.Error(fmt.Sprintf("error getting peer's msp id %v", err))
		}
		peerID, err := config.GetPeerID(peerConfigPath)
		if err != nil {
			return shim.Error(fmt.Sprintf("error getting peer's  id %v", err))
		}
		interval := config.GetDefaultRefreshInterval()
		logger.Debugf("******** Call initialize for [%s][%s][%v]\n", peerMspID, peerID, interval)
		configmgmtService.Initialize(stub, peerMspID)
		periodicRefresh(stub.GetChannelID(), peerID, peerMspID, interval)
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

func refresh(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {

	peerMspID, err := config.GetPeerMSPID(peerConfigPath)
	if err != nil {
		return shim.Error(fmt.Sprintf("error getting peer's msp id %v", err))
	}
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)
	instance.Refresh(stub, peerMspID)
	return shim.Success(nil)
}

//to generate key pair based on options submitted
//expected keytype and ephemeral flag in args
func generateKeyPair(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	if len(args) < 2 {
		return shim.Error(fmt.Sprintf("Required arguments are: key type and ephemeral flag"))
	}
	keyType := string(args[0])
	ephemeral, err := strconv.ParseBool(string(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("Ephemeral flag is not set"))
	}
	//check if requesteed option was supported
	options, err := getKeyOpts(keyType, ephemeral)
	if err != nil {
		return shim.Error(err.Error())
	}
	//generate key
	return generateKeyWithOpts(stub.GetChannelID(), options)
}

func getKeyOpts(keyType string, ephemeral bool) (bccsp.KeyGenOpts, error) {

	switch keyType {
	case "ECDSA":
		return &bccsp.ECDSAKeyGenOpts{Temporary: ephemeral}, nil
	case "ECDSAP256":
		return &bccsp.ECDSAP256KeyGenOpts{Temporary: ephemeral}, nil
	case "ECDSAP384":
		return &bccsp.ECDSAP384KeyGenOpts{Temporary: ephemeral}, nil
	case "RSA":
		return &bccsp.RSAKeyGenOpts{Temporary: ephemeral}, nil
	case "RSA1024":
		return &bccsp.RSA1024KeyGenOpts{Temporary: ephemeral}, nil
	case "RSA2048":
		return &bccsp.RSA2048KeyGenOpts{Temporary: ephemeral}, nil
	case "RSA3072":
		return &bccsp.RSA3072KeyGenOpts{Temporary: ephemeral}, nil
	case "RSA4096":
		return &bccsp.RSA4096KeyGenOpts{Temporary: ephemeral}, nil
	default:
		supportedAlgsMsg := strings.Join(supportedAlgs, ",")
		return nil, errors.Errorf("The key algorithm is invalid. Supported options: %s", supportedAlgsMsg)
	}

}

//generateKeyWithOpts to generate key using BCCSP
func generateKeyWithOpts(channelID string, opts bccsp.KeyGenOpts) pb.Response {

	cfgopts, err := config.GetBCCSPOpts(channelID, peerConfigPath)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Debugf("BCCSP Plugin option config map %v", cfgopts)
	//just once - initialize factory with options
	//if factory was already initialized this call will be ignored
	factory.InitFactories(cfgopts)
	logger.Debugf("****Passing opts %s %v", cfgopts.ProviderName, cfgopts)
	bccspsuite, err := factory.GetBCCSPFromOpts(cfgopts)
	if err != nil {
		logger.Debugf("Error initializing with options %s %s %s ", cfgopts.Pkcs11Opts.Library, cfgopts.Pkcs11Opts.Pin, cfgopts.Pkcs11Opts.Label)
		return shim.Error(fmt.Sprintf("Got error from GetBCCSPFromOpts in %v", err))
	}
	k, err := bccspsuite.KeyGen(opts)
	if err != nil {
		return shim.Error(fmt.Sprintf("Got error from KeyGen in %v %v", opts, err))
	}
	return parseKey(k)
}

//pass generated key (private/public) and return public to caller
func parseKey(k bccsp.Key) pb.Response {
	logger.Debugf("Parsing key %v", k)
	var pubKey bccsp.Key
	var err error
	if k.Private() {
		pubKey, err = k.PublicKey()
		if err != nil {
			return shim.Error(fmt.Sprintf("Error:getting public key %v", err))
		}
	} else {
		pubKey = k
	}
	pubKeyBts, err := pubKey.Bytes()
	if err != nil {
		return shim.Error(fmt.Sprintf("Error:getting public key bytes %v", err))
	}
	logger.Debugf("***PubKey - len '%d' - SKI: '%v'", len(pubKeyBts), pubKey.SKI())
	return shim.Success(pubKeyBts)

}

func periodicRefresh(channelID string, peerID string, peerMSPID string, refreshInterval time.Duration) {
	logger.Debugf("***Periodic refresh was called on [%d]\n", refreshInterval)
	go func() {
		for {
			time.Sleep(refreshInterval)
			sendRefreshRequest(channelID, peerID, peerMSPID)
			csccconfig, err := config.New(channelID, peerConfigPath)
			if err != nil {
				log.Debugf("Got error while creating config for channel %v\n", channelID)
			}
			if csccconfig == nil {
				refreshInterval = config.GetDefaultRefreshInterval()
			} else {
				if csccconfig.RefreshInterval < config.GetMinimumRefreshInterval() {
					refreshInterval = config.GetMinimumRefreshInterval()
				} else {
					refreshInterval = csccconfig.RefreshInterval
				}
			}
		}
	}()
}

func sendRefreshRequest(channelID string, peerID string, peerMSPID string) {
	//call to get snaps config from ledger and to initilaize cahce instance
	txService, err := txsnapservice.Get(channelID)
	if err != nil {
		logger.Debugf("Cannot get txService: %v", err)
		return
	}
	if txService.Config != nil {
		sendEndorseRequest(channelID, txService)
	}

}

func sendEndorseRequest(channelID string, txService *txsnapservice.TxServiceImpl) {
	peerConfig, err := txService.Config.GetLocalPeer()
	if err != nil {
		logger.Debugf("Cannot get local peer: %v", err)
	}
	s := []string{peerConfig.Host, strconv.Itoa(peerConfig.Port)}
	peerURL := strings.Join(s, ":")

	targetPeer, err := fabapi.NewPeer(peerURL, txService.Config.GetTLSRootCertPath(), "", txService.ClientConfig())
	if err != nil {
		logger.Debugf("Error creating target peer: %v", err)
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
