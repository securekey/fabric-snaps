/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"encoding/json"
	"fmt"
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
	sdkorderer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	fabricCommon "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
	configmanagerApi "github.com/securekey/fabric-snaps/configmanager/api"
)

// CommonSteps contain BDDContext
type CommonSteps struct {
	BDDContext *BDDContext
}

var logger = logging.NewLogger("test-logger")

var trxPR []*apitxn.TransactionProposalResponse
var queryValue string
var queryResult string

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

//queryChaincode ...
func (d *CommonSteps) queryChaincode(chaincodeID string,
	args []string, primaryPeer sdkApi.Peer) (string, error) {
	transactionProposalResponses, _, err := d.createAndSendTransactionProposal(
		chaincodeID, args, []apitxn.ProposalProcessor{primaryPeer}, nil)

	if err != nil {
		return "", fmt.Errorf("CreateAndSendTransactionProposal returned error: %v", err)
	}

	return string(transactionProposalResponses[0].ProposalResponse.GetResponse().Payload), nil
}

func (d *CommonSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.beforeScenario)
	s.AfterScenario(d.BDDContext.afterScenario)

	s.Step(`^the channel "([^"]*)" is created and all peers have joined$`, d.createChannelAndJoinAllPeers)
	s.Step(`^the channel "([^"]*)" is created and all peers from org "([^"]*)" have joined$`, d.createChannelAndJoinPeersFromOrg)
	s.Step(`^client invokes configuration snap on channel "([^"]*)" to load "([^"]*)" configuration on all peers$`, d.loadConfig)
	s.Step(`^we wait (\d+) seconds$`, d.wait)
	s.Step(`^client queries chaincode "([^"]*)" with args "([^"]*)" on all peers in the "([^"]*)" org on the "([^"]*)" channel$`, d.queryCConOrg)
	s.Step(`^client queries system chaincode "([^"]*)" with args "([^"]*)"$`, d.querySystemCC)

	s.Step(`^response from "([^"]*)" to client contains value "([^"]*)"$`, d.containsInQueryValue)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is installed from path "([^"]*)" to all peers$`, d.installChaincodeToAllPeers)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is installed from path "([^"]*)" to all peers in the "([^"]*)" org$`, d.installChaincodeToOrg)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is instantiated from path "([^"]*)" on all peers in the "([^"]*)" org on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.instantiateChaincodeOnOrg)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is instantiated from path "([^"]*)" on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.instantiateChaincode)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is deployed from path "([^"]*)" to all peers in the "([^"]*)" org on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.deployChaincodeToOrg)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is deployed from path "([^"]*)" to all peers on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.deployChaincode)
	s.Step(`^chaincode "([^"]*)" is warmed up on all peers in the "([^"]*)" org on the "([^"]*)" channel$`, d.warmUpCConOrg)
	s.Step(`^chaincode "([^"]*)" is warmed up on all peers on the "([^"]*)" channel$`, d.warmUpCC)

	s.Step(`^client invokes chaincode "([^"]*)" with args "([^"]*)" on all peers in the "([^"]*)" org on the "([^"]*)" channel$`, d.invokeCConOrg)

}

