/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	sdkFabricClientChannel "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/channel"
	sdkFabricTxn "github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn"

	sdkFabricTxnAdmin "github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn/admin"
	"github.com/hyperledger/fabric/common/cauthdsl"
	logging "github.com/op/go-logging"

	"github.com/DATA-DOG/godog"
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

var logger = logging.MustGetLogger("test-logger")

var trxPR []*apitxn.TransactionProposalResponse
var queryValue string
var queryResult string
var peer0Address = "localhost:7051"
var ordererAddress = "localhost:7050"
var peer0EventAddress = "localhost:7053"
var peer0TlsCert = "./fixtures/channel/crypto-config/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem"
var ordererTlSCert = "./fixtures/channel/crypto-config/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem"

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
func (d *CommonSteps) getEventHub() (sdkApi.EventHub, error) {
	eventHub, err := sdkFabApi.NewEventHub(d.BDDContext.Client)
	if err != nil {
		return nil, fmt.Errorf("GetDefaultImplEventHub failed: %v", err)
	}
	eventHub.SetPeerAddr(peer0EventAddress,
		peer0TlsCert, "peer0.org1.example.com")

	return eventHub, nil
}

func (d *CommonSteps) createChannelAndPeerJoinChannel(channelID string) error {
	//Get Channel
	channel, err := d.BDDContext.Client.NewChannel(channelID)
	if err != nil {
		return fmt.Errorf("Create channel (%s) failed: %v", channelID, err)
	}

	peer, err := sdkFabApi.NewPeer(peer0Address,
		peer0TlsCert, "peer0.org1.example.com", d.BDDContext.Client.Config())
	if err != nil {
		return fmt.Errorf("NewPeer failed: %v", err)
	}
	channel.AddPeer(peer)

	orderer, err := sdkFabApi.NewOrderer(ordererAddress,
		ordererTlSCert, "orderer.example.com", d.BDDContext.Client.Config())
	if err != nil {
		return fmt.Errorf("NewPeer failed: %v", err)
	}
	channel.AddOrderer(orderer)

	d.BDDContext.Channel = channel

	// Check if primary peer has joined channel
	alreadyJoined, err := HasPrimaryPeerJoinedChannel(d.BDDContext.Client, d.BDDContext.Org1Admin, channel)
	if err != nil {
		return fmt.Errorf("Error while checking if primary peer has already joined channel: %v", err)
	}

	if !alreadyJoined {
		// Create and join channel
		if err = sdkFabricTxnAdmin.CreateOrUpdateChannel(d.BDDContext.Client, d.BDDContext.OrdererAdmin, d.BDDContext.Org1Admin, channel, fmt.Sprintf("./fixtures/channel/%s.tx", channelID)); err != nil {
			return fmt.Errorf("CreateOrUpdateChannel returned error: %v", err)
		}

		time.Sleep(time.Second * 3)
		if err = sdkFabricTxnAdmin.CreateOrUpdateChannel(d.BDDContext.Client, d.BDDContext.OrdererAdmin, d.BDDContext.Org1Admin, channel, fmt.Sprintf("./fixtures/channel/%s.tx", "Org1MSPanchors")); err != nil {
			return fmt.Errorf("CreateChannel returned error: %v", err)
		}
		if err = sdkFabricTxnAdmin.JoinChannel(d.BDDContext.Client, d.BDDContext.Org1Admin, channel); err != nil {
			return fmt.Errorf("JoinChannel returned error: %v", err)
		}
	}
	return nil
}

func (d *CommonSteps) installAndInstantiateCC(ccType string, ccID string, version string, ccPath string, args string) error {
	// installCC requires AdminUser privileges so setting user context with Admin User
	d.BDDContext.Client.SetUserContext(d.BDDContext.Org1Admin)
	// must reset client user context to normal user once done with Admin privilieges
	defer d.BDDContext.Client.SetUserContext(d.BDDContext.Org1User)
	// SendInstallCC
	if err := sdkFabricTxnAdmin.SendInstallCC(d.BDDContext.Client,
		ccID, ccPath, version, nil, d.BDDContext.Channel.Peers(), d.getDeployPath(ccType)); err != nil {
		return fmt.Errorf("SendInstallProposal return error: %v", err)
	}

	argsArray := strings.Split(args, ",")

	eventHub, err := d.getEventHub()
	if err != nil {
		return err
	}

	if err := eventHub.Connect(); err != nil {
		return fmt.Errorf("Failed eventHub.Connect() [%s]", err)
	}

	defer eventHub.Disconnect()

	if err := sdkFabricTxnAdmin.SendInstantiateCC(d.BDDContext.Channel, ccID, argsArray,
		ccPath, version, cauthdsl.SignedByMspMember("Org1MSP"), []apitxn.ProposalProcessor{d.BDDContext.Channel.PrimaryPeer()},
		eventHub); err != nil {
		return err
	}

	return nil
}

