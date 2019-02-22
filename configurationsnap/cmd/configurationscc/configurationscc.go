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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/securekey/fabric-snaps/util/rolesmgr"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/signer"
	acl "github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/peer"
	protosMSP "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configmanager/pkg/mgmt"
	configmgmtService "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	cfgsnapapi "github.com/securekey/fabric-snaps/configurationsnap/api"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/config"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configurationscc/listener"
	"github.com/securekey/fabric-snaps/healthcheck"
	metricsutil "github.com/securekey/fabric-snaps/metrics/pkg/util"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
	"github.com/securekey/fabric-snaps/util"
	"github.com/securekey/fabric-snaps/util/errors"
)

// functionRegistry is a registry of the functions that are supported by configuration snap
var functionRegistry = map[string]func(shim.ChaincodeStubInterface, [][]byte, *Metrics) pb.Response{
	"healthCheck":     healthCheck,
	"save":            save,
	"get":             get,
	"getFromCache":    getFromCache,
	"delete":          delete,
	"refresh":         refresh,
	"generateKeyPair": generateKeyPair,
	"generateCSR":     generateCSR,
}

// signatureRegistry is a registry of the Signature Algorithms supported by configuration snap
var signatureRegistry = map[string]x509.SignatureAlgorithm{
	"ECDSAWithSHA1":    x509.ECDSAWithSHA1,
	"ECDSAWithSHA256":  x509.ECDSAWithSHA256,
	"ECDSAWithSHA384":  x509.ECDSAWithSHA384,
	"ECDSAWithSHA512":  x509.ECDSAWithSHA512,
	"SHA256WithRSAPSS": x509.SHA256WithRSAPSS,
	"SHA384WithRSAPSS": x509.SHA384WithRSAPSS,
	"SHA512WithRSAPSS": x509.SHA512WithRSAPSS,
	"DSAWithSHA256":    x509.DSAWithSHA256,
	"DSAWithSHA1":      x509.DSAWithSHA1,
	"SHA512WithRSA":    x509.SHA512WithRSA,
	"SHA384WithRSA":    x509.SHA384WithRSA,
	"SHA256WithRSA":    x509.SHA256WithRSA,
	"SHA1WithRSA":      x509.SHA1WithRSA,
	"MD5WithRSA":       x509.MD5WithRSA,
	"MD2WithRSA":       x509.MD2WithRSA,
}
var supportedAlgs = []string{"ECDSA", "ECDSAP256", "ECDSAP384", "RSA", "RSA1024", "RSA2048", "RSA3072", "RSA4096"}
var availableFunctions = functionSet()

var logger = logging.NewLogger("configsnap")

// ConfigurationSnap implementation
type ConfigurationSnap struct {
	metrics *Metrics
}

var peerConfigPath = ""

// aclProvider is used to check ACL
var aclProvider acl.ACLProvider

const (
	// configDataReadACLPrefix is the prefix for read-only (get) policy resource names
	configDataReadACLPrefix = "configdata/read/"

	// configDataWriteACLPrefix is the prefix for the write (save, delete) policy resource names
	configDataWriteACLPrefix = "configdata/write/"

	// configSnapName is the cc name used for event source
	configSnapName = "configurationsnap"

	// connectBackoff is the time to back off for connecting to event listener
	connectBackoff = time.Second
)

