/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	chmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/chmgmtclient"
	resmgmt "github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient"
	packager "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/ccpackager/gopackager"
	sdkFabricClientChannel "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/events"
	sdkorderer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"
	sdkpeer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	bccsputils "github.com/hyperledger/fabric/bccsp/utils"
	"github.com/pkg/errors"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
)

// CommonSteps contain BDDContext
type CommonSteps struct {
	BDDContext *BDDContext
}

//SnapTransactionRequest type will be passed as argument to a transaction snap
//ChannelID and ChaincodeID are mandatory fields
type SnapTransactionRequest struct {
	ChannelID           string            // required channel ID
	ChaincodeID         string            // required chaincode ID
	TransientMap        map[string][]byte // optional transient Map
	EndorserArgs        [][]byte          // optional args for endorsement
	CCIDsForEndorsement []string          // optional ccIDs For endorsement selection
	RegisterTxEvent     bool              // optional args for register Tx event (default is false)
}

var logger = logging.NewLogger("test-logger")

var trxPR []*apitxn.TransactionProposalResponse
var queryValue string
var queryResult string
var lastTxnID apitxn.TransactionID

// NewCommonSteps create new CommonSteps struct
func NewCommonSteps(context *BDDContext) *CommonSteps {
	return &CommonSteps{BDDContext: context}
}

// GetDeployPath ..
func (d *CommonSteps) getDeployPath(ccType string) string {
	// non "test" cc come from GOPATH
	if ccType != "test" {
		return os.Getenv("GOPATH")
	}

	// test cc come from fixtures
	pwd, _ := os.Getwd()
	return path.Join(pwd, "./fixtures")
}

// getEventHub initilizes the event hub
func (d *CommonSteps) getEventHub(client sdkApi.Resource) (sdkApi.EventHub, error) {
	eventHub, err := events.NewEventHub(client)
	if err != nil {
		return nil, fmt.Errorf("GetDefaultImplEventHub failed: %v", err)
	}
	peerConfig, err := d.BDDContext.resourceClients[d.BDDContext.Org1User].Config().PeerConfig("peerorg1", "peer0.org1.example.com")
	if err != nil {
		return nil, fmt.Errorf("Error reading peer config: %s", err)
	}
	serverHostOverride := ""
	if str, ok := peerConfig.GRPCOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}
	peerCert, err := peerConfig.TLSCACerts.TLSCert()
	if err != nil {
		return nil, fmt.Errorf("Error reading peer cert from the config: %s", err)
	}
	eventHub.SetPeerAddr(peerConfig.EventURL, peerCert, serverHostOverride)

	return eventHub, nil
}

func (d *CommonSteps) createChannelAndPeerJoinChannel(channelID string) error {
	// Get client Config
	config := d.BDDContext.resourceClients[d.BDDContext.Org1User].Config()
	//Get Channel
	channel, err := d.BDDContext.resourceClients[d.BDDContext.Org1User].NewChannel(channelID)
	if err != nil {
		return fmt.Errorf("Create channel (%s) failed: %v", channelID, err)
	}

	peerConfig, err := config.PeerConfig("peerorg1", "peer0.org1.example.com")
	if err != nil {
		return fmt.Errorf("Error reading peer config: %s", err)
	}
	mspID, err := config.MspID("peerorg1")
	if err != nil {
		return fmt.Errorf("Error getting peerorg1's mspID: %s", err)
	}
	peer, err := sdkpeer.New(config, sdkpeer.FromPeerConfig(&apiconfig.NetworkPeer{PeerConfig: *peerConfig, MspID: mspID}))
	if err != nil {
		return fmt.Errorf("New peer failed: %v", err)
	}
	channel.AddPeer(peer)

	ordererConfig, err := config.OrdererConfig("orderer.example.com")
	if err != nil {
		return fmt.Errorf("Could not load orderer config: %v", err)
	}

	orderer, err := sdkorderer.New(config, sdkorderer.FromOrdererConfig(ordererConfig))
	if err != nil {
		return fmt.Errorf("New rderer failed: %v", err)
	}
	channel.AddOrderer(orderer)

	d.BDDContext.Channel = channel

	// Check if primary peer has joined channel
	alreadyJoined, err := HasPrimaryPeerJoinedChannel(d.BDDContext.resourceClients[d.BDDContext.Org1Admin], d.BDDContext.Org1Admin, channel)
	if err != nil {
		return fmt.Errorf("Error while checking if primary peer has already joined channel: %v", err)
	}

	// Channel management client is responsible for managing channels (create/update)
	chMgmtClient, err := d.BDDContext.clients[d.BDDContext.OrdererAdmin].ChannelMgmt()
	if err != nil {
		return fmt.Errorf("Failed to create new channel management client: %s", err)
	}

	if !alreadyJoined {
		// Create and join channel
		req := chmgmt.SaveChannelRequest{ChannelID: channelID,
			ChannelConfig:   GetChannelTxPath(channelID),
			SigningIdentity: d.BDDContext.Org1Admin}

		if err = chMgmtClient.SaveChannel(req); err != nil {
			return errors.WithMessage(err, "SaveChannel failed")
		}
		time.Sleep(time.Second * 3)
		req = chmgmt.SaveChannelRequest{ChannelID: channelID,
			ChannelConfig:   GetChannelAnchorTxPath(channelID, "peerorg1"),
			SigningIdentity: d.BDDContext.Org1Admin}

		if err = chMgmtClient.SaveChannel(req); err != nil {
			return errors.WithMessage(err, "SaveChannel failed")
		}
		resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
		if err != nil {
			return fmt.Errorf("Failed to create new resource management client: %s", err)
		}
		if err = resMgmtClient.JoinChannel(channelID); err != nil {
			return fmt.Errorf("JoinChannel returned error: %v", err)
		}
	}
	return nil
}

