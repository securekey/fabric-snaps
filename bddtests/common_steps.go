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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	mspApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	contextImpl "github.com/hyperledger/fabric-sdk-go/pkg/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	fabricCommon "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/pkg/errors"
)

// CommonSteps contain BDDContext
type CommonSteps struct {
	BDDContext *BDDContext
}

var logger = logging.NewLogger("test-logger")

var queryValue string

type queryInfoResponse struct {
	Height            string
	CurrentBlockHash  string
	PreviousBlockHash string
}

// NewCommonSteps create new CommonSteps struct
func NewCommonSteps(context *BDDContext) *CommonSteps {
	//grpclog.SetLogger(logger)
	return &CommonSteps{BDDContext: context}
}

// GetDeployPath ..
func (d *CommonSteps) getDeployPath(ccType string) string {
	// test cc come from fixtures
	pwd, _ := os.Getwd()

	return path.Join(pwd, d.BDDContext.testCCPath)
}

func (d *CommonSteps) displayBlockFromChannel(blockNum int, channelID string) error {
	block, err := d.getBlocks(channelID, blockNum, 1)
	if err != nil {
		return err
	}
	logger.Infof("%s\n", block)
	return nil
}

func (d *CommonSteps) getBlocks(channelID string, blockNum, numBlocks int) (string, error) {
	orgID, err := d.BDDContext.OrgIDForChannel(channelID)
	if err != nil {
		return "", err
	}

	strBlockNum := fmt.Sprintf("%d", blockNum)
	strNumBlocks := fmt.Sprintf("%d", numBlocks)
	return NewFabCLI().Exec("query", "block", "--config", d.BDDContext.clientConfigFilePath+d.BDDContext.clientConfigFileName, "--cid", channelID, "--orgid", orgID, "--num", strBlockNum, "--traverse", strNumBlocks)
}

func (d *CommonSteps) displayBlocksFromChannel(numBlocks int, channelID string) error {
	height, err := d.getChannelBlockHeight(channelID)
	if err != nil {
		return fmt.Errorf("error getting channel height: %s", err)
	}

	block, err := d.getBlocks(channelID, height-1, numBlocks)
	if err != nil {
		return err
	}

	logger.Infof("%s\n", block)

	return nil
}

func (d *CommonSteps) getChannelBlockHeight(channelID string) (int, error) {
	orgID, err := d.BDDContext.OrgIDForChannel(channelID)
	if err != nil {
		return 0, err
	}

	resp, err := NewFabCLI().GetJSON("query", "info", "--config", d.BDDContext.clientConfigFilePath+d.BDDContext.clientConfigFileName, "--cid", channelID, "--orgid", orgID)
	if err != nil {
		return 0, err
	}

	var info queryInfoResponse
	if err := json.Unmarshal([]byte(resp), &info); err != nil {
		return 0, fmt.Errorf("Error unmarshalling JSON response: %s", err)
	}

	return strconv.Atoi(info.Height)
}

func (d *CommonSteps) displayLastBlockFromChannel(channelID string) error {
	return d.displayBlocksFromChannel(1, channelID)
}