// Init snap
func (configSnap *ConfigurationSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debugf("******** Init Config Snap on channel [%s]\n", stub.GetChannelID())
	if stub.GetChannelID() != "" {
		if !rolesmgr.HasEndorserRole() {
			logger.Infof("Not starting Config Snap on channel [%s] since this peer is not an endorser", stub.GetChannelID())
			return shim.Success(nil)
		}

		logger.Infof("******** Init Config Snap on channel [%s]", stub.GetChannelID())
		peerMspID, err := config.GetPeerMSPID(peerConfigPath)
		if err != nil {
			return util.CreateShimResponseFromError(errors.WithMessage(errors.InitializeSnapError, err, "Error initializing Configuration Snap"), logger, stub)
		}
		peerID, err := config.GetPeerID(peerConfigPath)
		if err != nil {
			return util.CreateShimResponseFromError(errors.WithMessage(errors.InitializeSnapError, err, "Error initializing Configuration Snap"), logger, stub)
		}
		interval := config.GetDefaultRefreshInterval()
		logger.Debugf("******** Call initialize for [%s][%s][%v]", peerMspID, peerID, interval)
		configmgmtService.Initialize(stub, peerMspID)

		eventSource := &listener.EventSource{
			ChannelID:   stub.GetChannelID(),
			ChaincodeID: configSnapName,
			EventName:   cfgsnapapi.ConfigCCEventName,
		}

		updateListener, err1 := listener.NewChaincodeListener(eventSource)
		if err1 != nil {
			return util.CreateShimResponseFromError(errors.WithMessage(errors.InitializeSnapError, err1, "Error initializing Configuration Snap"), logger, stub)
		}

		go listenConfigEvents(stub.GetChannelID(), updateListener, configSnap.metrics)

		periodicRefresh(stub.GetChannelID(), interval, configSnap.metrics)
	}
	return shim.Success(nil)
}

func listenConfigEvents(channelID string, updateListener listener.ChaincodeListener, metrics *Metrics) {
	for {
		eventChannel, err := updateListener.Listen()
		if err != nil {
			e := errors.Errorf(errors.SystemError, "Error listening for updates on channel [%s]: %s", channelID, err)
			logger.Warnf(e.Error())
			time.Sleep(connectBackoff)
			continue
		}

		for updateEvent := range eventChannel {
			if updateEvent == nil {
				continue
			}

			go sendRefreshRequest(channelID, metrics)
		}
	}
}

// Invoke is the main entry point for invocations
func (configSnap *ConfigurationSnap) Invoke(stub shim.ChaincodeStubInterface) (resp pb.Response) {

	defer util.HandlePanic(&resp, logger, stub)

	args := stub.GetArgs()
	if len(args) == 0 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, fmt.Sprintf("Function not provided. Expecting one of (%s)", availableFunctions)), logger, stub)
	}

	functionName := string(args[0])
	function, ok := functionRegistry[functionName]
	if !ok {
		return util.CreateShimResponseFromError(errors.New(errors.InvalidFunctionError, fmt.Sprintf("Invalid function: %s. Expecting one of (%s)", functionName, availableFunctions)), logger, stub)
	}

	functionArgs := args[1:]

	logger.Debugf("Invoking function [%s] with args: %s", functionName, functionArgs)
	return function(stub, functionArgs, configSnap.metrics)
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
func healthCheck(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {
	response := healthcheck.SmokeTest(healthcheck.ConfigurationScc, stub, args)
	if response.Status != shim.OK {
		errObj := errors.New(errors.SystemError, fmt.Sprintf("%s healthcheck failed: %s", healthcheck.ConfigurationScc, response.Message))
		logger.Error(errObj.GenerateLogMsg())
		return util.CreateShimResponseFromError(errObj, logger, stub)
	}
	return shim.Success(response.Payload)
}

//checkACLforKey - checks acl for the given config key
func checkACLforKey(stub shim.ChaincodeStubInterface, configKey *mgmtapi.ConfigKey, aclResourcePrefix string) errors.Error {
	if configKey.MspID == "" {
		return errors.New(errors.SystemError, "ACL check failed, config has empty msp")
	}

	resourceName := aclResourcePrefix + configKey.MspID

	logger.Debugf("Checking ACL for resource: %v", resourceName)

	sp, err := stub.GetSignedProposal()
	if err != nil {
		return errors.WithMessage(errors.SystemError, err, "ACL check failed, error getting signed proposal")
	}

	mspID, codedErr := getMspID(stub)
	if codedErr != nil {
		return codedErr
	}

	err = getACLProvider().CheckACL(resourceName, stub.GetChannelID(), sp)
	if err != nil {
		logger.Debugf("ACL check failed for resource: %s, with signing mspID: %s", resourceName, mspID)
		return errors.WithMessage(errors.ACLCheckError, err, fmt.Sprintf("ACL check failed for resource %s and mspID %s", resourceName, mspID))
	}

	return nil
}

//getMspID as a string from the creator of signed proposal
func getMspID(stub shim.ChaincodeStubInterface) (string, errors.Error) {
	creator, err := stub.GetCreator()
	if err != nil {
		return "", errors.WithMessage(errors.SystemError, err, "failed to get creator bytes")
	}
	sid := &protosMSP.SerializedIdentity{}
	if err := proto.Unmarshal(creator, sid); err != nil {
		return "", errors.WithMessage(errors.UnmarshalError, err, "failed to unmarshal creator")
	}
	return sid.Mspid, nil
}

//save - saves configuration passed in args
func save(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {
	configMsg := args[0]
	if len(configMsg) == 0 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "Config is empty-cannot be saved"), logger, stub)
	}

	// parse config message for ACL check
	configMessageMap, err := mgmt.ParseConfigMessage(configMsg, stub.GetTxID())
	if err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}

	for key := range configMessageMap {
		if err := checkACLforKey(stub, &key, configDataWriteACLPrefix); err != nil {
			return util.CreateShimResponseFromError(err, logger, stub)
		}
	}

	cmngr := mgmt.NewConfigManager(stub)
	err = cmngr.Save(configMsg)
	if err != nil {
		logger.Errorf("Got error while saving config: %s", err.GenerateLogMsg())
		return util.CreateShimResponseFromError(err, logger, stub)
	}

	return shim.Success(nil)
}