func (d *CommonSteps) installAndInstantiateCC(ccType string, ccID string, version string, ccPath string, args string) error {
	org1AdminClient := d.BDDContext.resourceClients[d.BDDContext.Org1Admin]
	// Check if CC is installed
	installed, err := IsChaincodeInstalled(org1AdminClient, d.BDDContext.Channel.Peers()[0], ccID)
	if err != nil {
		return err
	}

	if installed {
		return nil
	}

	peers := d.BDDContext.Channel.Peers()
	var processors []apitxn.ProposalProcessor
	for _, peer := range peers {
		processors = append(processors, peer)
	}

	// SendInstallCC
	resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
	if err != nil {
		return fmt.Errorf("Failed to create new resource management client: %s", err)
	}

	ccPkg, err := packager.NewCCPackage(ccPath, d.getDeployPath(ccType))
	if err != nil {
		return err
	}

	installRqst := resmgmt.InstallCCRequest{Name: ccID, Path: ccPath, Version: version, Package: ccPkg}
	_, err = resMgmtClient.InstallCC(installRqst)
	if err != nil {
		return err
	}

	argsArray := strings.Split(args, ",")

	eventHub, err := d.getEventHub(org1AdminClient)
	if err != nil {
		return err
	}

	if err := eventHub.Connect(); err != nil {
		return fmt.Errorf("Failed eventHub.Connect() [%s]", err)
	}

	defer eventHub.Disconnect()

	instantiateRqst := resmgmt.InstantiateCCRequest{Name: ccID, Path: ccPath, Version: version, Args: GetByteArgs(argsArray), Policy: cauthdsl.SignedByMspMember("Org1MSP")}
	instantiateOpts := resmgmt.InstantiateCCOpts{
		Targets: peers,
	}
	err = resMgmtClient.InstantiateCCWithOpts(d.BDDContext.Channel.Name(), instantiateRqst, instantiateOpts)

	return err
}

func (d *CommonSteps) queryCCForError(ccID string, channelID string, args string) error {
	argsArray := strings.Split(args, ",")

	if channelID != "" && d.BDDContext.Channel.Name() != channelID {
		return fmt.Errorf("Channel(%s) not created", channelID)
	}

	var err error
	if channelID != "" {
		queryResult, err = d.queryChaincode(d.BDDContext.Channel, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())
	} else {
		queryResult, err = d.queryChaincode(nil, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())
	}
	if err == nil {
		return fmt.Errorf("Expected error here 'invoke Endorser  returned error....'")
	}

	return nil
}