func (d *CommonSteps) wait(seconds int) error {
	logger.Infof("Waiting [%d] seconds\n", seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
	return nil
}

func (d *CommonSteps) createChannelAndJoinAllPeers(channelID string) error {
	return d.createChannelAndJoinPeers(channelID, d.BDDContext.Orgs())
}

func (d *CommonSteps) createChannelAndJoinPeersFromOrg(channelID, orgs string) error {
	orgList := strings.Split(orgs, ",")
	if len(orgList) == 0 {
		return fmt.Errorf("must specify at least one org ID")
	}
	return d.createChannelAndJoinPeers(channelID, orgList)
}

func (d *CommonSteps) createChannelAndJoinPeers(channelID string, orgs []string) error {
	logger.Infof("Creating channel [%s] and joining all peers from orgs %s", channelID, orgs)
	if len(orgs) == 0 {
		return fmt.Errorf("no orgs specified")
	}

	for _, orgID := range orgs {
		peersConfig, ok := d.BDDContext.clientConfig.PeersConfig(orgID)
		if !ok {
			return fmt.Errorf("could not get peers config for org [%s]", orgID)
		}
		if len(peersConfig) == 0 {
			return fmt.Errorf("no peers for org [%s]", orgID)
		}
		if err := d.joinPeersToChannel(channelID, orgID, peersConfig); err != nil {
			return fmt.Errorf("error joining peer to channel: %s", err)
		}

	}

	return nil
}

func (d *CommonSteps) joinPeersToChannel(channelID, orgID string, peersConfig []fabApi.PeerConfig) error {

	for _, peerConfig := range peersConfig {
		serverHostOverride := ""
		if str, ok := peerConfig.GRPCOptions["ssl-target-name-override"].(string); ok {
			serverHostOverride = str
		}
		d.BDDContext.AddPeerConfigToChannel(&PeerConfig{Config: peerConfig, OrgID: orgID, MspID: d.BDDContext.peersMspID[serverHostOverride], PeerID: serverHostOverride}, channelID)
	}
	peer, err := d.BDDContext.OrgUserContext(orgID, ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: peersConfig[0]})
	if err != nil {
		return errors.WithMessage(err, "NewPeer failed")
	}
	resourceMgmt := d.BDDContext.ResMgmtClient(orgID, ADMIN)

	// Check if primary peer has joined channel
	alreadyJoined, err := HasPrimaryPeerJoinedChannel(channelID, resourceMgmt, d.BDDContext.OrgUserContext(orgID, ADMIN), peer)
	if err != nil {
		return fmt.Errorf("Error while checking if primary peer has already joined channel: %s", err)
	} else if alreadyJoined {
		logger.Infof("alreadyJoined orgID [%s]\n", orgID)
		return nil
	}

	if d.BDDContext.ChannelCreated(channelID) == false {
		// only the first peer of the first org can create a channel
		logger.Infof("Creating channel [%s]\n", channelID)
		txPath := GetChannelTxPath(channelID)
		if txPath == "" {
			return fmt.Errorf("channel TX path not found for channel: %s", channelID)
		}

		// Create and join channel
		req := resmgmt.SaveChannelRequest{ChannelID: channelID,
			ChannelConfigPath: txPath,
			SigningIdentities: []mspApi.SigningIdentity{d.BDDContext.OrgUserContext(orgID, ADMIN)}}

		if _, err = resourceMgmt.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts)); err != nil {
			return errors.WithMessage(err, "SaveChannel failed")
		}
	}

	logger.Infof("Updating anchor peers for org [%s] on channel [%s]\n", orgID, channelID)

	// Update anchors for peer org
	anchorTxPath := GetChannelAnchorTxPath(channelID, orgID)
	if anchorTxPath == "" {
		return fmt.Errorf("anchor TX path not found for channel [%s] and org [%s]", channelID, orgID)
	}
	// Create channel (or update if it already exists)
	req := resmgmt.SaveChannelRequest{ChannelID: channelID,
		ChannelConfigPath: anchorTxPath,
		SigningIdentities: []mspApi.SigningIdentity{d.BDDContext.OrgUserContext(orgID, ADMIN)}}

	if _, err := resourceMgmt.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts)); err != nil {
		return errors.WithMessage(err, "SaveChannel failed")
	}

	d.BDDContext.createdChannels[channelID] = true

	// Join Channel without error for anchor peers only. ignore JoinChannel error for other peers as AnchorePeer with JoinChannel will add all org's peers

	resMgmtClient := d.BDDContext.ResMgmtClient(orgID, ADMIN)
	if err = resMgmtClient.JoinChannel(channelID, resmgmt.WithRetry(retry.DefaultResMgmtOpts)); err != nil {
		return fmt.Errorf("JoinChannel returned error: %s", err)
	}

	return nil
}

// InvokeCConOrg invoke cc on org
func (d *CommonSteps) InvokeCConOrg(ccID, args, orgIDs, channelID string) error {
	if _, err := d.InvokeCCWithArgs(ccID, channelID, d.OrgPeers(orgIDs, channelID), strings.Split(args, ","), nil); err != nil {
		return fmt.Errorf("InvokeCCWithArgs return error: %s", err)
	}
	return nil
}

//InvokeCCWithArgsAsAdmin invoke cc with args as admin user type
func (d *CommonSteps) InvokeCCWithArgsAsAdmin(ccID, channelID string, targets []*PeerConfig, args []string, transientData map[string][]byte) (channel.Response, error) {
	return d.invokeCCWithArgs(ccID, channelID, targets, args, transientData, ADMIN)
}