//get - gets configuration using configkey as criteria
func get(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {

	configKey, codedErr := getKey(args)
	if codedErr != nil {
		return util.CreateShimResponseFromError(codedErr, logger, stub)
	}

	if codedErr := checkACLforKey(stub, configKey, configDataReadACLPrefix); codedErr != nil {
		return util.CreateShimResponseFromError(codedErr, logger, stub)
	}

	//valid key
	cmngr := mgmt.NewConfigManager(stub)
	config, codedErr := cmngr.Get(*configKey)
	if codedErr != nil {
		logger.Errorf("Get for key %+v returns error: %s", configKey, codedErr.GenerateLogMsg())
		return util.CreateShimResponseFromError(errors.WithMessage(errors.GetConfigError, codedErr, fmt.Sprintf("Failed to retrieve config for config key %+v", configKey)), logger, stub)
	}

	payload, err := json.Marshal(config)
	if err != nil {
		errObj := errors.WithMessage(errors.SystemError, err, "Failed to marshal config")
		logger.Errorf(errObj.GenerateLogMsg())
		return util.CreateShimResponseFromError(errObj, logger, stub)

	}

	return shim.Success(payload)
}

//delete - deletes configuration using config key as criteria
func delete(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {

	configKey, err := getKey(args)
	if err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}

	if err := checkACLforKey(stub, configKey, configDataWriteACLPrefix); err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}

	//valid key
	cmngr := mgmt.NewConfigManager(stub)
	if err := cmngr.Delete(*configKey); err != nil {
		logger.Errorf("Got error while deleting config: %s", err.GenerateLogMsg())
		return util.CreateShimResponseFromError(err, logger, stub)

	}
	return shim.Success(nil)
}

func refresh(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {
	startTime := time.Now()
	defer func() { metrics.ConfigRefresh.Observe(time.Since(startTime).Seconds()) }()

	if len(args) < 1 {
		return util.CreateShimResponseFromError(errors.New(errors.MissingRequiredParameterError, "expecting first arg to be a JSON array of MSP IDs"), logger, stub)
	}

	var msps []string
	err := json.Unmarshal(args[0], &msps)
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.UnmarshalError, err, "Failed to unmarshal msp IDs"), logger, stub)
	}

	peerMspID, err := config.GetPeerMSPID(peerConfigPath)
	if err != nil {
		return util.CreateShimResponseFromError(errors.WithMessage(errors.UnmarshalError, err, "Failed to get peer msp ID"), logger, stub)
	}

	// ACL check
	if err := checkACLforKey(stub, &mgmtapi.ConfigKey{MspID: peerMspID}, configDataReadACLPrefix); err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}

	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)

	for _, msp := range msps {
		logger.Debugf("****** Refresh msp id %s", msp)
		instance.Refresh(stub, msp)
	}
	instance.Refresh(stub, cfgsnapapi.GeneralMspID)

	return shim.Success(nil)
}