func (d *CommonSteps) queryCC(ccID string, channelID string, args string) error {

	// Get Query value
	argsArray := strings.Split(args, ",")

	if len(argsArray) > 1 && argsArray[1] == "verifyTransactionProposalSignature" {
		signedProposalBytes, err := proto.Marshal(trxPR[0].Proposal.SignedProposal)
		if err != nil {
			return fmt.Errorf("Marshal SignedProposal return error: %v", err)
		}
		argsArray[3] = string(signedProposalBytes)
	}
	if len(argsArray) > 1 && argsArray[1] == "commitTransaction" {
		argsArray[3] = queryResult
	}
	if channelID != "" && d.BDDContext.Channel.Name() != channelID {
		return fmt.Errorf("Channel(%s) not created", channelID)
	}

	var err error
	if channelID != "" {
		queryResult, err = d.queryChaincode(d.BDDContext.Channel, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())
	} else {
		queryResult, err = d.queryChaincode(nil, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())
	}
	if err != nil {
		return fmt.Errorf("QueryChaincode return error: %v", err)
	}
	queryValue = queryResult
	if len(argsArray) > 1 && argsArray[1] == "endorseTransaction" {
		err := json.Unmarshal([]byte(queryResult), &trxPR)
		if err != nil {
			return fmt.Errorf("Unmarshal(%s) to TransactionProposalResponse return error: %v", queryValue, err)
		}
		queryValue = string(trxPR[0].ProposalResponse.GetResponse().Payload)
	}

	logger.Debugf("QueryChaincode return value: %s", queryValue)

	return nil
}

func (d *CommonSteps) invokeCC(ccID string, channelID string, args string) error {

	// Get Query value
	argsArray := strings.Split(args, ",")

	if channelID != "" && d.BDDContext.Channel.Name() != channelID {
		return fmt.Errorf("Channel(%s) not created", channelID)
	}

	err := d.invokeChaincode(d.BDDContext.resourceClients[d.BDDContext.Org1User], d.BDDContext.Channel, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())

	if err != nil {
		return fmt.Errorf("invokeChaincode return error: %v", err)
	}

	return nil
}

func (d *CommonSteps) checkQueryValue(value string, ccID string) error {
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if queryValue != value {
		return fmt.Errorf("Query value(%s) is not equal to the expected value(%s)", queryValue, value)
	}

	return nil
}

func (d *CommonSteps) copyConfigFile(src, dest string) error {
	logger.Debugf("copying config files %s %s\n", src, dest)
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return fmt.Errorf("%v", err)
	}
	err = out.Sync()
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	logger.Debugf("Config was copied\n")
	return nil
}
func (d *CommonSteps) containsInQueryValue(ccID string, value string) error {
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	logger.Debugf("Query value %s and tested value %s", queryValue, value)
	if !strings.Contains(queryValue, value) {
		return fmt.Errorf("Query value(%s) doesn't contain expected value(%s)", queryValue, value)
	}
	return nil
}

//checkCSR to verify that CSR was created
func (d *CommonSteps) checkCSR(ccID string) error {
	//key bytes returned
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if strings.Contains(queryValue, "Error") {
		return fmt.Errorf("QueryValue contains error: %s", queryValue)
	}
	//response contains public key bytes
	raw := []byte(queryValue)
	csr := pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: raw,
	})

	logger.Debugf("CSR was created \n%v\n", string(csr))
	//returned certificate request should have fields configured in config.yaml
	c, e := x509.ParseCertificateRequest(raw)
	if e != nil {
		return e
	}
	if c.Subject.Organization[0] == "" {
		return errors.Errorf("CSR should have non nil subject-organization")
	}
	logger.Debugf("CSR was created \n%v\n", c.Subject)
	return nil
}

func (d *CommonSteps) checkKeyGenResponse(ccID string, expectedKeyType string) error {
	//key bytes returned
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if strings.Contains(queryValue, "Error") {
		return fmt.Errorf("QueryValue contains error: %s", queryValue)
	}
	//response contains public key bytes
	raw := []byte(queryValue)
	pk, err := bccsputils.DERToPublicKey(raw)
	if err != nil {
		return errors.Wrap(err, "failed marshalling der to public key")
	}
	switch k := pk.(type) {
	case *ecdsa.PublicKey:
		if !strings.Contains(expectedKeyType, "ECDSA") {
			return errors.Errorf("Expected ECDSA key but got %v", k)
		}
		ecdsaPK, ok := pk.(*ecdsa.PublicKey)
		if !ok {
			return errors.New("failed casting to ECDSA public key. Invalid raw material")
		}
		ecPt := elliptic.Marshal(ecdsaPK.Curve, ecdsaPK.X, ecdsaPK.Y)
		hash := sha256.Sum256(ecPt)
		ski := hash[:]
		if len(ski) == 0 {
			return errors.New("Expected valid SKI for PK")
		}

	case *rsa.PublicKey:
		if !strings.Contains(expectedKeyType, "RSA") {
			return errors.Errorf("Expected RSA key but got %v", k)
		}
		rsaPK, ok := pk.(*rsa.PublicKey)
		if !ok {
			return errors.New("failed casting to RSA public key. Invalid raw material")
		}
		PubASN1, err := x509.MarshalPKIXPublicKey(rsaPK)
		if err != nil {
			return err
		}
		if len(PubASN1) == 0 {
			return errors.New("Invalid RSA key")
		}
	default:
		logger.Debugf("Not supported %v", k)
		return errors.Errorf("Received unsupported key type")
	}

	return nil
}