//InvokeCCWithArgs invoke cc with args as regular user
func (d *CommonSteps) InvokeCCWithArgs(ccID, channelID string, targets []*PeerConfig, args []string, transientData map[string][]byte) (channel.Response, error) {
	return d.invokeCCWithArgs(ccID, channelID, targets, args, transientData, USER)
}

// invokeCCWithArgs ...
func (d *CommonSteps) invokeCCWithArgs(ccID, channelID string, targets []*PeerConfig, args []string, transientData map[string][]byte, userType string) (channel.Response, error) {
	if len(targets) == 0 {
		return channel.Response{}, fmt.Errorf("no target peer specified")
	}

	//	logger.Infof("Invoking chaincode [%s] with args [%s] on channel [%s]\n", ccID, args, channelID)

	var peers []fabApi.Peer

	for _, target := range targets {

		targetPeer, err := d.BDDContext.OrgUserContext(targets[0].OrgID, ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return channel.Response{}, errors.WithMessage(err, "NewPeer failed")
		}
		peers = append(peers, targetPeer)
	}

	chClient, err := d.BDDContext.OrgChannelClient(targets[0].OrgID, userType, channelID)
	if err != nil {
		return channel.Response{}, fmt.Errorf("Failed to create new channel client: %s", err)
	}

	retryOpts := retry.DefaultOpts
	retryOpts.RetryableCodes = retry.ChannelClientRetryableCodes

	response, err := chClient.Execute(
		channel.Request{
			ChaincodeID: ccID,
			Fcn:         args[0],
			Args:        GetByteArgs(args[1:]),
		},
		channel.WithTargets(peers...),
		channel.WithRetry(retryOpts),
	)

	if err != nil {
		return channel.Response{}, fmt.Errorf("InvokeChaincode return error: %s", err)
	}
	return response, nil
}

func (d *CommonSteps) queryCConOrg(ccID, args, orgIDs, channelID string) error {
	queryValue = ""

	var err error
	queryValue, err = d.QueryCCWithArgs(false, ccID, channelID, strings.Split(args, ","), nil, d.OrgPeers(orgIDs, channelID)...)
	if err != nil {
		return fmt.Errorf("QueryCCWithArgs return error: %s", err)
	}
	logger.Debugf("QueryCCWithArgs return value: [%s]", queryValue)
	return nil
}

func (d *CommonSteps) querySystemCC(ccID, args, orgID, channelID string) error {
	queryValue = ""

	peersConfig, ok := d.BDDContext.clientConfig.PeersConfig(orgID)
	if !ok {
		return fmt.Errorf("could not get peers config for org [%s]", orgID)
	}

	serverHostOverride := ""
	if str, ok := peersConfig[0].GRPCOptions["ssl-target-name-override"].(string); ok {
		serverHostOverride = str
	}
	argsArray := strings.Split(args, ",")

	var err error
	queryValue, err = d.QueryCCWithArgs(true, ccID, channelID, argsArray, nil,
		[]*PeerConfig{&PeerConfig{Config: peersConfig[0], OrgID: orgID, MspID: d.BDDContext.peersMspID[serverHostOverride], PeerID: serverHostOverride}}...)
	if err != nil {
		return fmt.Errorf("QueryCCWithArgs return error: %s", err)
	}
	logger.Debugf("QueryCCWithArgs return value: [%s]", queryValue)
	return nil
}

// QueryCCWithArgs ...
func (d *CommonSteps) QueryCCWithArgs(systemCC bool, ccID, channelID string, args []string, transientData map[string][]byte, targets ...*PeerConfig) (string, error) {
	return d.QueryCCWithOpts(systemCC, ccID, channelID, args, 0, true, 0, transientData, targets...)
}