func (d *CommonSteps) queryCC(ccID string, channelID string, args string) error {

	// Get Query value
	argsArray := strings.Split(args, ",")

	if argsArray[0] == "endorseAndCommitTransaction" || argsArray[0] == "endorseTransaction" {
		argsArray = d.createTransactionSnapRequest(argsArray[0], argsArray[2], argsArray[1], argsArray[3:], true)
	}
	if argsArray[0] == "verifyTransactionProposalSignature" {
		signedProposalBytes, err := proto.Marshal(trxPR[0].Proposal.SignedProposal)
		if err != nil {
			return fmt.Errorf("Marshal SignedProposal return error: %v", err)
		}
		argsArray[2] = string(signedProposalBytes)
	}
	if argsArray[0] == "commitTransaction" {
		argsArray[2] = queryResult
	}
	if channelID != "" && d.BDDContext.Channel.Name() != channelID {
		return fmt.Errorf("Channel(%s) not created", channelID)
	}

	var err error
	if channelID != "" {
		queryResult, err = d.queryChaincode(d.BDDContext.Client, d.BDDContext.Channel, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())
	} else {
		queryResult, err = d.queryChaincode(d.BDDContext.Client, nil, ccID, argsArray, d.BDDContext.Channel.PrimaryPeer())
	}
	if err != nil {
		return fmt.Errorf("QueryChaincode return error: %v", err)
	}
	queryValue = queryResult
	if argsArray[0] == "endorseTransaction" {
		err := json.Unmarshal([]byte(queryResult), &trxPR)
		if err != nil {
			return fmt.Errorf("Unmarshal(%s) to TransactionProposalResponse return error: %v", queryValue, err)
		}
		queryValue = string(trxPR[0].ProposalResponse.GetResponse().Payload)
	}

	logger.Debugf("QueryChaincode return value: %s", queryValue)

	return nil
}

func (d *CommonSteps) invokeCC(ccID string, args string) error {

	// Get Query value
	argsArray := strings.Split(args, ",")

	return d.invokeCCWithArgs(ccID, argsArray)

}

func (d *CommonSteps) invokeCCWithArgs(ccID string, args []string) error {
	eventHub, err := d.getEventHub()
	if err != nil {
		return fmt.Errorf("getEventHub return error: %v", err)
	}

	_, err = sdkFabricTxn.InvokeChaincode(d.BDDContext.Client, d.BDDContext.Channel, []apitxn.ProposalProcessor{d.BDDContext.Channel.PrimaryPeer()},
		eventHub, ccID, args[0], args[1:], nil)
	if err != nil {
		return fmt.Errorf("InvokeChaincode return error: %v", err)
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

func (d *CommonSteps) containsInQueryValue(ccID string, value string) error {
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	if !strings.Contains(queryValue, value) {
		return fmt.Errorf("Query value(%s) doesn't contain expected value(%s)", queryValue, value)
	}
	return nil
}

// createAndSendTransactionProposal ...
func (d *CommonSteps) createAndSendTransactionProposal(channel sdkApi.Channel, chainCodeID string,
	args []string, targets []apitxn.ProposalProcessor, transientData map[string][]byte) ([]*apitxn.TransactionProposalResponse, string, error) {

	request := apitxn.ChaincodeInvokeRequest{
		Targets:      targets,
		Fcn:          args[0],
		Args:         args[1:],
		TransientMap: transientData,
		ChaincodeID:  chainCodeID,
	}
	var transactionProposalResponses []*apitxn.TransactionProposalResponse
	var txnID apitxn.TransactionID
	var err error
	if channel == nil {
		transactionProposalResponses, txnID, err = sdkFabricClientChannel.SendTransactionProposalWithChannelID("", request, d.BDDContext.Client)
	} else {
		transactionProposalResponses, txnID, err = channel.SendTransactionProposal(request)
	}
	if err != nil {
		return nil, txnID.ID, err
	}

	for _, v := range transactionProposalResponses {
		if v.Err != nil {
			return nil, txnID.ID, fmt.Errorf("invoke Endorser %s returned error: %v", v.Endorser, v.Err)
		}
		if v.ProposalResponse.Response.Status != 200 {
			return nil, txnID.ID, fmt.Errorf("invoke Endorser %s returned status: %v", v.Endorser, v.ProposalResponse.Response.Status)
		}
	}

	return transactionProposalResponses, txnID.ID, nil
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
func (d *CommonSteps) queryChaincode(client sdkApi.FabricClient, channel sdkApi.Channel, chaincodeID string,
	args []string, primaryPeer sdkApi.Peer) (string, error) {
	transactionProposalResponses, _, err := d.createAndSendTransactionProposal(channel,
		chaincodeID, args, []apitxn.ProposalProcessor{primaryPeer}, nil)

	if err != nil {
		return "", fmt.Errorf("CreateAndSendTransactionProposal returned error: %v", err)
	}

	return string(transactionProposalResponses[0].ProposalResponse.GetResponse().Payload), nil
}

func (d *CommonSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.beforeScenario)
	s.AfterScenario(d.BDDContext.afterScenario)
	s.Step(`^fabric has channel "([^"]*)" and p0 joined channel$`, d.createChannelAndPeerJoinChannel)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" version "([^"]*)" from path "([^"]*)" is installed and instantiated with args "([^"]*)"$`, d.installAndInstantiateCC)
	s.Step(`^client C1 query chaincode "([^"]*)" on channel "([^"]*)" with args "([^"]*)" on p0$`, d.queryCC)
	s.Step(`^C1 receive value "([^"]*)" from "([^"]*)"$`, d.checkQueryValue)
	s.Step(`^response from "([^"]*)" to client C1 contains value "([^"]*)"$`, d.containsInQueryValue)
	s.Step(`^client C1 invoke chaincode "([^"]*)" with args "([^"]*)" on p0$`, d.invokeCC)
}