// createAndSendTransactionProposal ...
func (d *CommonSteps) createAndSendTransactionProposal(channel sdkApi.Channel, chainCodeID string,
	args []string, targets []apitxn.ProposalProcessor, transientData map[string][]byte) ([]*apitxn.TransactionProposalResponse, apitxn.TransactionID, error) {

	request := apitxn.ChaincodeInvokeRequest{
		Targets:      targets,
		Fcn:          args[0],
		Args:         GetByteArgs(args[1:]),
		TransientMap: transientData,
		ChaincodeID:  chainCodeID,
	}
	var transactionProposalResponses []*apitxn.TransactionProposalResponse
	var txnID apitxn.TransactionID
	var err error
	if channel == nil {
		transactionProposalResponses, txnID, err = sdkFabricClientChannel.SendTransactionProposalWithChannelID("", request, d.BDDContext.resourceClients[d.BDDContext.Org1User])
	} else {
		transactionProposalResponses, txnID, err = channel.SendTransactionProposal(request)
	}
	if err != nil {
		return nil, txnID, err
	}

	for _, v := range transactionProposalResponses {
		if v.Err != nil {
			return nil, txnID, fmt.Errorf("invoke Endorser %s returned error: %v", v.Endorser, v.Err)
		}
		if v.ProposalResponse.Response.Status != 200 {
			return nil, txnID, fmt.Errorf("invoke Endorser %s returned status: %v", v.Endorser, v.ProposalResponse.Response.Status)
		}
	}

	return transactionProposalResponses, txnID, nil
}

func (d *CommonSteps) createTransactionSnapRequest(functionName string, chaincodeID string, chnlID string, clientArgs []string, registerTxEvent bool) []string {

	endorserArgs := make([][]byte, len(clientArgs))
	for i, v := range clientArgs {
		endorserArgs[i] = []byte(v)

	}
	snapTxReq := SnapTransactionRequest{ChannelID: chnlID,
		ChaincodeID:         chaincodeID,
		TransientMap:        nil,
		EndorserArgs:        endorserArgs,
		CCIDsForEndorsement: nil,
		RegisterTxEvent:     registerTxEvent}
	snapTxReqB, _ := json.Marshal(snapTxReq)

	var args []string
	args = append(args, functionName)
	args = append(args, string(snapTxReqB))
	return args
}

//queryChaincode ...
func (d *CommonSteps) queryChaincode(channel sdkApi.Channel, chaincodeID string,
	args []string, primaryPeer sdkApi.Peer) (string, error) {
	transactionProposalResponses, _, err := d.createAndSendTransactionProposal(channel,
		chaincodeID, args, []apitxn.ProposalProcessor{primaryPeer}, nil)

	if err != nil {
		return "", fmt.Errorf("CreateAndSendTransactionProposal returned error: %v", err)
	}

	return string(transactionProposalResponses[0].ProposalResponse.GetResponse().Payload), nil
}

func (d *CommonSteps) loadConfig(channelID string, snaps string) error {
	if channelID != "" && d.BDDContext.Channel.Name() != channelID {
		return fmt.Errorf("Channel(%s) not created", channelID)
	}
	snapsArray := strings.Split(snaps, ",")
	for _, snap := range snapsArray {
		var argsArray []string
		configData, err := ioutil.ReadFile(fmt.Sprintf("./fixtures/config/snaps/%s/config.yaml", snap))
		if err != nil {
			return fmt.Errorf("file error: %v", err)
		}
		config := &configmanagerApi.ConfigMessage{MspID: "Org1MSP", Peers: []configmanagerApi.PeerConfig{configmanagerApi.PeerConfig{PeerID: "peer0.org1.example.com", App: []configmanagerApi.AppConfig{configmanagerApi.AppConfig{AppName: snap, Config: string(configData)}}}}}
		configBytes, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("cannot Marshal %s", err)
		}
		argsArray = append(argsArray, "save")
		argsArray = append(argsArray, string(configBytes))
		err = d.invokeChaincode(d.BDDContext.resourceClients[d.BDDContext.Org1User], d.BDDContext.Channel, "configurationsnap", argsArray, d.BDDContext.Channel.PrimaryPeer())
		if err != nil {
			return fmt.Errorf("invokeChaincode return error: %v", err)
		}

	}
	return nil
}