// QueryCCWithOpts ...
func (d *CommonSteps) QueryCCWithOpts(systemCC bool, ccID, channelID string, args []string, timeout time.Duration, concurrent bool, interval time.Duration, transientData map[string][]byte, targets ...*PeerConfig) (string, error) {
	if len(targets) == 0 {
		logger.Errorf("No target specified")
		return "", errors.New("no targets specified")
	}

	var peers []fabApi.Peer
	var orgID string
	var queryResult string
	for _, target := range targets {
		orgID = target.OrgID

		targetPeer, err := d.BDDContext.OrgUserContext(orgID, ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: target.Config})
		if err != nil {
			return "", errors.WithMessage(err, "NewPeer failed")
		}

		peers = append(peers, targetPeer)
	}

	chClient, err := d.BDDContext.OrgChannelClient(orgID, ADMIN, channelID)
	if err != nil {
		logger.Errorf("Failed to create new channel client: %s", err)
		return "", errors.Wrap(err, "Failed to create new channel client")
	}
	if systemCC {
		// Create a system channel client

		systemHandlerChain := invoke.NewProposalProcessorHandler(
			NewCustomEndorsementHandler(
				d.BDDContext.OrgUserContext(orgID, USER),
				invoke.NewEndorsementValidationHandler(),
			))

		resp, err := chClient.InvokeHandler(systemHandlerChain, channel.Request{
			ChaincodeID:  ccID,
			Fcn:          args[0],
			Args:         GetByteArgs(args[1:]),
			TransientMap: transientData,
		}, channel.WithTargets(peers...), channel.WithTimeout(fabApi.Execute, timeout))
		if err != nil {
			return "", fmt.Errorf("QueryChaincode return error: %s", err)
		}
		queryResult = string(resp.Payload)
		return queryResult, nil
	}

	if concurrent {

		resp, err := chClient.Query(channel.Request{
			ChaincodeID:  ccID,
			Fcn:          args[0],
			Args:         GetByteArgs(args[1:]),
			TransientMap: transientData,
		}, channel.WithTargets(peers...), channel.WithTimeout(fabApi.Execute, timeout))
		if err != nil {
			return "", fmt.Errorf("QueryChaincode return error: %s", err)
		}
		queryResult = string(resp.Payload)

	} else {
		var errs []error
		for _, peer := range peers {
			if len(args) > 0 && args[0] == "warmup" {
				logger.Infof("Warming up chaincode [%s] on peer [%s] in channel [%s]", ccID, peer.URL(), channelID)
			}
			resp, err := chClient.Query(channel.Request{
				ChaincodeID:  ccID,
				Fcn:          args[0],
				Args:         GetByteArgs(args[1:]),
				TransientMap: transientData,
			}, channel.WithTargets([]fabApi.Peer{peer}...), channel.WithTimeout(fabApi.Execute, timeout))
			if err != nil {
				errs = append(errs, err)
			} else {
				queryResult = string(resp.Payload)
			}
			if interval > 0 {
				logger.Infof("Waiting %s\n", interval)
				time.Sleep(interval)
			}
		}
		if len(errs) > 0 {
			return "", fmt.Errorf("QueryChaincode return error: %s", errs[0])
		}
	}

	logger.Debugf("QueryChaincode return value: [%s]", queryResult)
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

func (d *CommonSteps) equalQueryValue(ccID string, value string) error {
	if queryValue == "" {
		return fmt.Errorf("QueryValue is empty")
	}
	logger.Infof("Query value %s and tested value %s", queryValue, value)
	if queryValue != value {
		return fmt.Errorf("Query value(%s) doesn't equal expected value(%s)", queryValue, value)
	}
	return nil
}

func (d *CommonSteps) installChaincodeToAllPeers(ccType, ccID, ccPath string) error {
	logger.Infof("Installing chaincode [%s] from path [%s] to all peers", ccID, ccPath)
	return d.installChaincodeToOrg(ccType, ccID, ccPath, "", "")
}

func (d *CommonSteps) installChaincodeToAllPeersExcept(ccType, ccID, ccPath, blackListRegex string) error {
	logger.Infof("Installing chaincode [%s] from path [%s] to all peers except [%s]", ccID, ccPath, blackListRegex)
	return d.installChaincodeToOrg(ccType, ccID, ccPath, "", blackListRegex)
}

func (d *CommonSteps) instantiateChaincode(ccType, ccID, ccPath, channelID, args, ccPolicy, collectionNames string) error {
	logger.Infof("Preparing to instantiate chaincode [%s] from path [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]", ccID, ccPath, channelID, args, ccPolicy, collectionNames)
	return d.instantiateChaincodeWithOpts(ccType, ccID, ccPath, "", channelID, args, ccPolicy, collectionNames, false)
}

func (d *CommonSteps) instantiateChaincodeOnOrg(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames string) error {
	logger.Infof("Preparing to instantiate chaincode [%s] from path [%s] to orgs [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]", ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames)
	return d.instantiateChaincodeWithOpts(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames, false)
}

