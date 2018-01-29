/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	sdkpeer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric/bccsp"
	factory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/signer"
	shim "github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	mgmt "github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	config "github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/config"
	"github.com/securekey/fabric-snaps/healthcheck"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/txsnapservice"
	errors "github.com/securekey/fabric-snaps/util/errors"
)

// functionRegistry is a registry of the functions that are supported by configuration snap
var functionRegistry = map[string]func(shim.ChaincodeStubInterface, [][]byte) pb.Response{
	"healthCheck":     healthCheck,
	"save":            save,
	"get":             get,
	"delete":          delete,
	"refresh":         refresh,
	"generateKeyPair": generateKeyPair,
	"generateCSR":     generateCSR,
}
var supportedAlgs = []string{"ECDSA", "ECDSAP256", "ECDSAP384", "RSA", "RSA1024", "RSA2048", "RSA3072", "RSA4096"}
var availableFunctions = functionSet()

var logger = logging.NewLogger("configsnap")

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
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
		peerID, err := config.GetPeerID(peerConfigPath)
		if err != nil {
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
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

//to generate CSR based on supplied arguments
//first arg: key type (ECDSA, RSA)
//second arg : ephemeral flag (true/false)
//third  arg: signature algorithm (one of x509.SignatureAlgorithm)
func generateCSR(stub shim.ChaincodeStubInterface, args [][]byte) pb.Response {
	//check args
	if len(args) < 3 {
		return shim.Error(fmt.Sprintf("Required arguments are: [key type,ephemeral flag and CSR's signature algorithm"))
	}
	keyType := string(args[0])
	ephemeral, err := strconv.ParseBool(string(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("Ephemeral flag is not set"))
	}
	sigAlgType := string(args[2])
	//get requested key options
	options, err := getKeyOpts(keyType, ephemeral)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Debugf("Keygen options %v")
	bccspsuite, keys, err := getBCCSPAndKeyPair(stub.GetChannelID(), options)
	if err != nil {
		return shim.Error(err.Error())
	}
	csrTemplate, err := getCSRTemplate(stub.GetChannelID(), keys, keyType, sigAlgType)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Debugf("Certificate request template %v", csrTemplate)
	//generate the csr request
	cryptoSigner, err := signer.New(bccspsuite, keys)
	if err != nil {
		return shim.Error(err.Error())
	}
	csrReq, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, cryptoSigner)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Debugf("CSR was created. Len is %d", len(csrReq))
	return shim.Success(csrReq)

}

func getCSRTemplate(channelID string, keys bccsp.Key, keyType string, sigAlgType string) (x509.CertificateRequest, error) {
	var csrTemplate x509.CertificateRequest
	sigAlg, err := getSignatureAlg(sigAlgType)
	if err != nil {
		return csrTemplate, err
	}
	//generate subject for CSR
	asn1Subj, err := getCSRSubject(channelID)
	if err != nil {
		return csrTemplate, err
	}

	csrConfig, err := config.GetCSRConfigOptions(channelID, peerConfigPath)
	if err != nil {
		return csrTemplate, err
	}
	if keys == nil {
		return csrTemplate, errors.New(errors.GeneralError, "Invalid key")
	}
	pubKey, err := keys.PublicKey()
	if err != nil {
		logger.Debugf("Get error parsing public key %v", err)
		return csrTemplate, err
	}
	pubKeyAlg, err := getPublicKeyAlg(keyType)
	if err != nil {
		logger.Debugf("Get error parsing public key alg %v", err)
		return csrTemplate, err
	}
	//generate a csr template
	csrTemplate = x509.CertificateRequest{
		Version:            1,
		RawSubject:         asn1Subj,
		SignatureAlgorithm: sigAlg,
		PublicKeyAlgorithm: pubKeyAlg,
		PublicKey:          pubKey,
		//subject alternative names
		DNSNames:       csrConfig.DNSNames,
		EmailAddresses: csrConfig.EmailAddresses,
		IPAddresses:    csrConfig.IPAddresses,
	}
	logger.Debugf("Certificate request template %v", csrTemplate)
	return csrTemplate, nil

}