//getFromCache - gets configuration using configkey as criteria from cache
func getFromCache(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {

	configKey, err := getKey(args)
	if err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}

	if err := checkACLforKey(stub, configKey, configDataReadACLPrefix); err != nil {
		return util.CreateShimResponseFromError(err, logger, stub)
	}
	//valid key
	x := configmgmtService.GetInstance()
	instance := x.(*configmgmtService.ConfigServiceImpl)
	config, err := instance.GetFromCache(stub.GetChannelID(), *configKey)
	if err != nil {
		logger.Errorf("Get for key %+v returns error: %s", configKey, err.GenerateLogMsg())
		return util.CreateShimResponseFromError(err, logger, stub)
	}
	return shim.Success(config)
}

//to generate key pair based on options submitted
//expected keytype and ephemeral flag in args
func generateKeyPair(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {
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
func generateCSR(stub shim.ChaincodeStubInterface, args [][]byte, metrics *Metrics) pb.Response {
	//check args
	if len(args) < 4 {
		return shim.Error(fmt.Sprintf("Required arguments are: [key type,ephemeral flag, CSR's signature algorithm and common name"))
	}
	keyType := string(args[0])
	ephemeral, err := strconv.ParseBool(string(args[1]))
	if err != nil {
		return shim.Error(fmt.Sprintf("Ephemeral flag is not set"))
	}
	sigAlgType := string(args[2])
	csrCommonName := string(args[3])
	//get requested key options
	options, err := getKeyOpts(keyType, ephemeral)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Debugf("Keygen options %+v", options)
	bccspsuite, keys, err := getBCCSPAndKeyPair(stub.GetChannelID(), options)
	if err != nil {
		return shim.Error(err.Error())
	}
	csrTemplate, err := getCSRTemplate(stub.GetChannelID(), keys, keyType, sigAlgType, csrCommonName)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Debugf("Certificate request template %+v", csrTemplate)
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

func getCSRTemplate(channelID string, keys bccsp.Key, keyType string, sigAlgType string, csrCommonName string) (x509.CertificateRequest, error) {

	var csrTemplate x509.CertificateRequest
	sigAlg, err := getSignatureAlg(sigAlgType)
	if err != nil {
		return csrTemplate, err
	}
	//generate subject for CSR
	asn1Subj, err := getCSRSubject(channelID, csrCommonName)
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
		logger.Debugf("Get error parsing public key %s", err)
		return csrTemplate, err
	}
	pubKeyAlg, err := getPublicKeyAlg(keyType)
	if err != nil {
		logger.Debugf("Get error parsing public key alg %s", err)
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
	logger.Debugf("Certificate request template %+v", csrTemplate)
	return csrTemplate, nil

}

func getCSRSubject(channelID string, csrCommonName string) ([]byte, error) {
	if channelID == "" {
		return nil, errors.Errorf(errors.GeneralError, "Channel is required")
	}
	//get csr configuration - from config(HL)
	csrConfig, err := getCSRConfig(channelID, peerConfigPath)
	if err != nil {
		return nil, err
	}
	logger.Debugf("csrConfig options %+v", csrConfig)
	subj := pkix.Name{
		CommonName:         csrCommonName,
		Country:            []string{csrConfig.Country},
		Province:           []string{csrConfig.StateProvince},
		Locality:           []string{csrConfig.Locality},
		Organization:       []string{csrConfig.Org},
		OrganizationalUnit: []string{csrConfig.OrgUnit},
	}
	logger.Debugf("Subject options %+v", subj)

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

	bccspProvider, err := config.GetBCCSPProvider(peerConfigPath)
	if err != nil {
		return bccspsuite, k, err
	}
	logger.Debugf("***Configured BCCSP provider's ID is %s", bccspProvider)
	bccspsuite, err = factory.GetBCCSP(bccspProvider)
	if err != nil {
		logger.Debugf("Error getting BCCSP based on provider ID %s %s", bccspProvider, err)
		return bccspsuite, k, errors.Wrap(errors.GeneralError, err, "BCCSP Initialize failed")
	}
	logger.Debugf("***Configured BCCSP provider is %s", reflect.TypeOf(bccspsuite))
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
	var signatureAlgorithm x509.SignatureAlgorithm

	if signatureAlgorithm, ok := signatureRegistry[algorithm]; ok {
		return signatureAlgorithm, nil
	}
	err := errors.New(errors.GeneralError, "Alg is not supported.")
	return signatureAlgorithm, err
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
		return shim.Error(fmt.Sprintf("Got error from getBCCSPAndKeyPair in %+v %s", opts, err))
	}
	return parseKey(k)
}

//pass generated key (private/public) and return public to caller
func parseKey(k bccsp.Key) pb.Response {
	//logger.Debugf("Parsing key %v", k)
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

func periodicRefresh(channelID string, refreshInterval time.Duration, metrics *Metrics) {
	logger.Debugf("***Periodic refresh was called on [%d]\n", refreshInterval)
	go func() {
		for {
			time.Sleep(refreshInterval)
			sendRefreshRequest(channelID, metrics)
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

func sendRefreshRequest(channelID string, metrics *Metrics) {
	startTime := time.Now()
	defer func() { metrics.ConfigPeriodicRefresh.Observe(time.Since(startTime).Seconds()) }()

	//call to get snaps config from ledger and to initilaize cahce instance
	txService, err := txsnapservice.Get(channelID)
	if err != nil {
		logger.Debugf("Cannot get txService: %v", err)
		return
	}
	sendEndorseRequest(channelID, txService)
}

func sendEndorseRequest(channelID string, txService *txsnapservice.TxServiceImpl) {
	targetPeer, err := txService.GetLocalPeer()
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, "Error creating target peer when sending refresh request").GenerateLogMsg())
		return
	}

	chConfig, err := txService.FcClient.ChannelConfig()
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, "Error getting channel config").GenerateLogMsg())
		return
	}

	var mspIDs []string
	for _, mspConfig := range chConfig.MSPs() {
		fabricMSPConfig := &protosMSP.FabricMSPConfig{}
		err := proto.Unmarshal(mspConfig.Config, fabricMSPConfig)
		if err != nil {
			logger.Errorf(errors.WithMessage(errors.SystemError, err, "Error unmarshalling MSP config").GenerateLogMsg())
		}
		mspIDs = append(mspIDs, fabricMSPConfig.Name)
	}

	mspIDsBytes, err := json.Marshal(mspIDs)
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, "Error marshalling JSON args").GenerateLogMsg())
		return
	}

	args := [][]byte{[]byte("refresh"), mspIDsBytes}
	txSnapReq := createTransactionSnapRequest("configurationsnap", channelID, args, nil, nil)

	_, err = txService.EndorseTransaction(txSnapReq, []fabApi.Peer{targetPeer})
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, "configuration snap 'referesh' failed").GenerateLogMsg())
		return
	}
}