// RegisterTxEvent registers on the given eventhub for the give transaction
// returns a boolean channel which receives true when the event is complete
// and an error channel for errors
func (d *CommonSteps) RegisterTxEvent(txID apitxn.TransactionID, eventHub sdkApi.EventHub) (chan bool, chan error) {
	done := make(chan bool)
	fail := make(chan error)

	eventHub.RegisterTxEvent(txID, func(txId string, errorCode pb.TxValidationCode, err error) {
		if err != nil {
			fail <- err
		} else {
			done <- true
		}
	})

	return done, fail
}

//invokeChaincode ...
func (d *CommonSteps) invokeChaincode(client sdkApi.Resource, channel sdkApi.Channel, chaincodeID string,
	args []string, primaryPeer sdkApi.Peer) error {
	transactionProposalResponses, txID, err := d.createAndSendTransactionProposal(channel,
		chaincodeID, args, []apitxn.ProposalProcessor{primaryPeer}, nil)

	if err != nil {
		return fmt.Errorf("CreateAndSendTransactionProposal returned error: %v", err)
	}

	tx, err := channel.CreateTransaction(transactionProposalResponses)
	if err != nil {
		return errors.WithMessage(err, "CreateTransaction failed")
	}

	transactionResponse, err := channel.SendTransaction(tx)
	if err != nil {
		return errors.WithMessage(err, "SendTransaction failed")

	}

	eventHub, err := d.getEventHub(client)
	if err != nil {
		return err
	}

	if err := eventHub.Connect(); err != nil {
		return fmt.Errorf("Failed eventHub.Connect() [%s]", err)
	}

	defer eventHub.Disconnect()

	// Register for commit event
	done, fail := d.RegisterTxEvent(txID, eventHub)

	if transactionResponse.Err != nil {
		return errors.Wrapf(transactionResponse.Err, "orderer %s failed", transactionResponse.Orderer)
	}
	select {
	case <-done:
	case cerr := <-fail:
		return errors.Wrapf(cerr, "invoke failed for txid %s", txID)
	case <-time.After(time.Second * 30):
		return errors.Errorf("invoke didn't receive block event for txid %s", txID)
	}
	return nil

}

func (d *CommonSteps) wait(seconds int) error {
	logger.Infof("Waiting [%d] seconds\n", seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (d *CommonSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.beforeScenario)
	s.AfterScenario(d.BDDContext.afterScenario)
	s.Step(`^fabric has channel "([^"]*)" and p0 joined channel$`, d.createChannelAndPeerJoinChannel)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" version "([^"]*)" from path "([^"]*)" is installed and instantiated with args "([^"]*)"$`, d.installAndInstantiateCC)
	s.Step(`^client C1 query chaincode "([^"]*)" on channel "([^"]*)" with args "([^"]*)" on p0$`, d.queryCC)
	s.Step(`^C1 receive value "([^"]*)" from "([^"]*)"$`, d.checkQueryValue)
	s.Step(`^response from "([^"]*)" to client C1 contains value "([^"]*)"$`, d.containsInQueryValue)
	s.Step(`^client C1 invokes configuration snap on channel "([^"]*)" to load "([^"]*)" configuration on p0$`, d.loadConfig)
	s.Step(`^client C1 invokes chaincode "([^"]*)" on channel "([^"]*)" with args "([^"]*)" on p0$`, d.invokeCC)
	s.Step(`^client C1 waits (\d+) seconds$`, d.wait)
	s.Step(`^client C1 copies "([^"]*)" to "([^"]*)"$`, d.copyConfigFile)
	s.Step(`^client C1 query chaincode with error "([^"]*)" on channel "([^"]*)" with args "([^"]*)" on p0$`, d.queryCCForError)
	s.Step(`^response from "([^"]*)" to client C1 has key and key type is "([^"]*)" on p0$`, d.checkKeyGenResponse)
	s.Step(`^response from "([^"]*)" to client C1 has CSR on p0$`, d.checkCSR)

}