func getCSRSubject(channelID string) ([]byte, error) {
	if channelID == "" {
		return nil, errors.Errorf(errors.GeneralError, "Channel is required")
	}
	//get csr configuration - from config(HL)
	csrConfig, err := getCSRConfig(channelID, peerConfigPath)
	if err != nil {
		return nil, err
	}
	logger.Debugf("csrConfig options %v", csrConfig)
	subj := pkix.Name{
		CommonName:         csrConfig.CommonName,
		Country:            []string{csrConfig.Country},
		Province:           []string{csrConfig.StateProvince},
		Locality:           []string{csrConfig.Locality},
		Organization:       []string{csrConfig.Org},
		OrganizationalUnit: []string{csrConfig.OrgUnit},
	}
	logger.Debugf("Subject options %v", subj)

	rawSubj := subj.ToRDNSequence()

	asn1Subj, err := asn1.Marshal(rawSubj)
	if err != nil {
		return nil, err
	}
	return asn1Subj, nil
}

func getCSRConfig(channelID string, peerConfigPath string) (*config.CSRConfig, error) {
	if channelID == "" {
		return nil, errors.New(errors.GeneralError, "Channel is required")
	}

	csrConfig, err := config.GetCSRConfigOptions(channelID, peerConfigPath)
	if err != nil {
		return nil, err
	}
	if csrConfig.CommonName == "" {
		return nil, errors.New(errors.GeneralError, "Common name is required")

	}
	if csrConfig.Country == "" {
		return nil, errors.New(errors.GeneralError, "Country name is required")
	}
	if csrConfig.StateProvince == "" {
		return nil, errors.New(errors.GeneralError, "StateProvince name is required")
	}
	if csrConfig.Locality == "" {
		return nil, errors.New(errors.GeneralError, "Locality name is required")
	}
	if csrConfig.Org == "" {
		return nil, errors.New(errors.GeneralError, "Organization name is required")
	}
	if csrConfig.OrgUnit == "" {
		return nil, errors.New(errors.GeneralError, "OrganizationalUnit name is required")
	}
	return csrConfig, nil

}

func getBCCSPAndKeyPair(channelID string, opts bccsp.KeyGenOpts) (bccsp.BCCSP, bccsp.Key, error) {
	var k bccsp.Key
	var err error
	var bccspsuite bccsp.BCCSP

	if channelID == "" {
		return bccspsuite, k, errors.New(errors.GeneralError, "Channel is required")

	}
	if opts == nil {
		return bccspsuite, k, errors.New(errors.GeneralError, "The key gen option is required")
	}

	cfgopts, err := config.GetBCCSPOpts(channelID, peerConfigPath)
	if err != nil {
		return bccspsuite, k, err
	}

	logger.Debugf("BCCSP Plugin option config map %v", cfgopts)
	//just once - initialize factory with options
	//if factory was already initialized this call will be ignored
	factory.InitFactories(cfgopts)
	logger.Debugf("****Passing opts %s %v", cfgopts.ProviderName, cfgopts)
	bccspsuite, err = factory.GetBCCSPFromOpts(cfgopts)
	if err != nil {
		logger.Debugf("Error initializing with options %s %s %s ", cfgopts.Pkcs11Opts.Library, cfgopts.Pkcs11Opts.Pin, cfgopts.Pkcs11Opts.Label)
		return bccspsuite, k, errors.Wrap(errors.GeneralError, err, "BCCSP Initialize failed")
	}
	k, err = bccspsuite.KeyGen(opts)
	if err != nil {
		return bccspsuite, k, errors.Wrap(errors.GeneralError, err, "Key Gen failed")
	}
	return bccspsuite, k, nil
}