func (d *CommonSteps) deployChaincode(ccType, ccID, ccPath, channelID, args, ccPolicy, collectionPolicy string) error {
	logger.Infof("Installing and instantiating chaincode [%s] from path [%s] to channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]", ccID, ccPath, channelID, args, ccPolicy, collectionPolicy)
	return d.deployChaincodeToOrg(ccType, ccID, ccPath, "", channelID, args, ccPolicy, collectionPolicy)
}

func (d *CommonSteps) installChaincodeToOrg(ccType, ccID, ccPath, orgIDs, blackListRegex string) error {
	logger.Infof("Preparing to install chaincode [%s] from path [%s] to orgs [%s] - Blacklisted peers: [%s]", ccID, ccPath, orgIDs, blackListRegex)

	var oIDs []string
	if orgIDs != "" {
		oIDs = strings.Split(orgIDs, ",")
	} else {
		oIDs = d.BDDContext.orgs
	}

	for _, orgID := range oIDs {
		targets, err := d.getLocalTargets(orgID, blackListRegex)
		if err != nil {
			return err
		}

		resMgmtClient := d.BDDContext.ResMgmtClient(orgID, ADMIN)

		ccPkg, err := gopackager.NewCCPackage(ccPath, d.getDeployPath(ccType))
		if err != nil {
			return err
		}

		if len(targets) == 0 {
			return errors.Errorf("no targets for chaincode [%s]", ccID)
		}

		logger.Infof("... installing chaincode [%s] from path [%s] to targets %s", ccID, ccPath, targets)
		_, err = resMgmtClient.InstallCC(
			resmgmt.InstallCCRequest{Name: ccID, Path: ccPath, Version: "v1", Package: ccPkg},
			resmgmt.WithRetry(retry.DefaultResMgmtOpts),
			resmgmt.WithTargetEndpoints(targets...),
		)
		if err != nil {
			return fmt.Errorf("SendInstallProposal return error: %s", err)
		}
	}
	return nil
}

func (d *CommonSteps) getLocalTargets(orgID string, blackListRegex string) ([]string, error) {
	return getLocalTargets(d.BDDContext, orgID, blackListRegex)
}

func getLocalTargets(context *BDDContext, orgID string, blackListRegex string) ([]string, error) {
	var blacklistedPeersRegex *regexp.Regexp
	if blackListRegex != "" {
		var err error
		blacklistedPeersRegex, err = regexp.Compile(blackListRegex)
		if err != nil {
			return nil, err
		}
	}

	contextProvider := func() (contextApi.Client, error) {
		return context.OrgUserContext(orgID, ADMIN), nil
	}

	localContext, err := contextImpl.NewLocal(contextProvider)
	if err != nil {
		return nil, err
	}

	peers, err := localContext.LocalDiscoveryService().GetPeers()
	if err != nil {
		return nil, err
	}

	var peerURLs []string
	for _, peer := range peers {
		if blacklistedPeersRegex != nil && blacklistedPeersRegex.MatchString(peer.URL()) {
			logger.Infof("Not returning local peer [%s] since it is blacklisted", peer.URL())
			continue
		}
		peerURLs = append(peerURLs, peer.URL())
	}

	return peerURLs, nil
}

func (d *CommonSteps) instantiateChaincodeWithOpts(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames string, allPeers bool) error {
	logger.Infof("Preparing to instantiate chaincode [%s] from path [%s] to orgs [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]", ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames)

	peers := d.OrgPeers(orgIDs, channelID)
	if len(peers) == 0 {
		return errors.Errorf("no peers found for orgs [%s]", orgIDs)
	}
	chaincodePolicy, err := d.newChaincodePolicy(ccPolicy, channelID)
	if err != nil {
		return fmt.Errorf("error creating endorsement policy: %s", err)
	}

	var sdkPeers []fabApi.Peer
	var orgID string

	for _, pconfig := range peers {
		orgID = pconfig.OrgID

		sdkPeer, err := d.BDDContext.OrgUserContext(orgID, ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: pconfig.Config})
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

	resMgmtClient := d.BDDContext.ResMgmtClient(orgID, ADMIN)

	logger.Infof("Instantiating chaincode [%s] from path [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s] to the following peers: [%s]", ccID, ccPath, channelID, args, ccPolicy, collectionNames, peersAsString(sdkPeers))

	_, err = resMgmtClient.InstantiateCC(
		channelID,
		resmgmt.InstantiateCCRequest{
			Name:       ccID,
			Path:       ccPath,
			Version:    "v1",
			Args:       GetByteArgs(strings.Split(args, ",")),
			Policy:     chaincodePolicy,
			CollConfig: collConfig,
		},
		resmgmt.WithTargets(sdkPeers...),
		resmgmt.WithTimeout(fabApi.Execute, 5*time.Minute),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
	)

	if err != nil && strings.Contains(err.Error(), "already exists") {
		logger.Warnf("error from InstantiateCC %v", err)
		return nil
	}
	return err
}