func (d *CommonSteps) wait(seconds int) error {
	logger.Infof("Waiting [%d] seconds\n", seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (d *CommonSteps) createChannelAndJoinAllPeers(channelID string) error {
	logger.Infof("Creating channel [%s] and joining all peers from orgs [%v]\n", channelID, d.BDDContext.Orgs)
	return d.createChannelAndJoinPeers(channelID, d.BDDContext.Orgs())
}

func (d *CommonSteps) createChannelAndJoinPeersFromOrg(channelID, orgs string) error {
	logger.Infof("Creating channel [%s] and joining all peers from orgs [%s]\n", channelID, orgs)
	orgList := strings.Split(orgs, ",")
	if len(orgList) == 0 {
		return fmt.Errorf("must specify at least one org ID")
	}
	return d.createChannelAndJoinPeers(channelID, orgList)
}

func (d *CommonSteps) createChannelAndJoinPeers(channelID string, orgs []string) error {
	if len(orgs) == 0 {
		return fmt.Errorf("no orgs specified")
	}
	// Create Orderer
	ordererConfig, err := d.BDDContext.ClientConfig.OrdererConfig("orderer.example.com")
	if err != nil {
		return fmt.Errorf("Could not load orderer config: %v", err)
	}
	orderer, err := sdkorderer.New(d.BDDContext.ClientConfig, sdkorderer.FromOrdererConfig(ordererConfig))
	if err != nil {
		return fmt.Errorf("New orderer failed: %v", err)
	}

	for index, orgID := range orgs {
		peersConfig, err := d.BDDContext.ClientConfig.PeersConfig(orgID)
		if err != nil {
			return fmt.Errorf("error getting peers config: %s", err)
		}
		if len(peersConfig) == 0 {
			return fmt.Errorf("no peers for org [%s]", orgID)
		}
		for pindex, peerConfig := range peersConfig {
			if err := d.joinPeerToChannel(orderer, channelID, orgID, peerConfig, index == 0, pindex == 0); err != nil {
				return fmt.Errorf("error joining peer to channel: %s", err)
			}
		}
	}

	return nil
}

func (d *CommonSteps) joinPeerToChannel(orderer *sdkorderer.Orderer, channelID, orgID string, peerConfig apiconfig.PeerConfig, createChannel, updateAnchorPeers bool) error {
	serverHostOverride := ""
	if str, ok := peerConfig.GRPCOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}
	peer, err := d.BDDContext.Sdk.FabricProvider().NewPeerFromConfig(&apiconfig.NetworkPeer{PeerConfig: peerConfig})
	if err != nil {
		return errors.WithMessage(err, "NewPeer failed")
	}

	orgClient := d.BDDContext.ClientForOrg(orgID)
	if orgClient == nil {
		return fmt.Errorf("org client for org: %s not found in BDD context", orgID)
	}
	//TODO temp solution to get peer id and msp id
	peerMspID := make(map[string]string)
	peerMspID["peer0.org1.example.com"] = "Org1MSP"

	d.BDDContext.AddPeerConfigToChannel(&PeerConfig{Config: peerConfig, OrgID: orgID, MspID: peerMspID[serverHostOverride], PeerID: serverHostOverride}, channelID)

	//Get Channel
	channel, err := d.BDDContext.resourceClients[d.BDDContext.Org1Admin].NewChannel(channelID)
	if err != nil {
		return fmt.Errorf("Create channel (%s) failed: %v", channelID, err)
	}

	if updateAnchorPeers {
		channel.AddOrderer(orderer)
	}
	channel.AddPeer(peer)
	d.BDDContext.primaryPeer = peer

	// Check if primary peer has joined channel
	alreadyJoined, err := HasPrimaryPeerJoinedChannel(d.BDDContext.resourceClients[d.BDDContext.Org1Admin], d.BDDContext.Org1Admin, channel)
	if err != nil {
		return fmt.Errorf("Error while checking if primary peer has already joined channel: %v", err)
	} else if alreadyJoined {
		return nil
	}

	// Channel management client is responsible for managing channels (create/update)
	chMgmtClient, err := d.BDDContext.clients[d.BDDContext.OrdererAdmin].ChannelMgmt()
	if err != nil {
		return fmt.Errorf("Failed to create new channel management client: %s", err)
	}

	if createChannel && channel.PrimaryPeer() == peer {
		// only the first peer of the first org can create a channel
		logger.Infof("Creating channel [%s]\n", channelID)
		txPath := GetChannelTxPath(channelID)
		if txPath == "" {
			return fmt.Errorf("channel TX path not found for channel: %s", channelID)
		}

		// Create and join channel
		req := chmgmt.SaveChannelRequest{ChannelID: channelID,
			ChannelConfig:   txPath,
			SigningIdentity: d.BDDContext.Org1Admin}

		if err = chMgmtClient.SaveChannel(req); err != nil {
			return errors.WithMessage(err, "SaveChannel failed")
		}

		// Sleep a while to avoid the SERVICE_UNAVAILABLE error that occurs after a new channel
		// has been created but is not ready yet when you attempt to join peers to it.
		logger.Infof("Waiting 30 seconds for orderers to sync ...\n")
		time.Sleep(time.Second * 30)
	}

	if updateAnchorPeers {
		logger.Infof("Updating anchor peers for org [%s] on channel [%s]\n", orgID, channelID)

		// Update anchors for peer org
		anchorTxPath := GetChannelAnchorTxPath(channelID, orgID)
		if anchorTxPath == "" {
			return fmt.Errorf("anchor TX path not found for channel [%s] and org [%s]", channelID, orgID)
		}
		// Create channel (or update if it already exists)
		req := chmgmt.SaveChannelRequest{ChannelID: channelID,
			ChannelConfig:   anchorTxPath,
			SigningIdentity: d.BDDContext.Org1Admin}

		if err = chMgmtClient.SaveChannel(req); err != nil {
			return errors.WithMessage(err, "SaveChannel failed")
		}
	}

	// Join Channel without error for anchor peers only. ignore JoinChannel error for other peers as AnchorePeer with JoinChannel will add all org's peers
	resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
	if err != nil {
		return fmt.Errorf("Failed to create new resource management client: %s", err)
	}
	if err = resMgmtClient.JoinChannel(channelID); err != nil {
		return fmt.Errorf("JoinChannel returned error: %v", err)
	}

	return nil
}

func (d *CommonSteps) loadConfig(channelID string, snaps string) error {
	logger.Infof("Loading snap config for channel [%s]...\n", channelID)

	snapsArray := strings.Split(snaps, ",")
	peersConfig := d.BDDContext.PeersByChannel(channelID)
	if len(peersConfig) == 0 {
		return fmt.Errorf("no peers are joined to channel [%s]", channelID)
	}

	for _, peerConfig := range peersConfig {
		logger.Infof("Loading config for peer [%s] on channel [%s]..\n", peerConfig.PeerID, channelID)

		pConfig := &configmanagerApi.PeerConfig{
			PeerID: peerConfig.PeerID,
		}

		for _, snap := range snapsArray {
			configData, err := ioutil.ReadFile(fmt.Sprintf("./fixtures/config/snaps/%s/config.yaml", snap))
			if err != nil {
				return fmt.Errorf("file error: %v", err)
			}
			pConfig.App = append(pConfig.App, configmanagerApi.AppConfig{AppName: snap, Config: string(configData)})
		}

		config := configmanagerApi.ConfigMessage{
			MspID: peerConfig.MspID,
			Peers: []configmanagerApi.PeerConfig{*pConfig},
		}

		configBytes, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("cannot Marshal %s", err)
		}

		var argsArray []string
		argsArray = append(argsArray, "save")
		argsArray = append(argsArray, string(configBytes))
		err = d.invokeCCWithArgs("configurationsnap", channelID, argsArray, nil, []*PeerConfig{peerConfig}...)

		if err != nil {
			return fmt.Errorf("invokeChaincode return error: %v", err)
		}

	}
	return nil
}