func getPublicKeyAlg(algorithm string) (x509.PublicKeyAlgorithm, error) {
	var sigAlg x509.PublicKeyAlgorithm
	switch algorithm {
	case "RSA":
		return x509.RSA, nil
	case "DSA":
		return x509.RSA, nil
	case "ECDSA":
		return x509.RSA, nil
	default:
		return sigAlg, errors.Errorf(errors.GeneralError, "Public key algorithm is not supported %s", algorithm)
	}
}
func getSignatureAlg(algorithm string) (x509.SignatureAlgorithm, error) {
	var sigAlg x509.SignatureAlgorithm
	switch algorithm {
	case "ECDSAWithSHA1":
		return x509.ECDSAWithSHA1, nil
	case "ECDSAWithSHA256":
		return x509.ECDSAWithSHA256, nil
	case "ECDSAWithSHA384":
		return x509.ECDSAWithSHA384, nil
	case "ECDSAWithSHA512":
		return x509.ECDSAWithSHA512, nil
	case "SHA256WithRSAPSS":
		return x509.SHA256WithRSAPSS, nil
	case "SHA384WithRSAPSS":
		return x509.SHA384WithRSAPSS, nil
	case "SHA512WithRSAPSS":
		return x509.SHA512WithRSAPSS, nil
	case "DSAWithSHA256":
		return x509.DSAWithSHA256, nil
	case "DSAWithSHA1":
		return x509.DSAWithSHA1, nil
	case "SHA512WithRSA":
		return x509.SHA512WithRSA, nil
	case "SHA384WithRSA":
		return x509.SHA384WithRSA, nil
	case "SHA256WithRSA":
		return x509.SHA256WithRSA, nil
	case "SHA1WithRSA":
		return x509.SHA1WithRSA, nil
	case "MD5WithRSA":
		return x509.MD5WithRSA, nil
	case "MD2WithRSA":
		return x509.MD2WithRSA, nil
	default:
		return sigAlg, errors.New(errors.GeneralError, "Alg is not supported.")

	}
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
		return nil, errors.Errorf(errors.GeneralError, "The key algorithm is invalid. Supported options: %s", supportedAlgsMsg)
	}

}

//generateKeyWithOpts to generate key using BCCSP
func generateKeyWithOpts(channelID string, opts bccsp.KeyGenOpts) pb.Response {

	_, k, err := getBCCSPAndKeyPair(channelID, opts)
	if err != nil {
		return shim.Error(fmt.Sprintf("Got error from getBCCSPAndKeyPair in %v %v", opts, err))
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
			return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
		}
	} else {
		pubKey = k
	}
	pubKeyBts, err := pubKey.Bytes()
	if err != nil {
		return pb.Response{Payload: nil, Status: shim.ERROR, Message: err.Error()}
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
				logger.Debugf("Got error while creating config for channel %v\n", channelID)
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

	targetPeer, err := sdkpeer.New(txService.ClientConfig(), sdkpeer.WithURL(peerURL),
		sdkpeer.WithTLSCert(txService.Config.GetTLSRootCert()), sdkpeer.WithServerName(""))
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
		return configKey, errors.New(errors.GeneralError, "Config is empty (no args)")
	}

	configBytes := args[0]
	if len(configBytes) == 0 {
		logger.Error("Config is empty (no key)")
		return configKey, errors.New(errors.GeneralError, "Config is empty (no key)")
	}
	if err := json.Unmarshal(configBytes, &configKey); err != nil {
		logger.Errorf("Got error %v unmarshalling config key %s", err, string(configBytes[:]))
		return configKey, errors.Errorf(errors.GeneralError, "Got error %v unmarshalling config key %s", err, string(configBytes[:]))
	}

	return configKey, nil
}

// New chaincode implementation
func New() shim.Chaincode {
	return &ConfigurationSnap{}
}

func main() {
}