func (d *CommonSteps) deployChaincodeToOrg(ccType, ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames string) error {
	logger.Infof("Installing and instantiating chaincode [%s] from path [%s] to orgs [%s] on channel [%s] with args [%s] and CC policy [%s] and collectionPolicy [%s]", ccID, ccPath, orgIDs, channelID, args, ccPolicy, collectionNames)

	peers := d.OrgPeers(orgIDs, channelID)
	if len(peers) == 0 {
		return errors.Errorf("no peers found for orgs [%s]", orgIDs)
	}
	chaincodePolicy, err := d.newChaincodePolicy(ccPolicy, channelID)
	if err != nil {
		return fmt.Errorf("error creating endirsement policy: %s", err)
	}

	var sdkPeers []fabApi.Peer
	var isInstalled bool
	var orgID string

	for _, pconfig := range peers {
		orgID = pconfig.OrgID

		sdkPeer, err := d.BDDContext.OrgUserContext(orgID, ADMIN).InfraProvider().CreatePeerFromConfig(&fabApi.NetworkPeer{PeerConfig: pconfig.Config})
		if err != nil {
			return errors.WithMessage(err, "NewPeer failed")
		}
		resourceMgmt := d.BDDContext.ResMgmtClient(orgID, ADMIN)
		isInstalled, err = IsChaincodeInstalled(resourceMgmt, sdkPeer, ccID)
		if err != nil {
			return fmt.Errorf("Error querying installed chaincodes: %s", err)
		}

		if !isInstalled {

			resMgmtClient := d.BDDContext.ResMgmtClient(orgID, ADMIN)
			ccPkg, err := gopackager.NewCCPackage(ccPath, d.getDeployPath(ccType))
			if err != nil {
				return err
			}

			installRqst := resmgmt.InstallCCRequest{Name: ccID, Path: ccPath, Version: "v1", Package: ccPkg}
			_, err = resMgmtClient.InstallCC(installRqst, resmgmt.WithRetry(retry.DefaultResMgmtOpts))
			if err != nil {
				return fmt.Errorf("SendInstallProposal return error: %s", err)
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

	resMgmtClient := d.BDDContext.ResMgmtClient(orgID, ADMIN)

	instantiateRqst := resmgmt.InstantiateCCRequest{Name: ccID, Path: ccPath, Version: "v1", Args: GetByteArgs(argsArray), Policy: chaincodePolicy,
		CollConfig: collConfig}

	_, err = resMgmtClient.InstantiateCC(
		channelID, instantiateRqst,
		resmgmt.WithTargets(sdkPeers...),
		resmgmt.WithTimeout(fabApi.Execute, 5*time.Minute),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
	)
	return err
}

func (d *CommonSteps) newChaincodePolicy(ccPolicy, channelID string) (*fabricCommon.SignaturePolicyEnvelope, error) {
	if ccPolicy != "" {
		// Create a signature policy from the policy expression passed in
		return newPolicy(ccPolicy)
	}

	netwkConfig := d.BDDContext.clientConfig.NetworkConfig()

	// Default policy is 'signed by any member' for all known orgs
	var mspIDs []string
	for _, orgID := range d.BDDContext.OrgsByChannel(channelID) {
		orgConfig, ok := netwkConfig.Organizations[strings.ToLower(orgID)]
		if !ok {
			return nil, errors.Errorf("org config not found for org ID %s", orgID)
		}
		mspIDs = append(mspIDs, orgConfig.MSPID)
	}
	logger.Infof("Returning SignedByAnyMember policy for MSPs %s", mspIDs)
	return cauthdsl.SignedByAnyMember(mspIDs), nil
}

//OrgPeers return array of PeerConfig
func (d *CommonSteps) OrgPeers(orgIDs, channelID string) []*PeerConfig {
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
	logger.Infof("Warming up chaincode [%s] on channel [%s]", ccID, channelID)
	return d.warmUpCConOrg(ccID, "", channelID)
}

func (d *CommonSteps) warmUpCConOrg(ccID, orgIDs, channelID string) error {
	logger.Infof("Warming up chaincode [%s] on orgs [%s] and channel [%s]", ccID, orgIDs, channelID)
	for {
		_, err := d.QueryCCWithOpts(false, ccID, channelID, []string{"warmup"}, 5*time.Minute, false, 0, nil, d.OrgPeers(orgIDs, channelID)...)
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

func (d *CommonSteps) defineCollectionConfig(id, collection, policy string, requiredPeerCount int, maxPeerCount int) error {
	logger.Infof("Defining collection config [%s] for collection [%s] - policy=[%s], requiredPeerCount=[%d], maxPeerCount=[%d]", id, collection, policy, requiredPeerCount, maxPeerCount)
	d.BDDContext.DefineCollectionConfig(id, collection, policy, int32(requiredPeerCount), int32(maxPeerCount))
	return nil
}

// RegisterSteps register steps
func (d *CommonSteps) RegisterSteps(s *godog.Suite) {
	s.BeforeScenario(d.BDDContext.BeforeScenario)
	s.AfterScenario(d.BDDContext.AfterScenario)

	s.Step(`^the channel "([^"]*)" is created and all peers have joined$`, d.createChannelAndJoinAllPeers)
	s.Step(`^the channel "([^"]*)" is created and all peers from org "([^"]*)" have joined$`, d.createChannelAndJoinPeersFromOrg)
	s.Step(`^we wait (\d+) seconds$`, d.wait)
	s.Step(`^client queries chaincode "([^"]*)" with args "([^"]*)" on all peers in the "([^"]*)" org on the "([^"]*)" channel$`, d.queryCConOrg)
	s.Step(`^client queries system chaincode "([^"]*)" with args "([^"]*)" on org "([^"]*)" peer on the "([^"]*)" channel$`, d.querySystemCC)
	s.Step(`^response from "([^"]*)" to client contains value "([^"]*)"$`, d.containsInQueryValue)
	s.Step(`^response from "([^"]*)" to client equal value "([^"]*)"$`, d.equalQueryValue)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is installed from path "([^"]*)" to all peers$`, d.installChaincodeToAllPeers)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is installed from path "([^"]*)" to all peers in the "([^"]*)" org$`, d.installChaincodeToOrg)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is installed from path "([^"]*)" to all peers except "([^"]*)"$`, d.installChaincodeToAllPeersExcept)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is instantiated from path "([^"]*)" on all peers in the "([^"]*)" org on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.instantiateChaincodeOnOrg)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is instantiated from path "([^"]*)" on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.instantiateChaincode)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is deployed from path "([^"]*)" to all peers in the "([^"]*)" org on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.deployChaincodeToOrg)
	s.Step(`^"([^"]*)" chaincode "([^"]*)" is deployed from path "([^"]*)" to all peers on the "([^"]*)" channel with args "([^"]*)" with endorsement policy "([^"]*)" with collection policy "([^"]*)"$`, d.deployChaincode)
	s.Step(`^chaincode "([^"]*)" is warmed up on all peers in the "([^"]*)" org on the "([^"]*)" channel$`, d.warmUpCConOrg)
	s.Step(`^chaincode "([^"]*)" is warmed up on all peers on the "([^"]*)" channel$`, d.warmUpCC)
	s.Step(`^client invokes chaincode "([^"]*)" with args "([^"]*)" on all peers in the "([^"]*)" org on the "([^"]*)" channel$`, d.InvokeCConOrg)
	s.Step(`^collection config "([^"]*)" is defined for collection "([^"]*)" as policy="([^"]*)", requiredPeerCount=(\d+), and maxPeerCount=(\d+)$`, d.defineCollectionConfig)
	s.Step(`^block (\d+) from the "([^"]*)" channel is displayed$`, d.displayBlockFromChannel)
	s.Step(`^the last (\d+) blocks from the "([^"]*)" channel are displayed$`, d.displayBlocksFromChannel)
	s.Step(`^the last block from the "([^"]*)" channel is displayed$`, d.displayLastBlockFromChannel)

}