func createTransactionSnapRequest(chaincodeID string, chnlID string,
	endorserArgs [][]byte, transientMap map[string][]byte,
	ccIDsForEndorsement []string) *api.SnapTransactionRequest {

	return &api.SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        transientMap,
		EndorserArgs:        endorserArgs,
		CCIDsForEndorsement: ccIDsForEndorsement}

}

//getKey gets config key from args
func getKey(args [][]byte) (*mgmtapi.ConfigKey, errors.Error) {
	configKey := &mgmtapi.ConfigKey{}
	if len(args) == 0 {
		return configKey, errors.New(errors.MissingRequiredParameterError, "Config is empty (no args)")
	}

	configBytes := args[0]
	if len(configBytes) == 0 {
		return configKey, errors.New(errors.MissingRequiredParameterError, "Config is empty (no key)")
	}
	if err := json.Unmarshal(configBytes, &configKey); err != nil {
		errObj := errors.WithMessage(errors.UnmarshalError, err, fmt.Sprintf("Got an error unmarshalling config key %s", string(configBytes[:])))
		logger.Errorf(errObj.GenerateLogMsg())
		return configKey, errObj
	}

	return configKey, nil
}

// getACLProvider gets the ACLProvider used for ACL checks
func getACLProvider() acl.ACLProvider {
	// always nil except for unit tests
	if aclProvider != nil {
		return aclProvider
	}

	return acl.NewACLProvider(peer.GetStableChannelConfig)
}

// New chaincode implementation
func New() shim.Chaincode {
	return &ConfigurationSnap{metrics: NewMetrics(metricsutil.GetMetricsInstance())}
}

func main() {
}