func (d *CommonSteps) invokeCConOrg(ccID, args, orgIDs, channelID string) error {
	err := d.invokeCCWithArgs(ccID, channelID, strings.Split(args, ","), nil, d.orgPeers(orgIDs, channelID)...)
	if err != nil {
		return fmt.Errorf("invokeCCWithArgs return error: %v", err)
	}
	return nil
}

// invokeCCWithArgs ...
func (d *CommonSteps) invokeCCWithArgs(ccID, channelID string, args []string, transientData map[string][]byte, targets ...*PeerConfig) error {
	if len(targets) == 0 {
		return fmt.Errorf("no target peer specified")
	}

	//	logger.Infof("Invoking chaincode [%s] with args [%v] on channel [%s]\n", ccID, args, channelID)

	var prosalProcessors []apitxn.ProposalProcessor

	for _, target := range targets {

		targetPeer, err := d.BDDContext.Sdk.FabricProvider().NewPeerFromConfig(&apiconfig.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return errors.WithMessage(err, "NewPeer failed")
		}
		prosalProcessors = append(prosalProcessors, targetPeer)
	}

	chClient, err := d.BDDContext.clients[d.BDDContext.Org1User].Channel(channelID)
	if err != nil {
		return fmt.Errorf("Failed to create new channel client: %s", err)
	}
	defer chClient.Close()

	_, _, err = chClient.ExecuteTxWithOpts(apitxn.ExecuteTxRequest{ChaincodeID: ccID, Fcn: args[0], Args: GetByteArgs(args[1:])},
		apitxn.ExecuteTxOpts{ProposalProcessors: prosalProcessors})
	if err != nil {
		return fmt.Errorf("InvokeChaincode return error: %v", err)
	}
	return err
}

// createAndSendTransactionProposal ...
func (d *CommonSteps) createAndSendTransactionProposal(chainCodeID string,
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
	transactionProposalResponses, txnID, err := sdkFabricClientChannel.SendTransactionProposalWithChannelID("", request, d.BDDContext.resourceClients[d.BDDContext.Org1Admin])
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

//TODO
func (d *CommonSteps) querySystemCC(ccID, args string) error {

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
	var err error
	queryResult, err = d.queryChaincode(ccID, argsArray, d.BDDContext.primaryPeer)
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

func (d *CommonSteps) queryCConOrg(ccID, args, orgIDs, channelID string) error {
	queryResult, err := d.queryCCWithArgs(ccID, channelID, strings.Split(args, ","), d.orgPeers(orgIDs, channelID)...)
	if err != nil {
		return fmt.Errorf("queryCCWithArgs return error: %v", err)
	}
	queryValue = queryResult
	logger.Debugf("queryCCWithArgs return value: %s", queryValue)
	return nil
}

// queryCCWithArgs ...
func (d *CommonSteps) queryCCWithArgs(ccID, channelID string, args []string, targets ...*PeerConfig) (string, error) {
	return d.queryCCWithOpts(ccID, channelID, args, 0, true, 0, targets...)
}

func (d *CommonSteps) queryCCWithOpts(ccID, channelID string, args []string, timeout time.Duration, concurrent bool, interval time.Duration, targets ...*PeerConfig) (string, error) {
	if len(targets) == 0 {
		logger.Errorf("No target specified\n")
		return "", errors.New("no targets specified")
	}

	var processors []apitxn.ProposalProcessor

	for _, target := range targets {

		targetPeer, err := d.BDDContext.Sdk.FabricProvider().NewPeerFromConfig(&apiconfig.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return "", errors.WithMessage(err, "NewPeer failed")
		}

		processors = append(processors, targetPeer)
	}

	client := d.BDDContext.clients[d.BDDContext.Org1Admin]
	chClient, err := client.Channel(channelID)
	if err != nil {
		logger.Errorf("Failed to create new channel client: %s\n", err)
		return "", errors.Wrap(err, "Failed to create new channel client")
	}
	defer chClient.Close()

	if concurrent {
		opts := apitxn.QueryOpts{
			ProposalProcessors: processors,
		}
		if timeout > 0 {
			opts.Timeout = timeout
		}

		result, err := chClient.QueryWithOpts(
			apitxn.QueryRequest{
				ChaincodeID: ccID,
				Fcn:         args[0],
				Args:        GetByteArgs(args[1:]),
			}, opts,
		)
		if err != nil {
			return "", fmt.Errorf("QueryChaincode return error: %v", err)
		}
		queryResult = string(result)
	} else {
		var errs []error
		for _, processor := range processors {
			opts := apitxn.QueryOpts{
				ProposalProcessors: []apitxn.ProposalProcessor{processor},
			}
			if timeout > 0 {
				opts.Timeout = timeout
			}

			logger.Infof("Querying CC [%s] on peer [%s]\n", ccID, processorsAsString(processor))
			result, err := chClient.QueryWithOpts(
				apitxn.QueryRequest{
					ChaincodeID: ccID,
					Fcn:         args[0],
					Args:        GetByteArgs(args[1:]),
				}, opts,
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				queryResult = string(result)
			}
			if interval > 0 {
				logger.Infof("Waiting %s\n", interval)
				time.Sleep(interval)
			}
		}
		if len(errs) > 0 {
			return "", fmt.Errorf("QueryChaincode return error: %v", errs[0])
		}
	}

	logger.Debugf("QueryChaincode return value: %s", queryResult)
	return queryResult, nil
}

func (d *CommonSteps) containsInQueryValue(ccID string, value string) error {
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	logger.Infof("Query value %s and tested value %s", queryValue, value)
	if !strings.Contains(queryValue, value) {
		return fmt.Errorf("Query value(%s) doesn't contain expected value(%s)", queryValue, value)
	}
	return nil
}

func (d *CommonSteps) installChaincodeToAllPeers(ccType, ccID, ccPath string) error {
	logger.Infof("Installing chaincode [%s] from path [%s] to all peers\n", ccID, ccPath, "")
	return d.installChaincodeToOrg(ccType, ccID, ccPath, "")
}

func (d *CommonSteps) instantiateChaincode(ccType, ccID, ccPath, channelID, args, ccPolicy, collectionNames string) error {
	logger.Infof("Preparing to instantiate chaincode [%s] from path [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]\n", ccID, ccPath, channelID, args, ccPolicy, collectionNames)
	return d.instantiateChaincodeWithOpts(ccType, ccID, ccPath, "", channelID, args, ccPolicy, collectionNames, false)
}

func (d *CommonSteps) instantiateChaincodeOnOrg(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames string) error {
	logger.Infof("Preparing to instantiate chaincode [%s] from path [%s] to orgs [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]\n", ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames)
	return d.instantiateChaincodeWithOpts(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames, false)
}

func (d *CommonSteps) deployChaincode(ccType, ccID, ccPath, channelID, args, ccPolicy, collectionPolicy string) error {
	logger.Infof("Installing and instantiating chaincode [%s] from path [%s] to channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]\n", ccID, ccPath, channelID, args, ccPolicy, collectionPolicy)
	return d.deployChaincodeToOrg(ccType, ccID, ccPath, "", channelID, args, ccPolicy, collectionPolicy)
}

func (d *CommonSteps) installChaincodeToOrg(ccType, ccID, ccPath, orgIDs string) error {
	logger.Infof("Preparing to install chaincode [%s] from path [%s] to orgs [%s]\n", ccID, ccPath, orgIDs)

	var oIDs []string
	if orgIDs != "" {
		oIDs = strings.Split(orgIDs, ",")
	} else {
		oIDs = d.BDDContext.orgs
	}

	for _, orgID := range oIDs {

		resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
		if err != nil {
			return fmt.Errorf("Failed to create new resource management client: %s", err)
		}

		ccPkg, err := packager.NewCCPackage(ccPath, d.getDeployPath(ccType))
		if err != nil {
			return err
		}

		logger.Infof("... installing chaincode [%s] from path [%s] to org [%s]\n", ccID, ccPath, orgID)
		_, err = resMgmtClient.InstallCC(resmgmt.InstallCCRequest{Name: ccID, Path: ccPath, Version: "v1", Package: ccPkg})
		if err != nil {
			return fmt.Errorf("SendInstallProposal return error: %v", err)
		}
	}
	return nil
}

func (d *CommonSteps) instantiateChaincodeWithOpts(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames string, allPeers bool) error {
	logger.Infof("Preparing to instantiate chaincode [%s] from path [%s] to orgs [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]\n", ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames)

	peers := d.orgPeers(orgIDs, channelID)
	if len(peers) == 0 {
		return errors.Errorf("no peers found for orgs [%s]", orgIDs)
	}
	chaincodePolicy, err := d.newChaincodePolicy(ccPolicy, channelID)
	if err != nil {
		return fmt.Errorf("error creating endirsement policy: %s", err)
	}

	var sdkPeers []sdkApi.Peer

	for _, pconfig := range peers {

		sdkPeer, err := d.BDDContext.Sdk.FabricProvider().NewPeerFromConfig(&apiconfig.NetworkPeer{PeerConfig: pconfig.Config})
		if err != nil {
			return errors.WithMessage(err, "NewPeer failed")
		}

		sdkPeers = append(sdkPeers, sdkPeer)
		if !allPeers {
			break
		}
	}

	var collConfig []*common.CollectionConfig
	if collectionNames != "" {
		// Define the private data collection policy config
		for _, collName := range strings.Split(collectionNames, ",") {
			logger.Infof("Configuring collection (%s) for CCID=%s", collName, ccID)
			config := d.BDDContext.CollectionConfig(collName)
			if config == nil {
				return errors.Errorf("no collection config defined for collection [%s]", collName)
			}
			policyEnv, err := d.newChaincodePolicy(config.Policy, channelID)
			if err != nil {
				return errors.Wrapf(err, "error creating collection policy for collection [%s]", collName)
			}
			collConfig = append(collConfig, NewCollectionConfig(config.Name, config.RequiredPeerCount, config.MaxPeerCount, policyEnv))
		}
	}

	resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
	if err != nil {
		return fmt.Errorf("Failed to create new resource management client: %s", err)
	}

	logger.Infof("Instantiating chaincode [%s] from path [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s] to the following peers: [%s]\n", ccID, ccPath, channelID, args, ccPolicy, collectionNames, peersAsString(sdkPeers))

	return resMgmtClient.InstantiateCCWithOpts(
		channelID,
		resmgmt.InstantiateCCRequest{
			Name:       ccID,
			Path:       ccPath,
			Version:    "v1",
			Args:       GetByteArgs(strings.Split(args, ",")),
			Policy:     chaincodePolicy,
			CollConfig: collConfig,
		},
		resmgmt.InstantiateCCOpts{
			Targets: sdkPeers,
			Timeout: 5 * time.Minute,
		},
	)
}

func (d *CommonSteps) deployChaincodeToOrg(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames string) error {
	logger.Infof("Installing and instantiating chaincode [%s] from path [%s] to orgs [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]\n", ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames)

	peers := d.orgPeers(orgIDs, channelID)
	if len(peers) == 0 {
		return errors.Errorf("no peers found for orgs [%s]", orgIDs)
	}
	chaincodePolicy, err := d.newChaincodePolicy(ccPolicy, channelID)
	if err != nil {
		return fmt.Errorf("error creating endirsement policy: %s", err)
	}

	var sdkPeers []sdkApi.Peer
	var isInstalled bool
	var orgID string

	for _, pconfig := range peers {
		orgID = pconfig.OrgID
		orgClient := d.BDDContext.ClientForOrg(orgID)
		if orgClient == nil {
			return fmt.Errorf("Org client for org: %s not found in BDD context", orgID)
		}

		chClient, err := orgClient.Channel(channelID)
		if err != nil {
			return errors.Wrap(err, "Failed to create new channel client")
		}
		defer chClient.Close()

		sdkPeer, err := d.BDDContext.Sdk.FabricProvider().NewPeerFromConfig(&apiconfig.NetworkPeer{PeerConfig: pconfig.Config})
		if err != nil {
			return errors.WithMessage(err, "NewPeer failed")
		}

		isInstalled, err = IsChaincodeInstalled(d.BDDContext.resourceClients[d.BDDContext.Org1Admin], sdkPeer, ccID)
		if err != nil {
			return fmt.Errorf("Error querying installed chaincodes: %s", err)
		}

		if !isInstalled {

			resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
			if err != nil {
				return fmt.Errorf("Failed to create new resource management client: %s", err)
			}

			ccPkg, err := packager.NewCCPackage(ccPath, d.getDeployPath(ccType))
			if err != nil {
				return err
			}

			installRqst := resmgmt.InstallCCRequest{Name: ccID, Path: ccPath, Version: "v1", Package: ccPkg}
			_, err = resMgmtClient.InstallCC(installRqst)
			if err != nil {
				return fmt.Errorf("SendInstallProposal return error: %v", err)
			}
		}

		sdkPeers = append(sdkPeers, sdkPeer)
	}

	argsArray := strings.Split(args, ",")

	var collConfig []*common.CollectionConfig
	if collectionNames != "" {
		// Define the private data collection policy config
		for _, collName := range strings.Split(collectionNames, ",") {
			logger.Infof("Configuring collection (%s) for CCID=%s", collName, ccID)
			config := d.BDDContext.CollectionConfig(collName)
			if config == nil {
				return errors.Errorf("no collection config defined for collection [%s]", collName)
			}
			policyEnv, err := d.newChaincodePolicy(config.Policy, channelID)
			if err != nil {
				return errors.Wrapf(err, "error creating collection policy for collection [%s]", collName)
			}
			collConfig = append(collConfig, NewCollectionConfig(config.Name, config.RequiredPeerCount, config.MaxPeerCount, policyEnv))
		}
	}

	resMgmtClient, err := d.BDDContext.clients[d.BDDContext.Org1Admin].ResourceMgmt()
	if err != nil {
		return fmt.Errorf("Failed to create new resource management client: %s", err)
	}

	instantiateRqst := resmgmt.InstantiateCCRequest{Name: ccID, Path: ccPath, Version: "v1", Args: GetByteArgs(argsArray), Policy: chaincodePolicy,
		CollConfig: collConfig}

	return resMgmtClient.InstantiateCCWithOpts(channelID, instantiateRqst, resmgmt.InstantiateCCOpts{Targets: sdkPeers, Timeout: 5 * time.Minute})
}

func (d *CommonSteps) newChaincodePolicy(ccPolicy, channelID string) (*fabricCommon.SignaturePolicyEnvelope, error) {
	if ccPolicy != "" {
		// Create a signature policy from the policy expression passed in
		return newPolicy(ccPolicy)
	}

	// Default policy is 'signed by any member' for all known orgs
	var mspIDs []string
	for _, orgID := range d.BDDContext.OrgsByChannel(channelID) {
		mspID, err := d.BDDContext.ClientConfig.MspID(orgID)
		if err != nil {
			return nil, errors.Errorf("Unable to get the MSP ID from org ID %s: %s", orgID, err)
		}
		mspIDs = append(mspIDs, mspID)
	}
	logger.Infof("Returning SignedByAnyMember policy for MSPs %v\n", mspIDs)
	return cauthdsl.SignedByAnyMember(mspIDs), nil
}

func (d *CommonSteps) orgPeers(orgIDs, channelID string) []*PeerConfig {
	var orgMap map[string]bool
	if orgIDs != "" {
		orgMap = make(map[string]bool)
		for _, orgID := range strings.Split(orgIDs, ",") {
			orgMap[orgID] = true
		}
	}
	var peers []*PeerConfig
	for _, pconfig := range d.BDDContext.PeersByChannel(channelID) {
		if orgMap == nil || orgMap[pconfig.OrgID] {
			peers = append(peers, pconfig)
		}
	}
	return peers
}

func (d *CommonSteps) warmUpCC(ccID, channelID string) error {
	logger.Infof("Warming up chaincode [%s] on channel [%s]\n", ccID, channelID)
	return d.warmUpCConOrg(ccID, "", channelID)
}

func (d *CommonSteps) warmUpCConOrg(ccID, orgIDs, channelID string) error {
	logger.Infof("Warming up chaincode [%s] on orgs [%s] and channel [%s]\n", ccID, orgIDs, channelID)
	for {
		_, err := d.queryCCWithOpts(ccID, channelID, []string{"whatever"}, 5*time.Minute, false, 0, d.orgPeers(orgIDs, channelID)...)
		if err != nil && strings.Contains(err.Error(), "premature execution - chaincode") {
			// Wait until we can successfully invoke the chaincode
			logger.Infof("Error warming up chaincode [%s]: %s. Retrying in 5 seconds...", ccID, err)
			time.Sleep(5 * time.Second)
		} else {
			// Don't worry about any other type of error
			return nil
		}
	}
}

func processorsAsString(processors ...apitxn.ProposalProcessor) string {
	str := ""
	for _, p := range processors {
		str += p.(sdkApi.Peer).URL() + " "
	}
	return str
}

func peersAsString(peers []sdkApi.Peer) string {
	str := ""
	for _, p := range peers {
		str += p.URL() + " "
	}
	return str
}

func newPolicy(policyString string) (*fabricCommon.SignaturePolicyEnvelope, error) {
	ccPolicy, err := FromString(policyString)
	if err != nil {
		return nil, errors.Errorf("invalid chaincode policy [%s]: %s", policyString, err)
	}
	return ccPolicy, nil
}
