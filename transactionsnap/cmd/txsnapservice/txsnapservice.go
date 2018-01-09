/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txsnapservice

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	sdkConfigApi "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	"github.com/hyperledger/fabric-sdk-go/pkg/logging"
	pb "github.com/hyperledger/fabric/protos/peer"
	protosPeer "github.com/securekey/fabric-snaps/membershipsnap/api/membership"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	txnSnapClient "github.com/securekey/fabric-snaps/transactionsnap/cmd/client"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/peerfilter"
	txsnapconfig "github.com/securekey/fabric-snaps/transactionsnap/cmd/config"
)

var logger = logging.NewLogger("txnsnap")
var registerTxEventTimeout time.Duration = 30

//PeerConfigPath use for testing
var PeerConfigPath = ""

//DoIntializeChannel use for testing
var DoIntializeChannel = true

// clientServiceImpl implements client service
type clientServiceImpl struct {
}

var clientService = newClientService()

//TxServiceImpl used to create transaction service
type TxServiceImpl struct {
	Config     api.Config
	FcClient   api.Client
	Membership api.MembershipManager
}

//Get will return txService to caller
func Get(channelID string) (*TxServiceImpl, error) {
	return newTxService(channelID)
}

type apiConfig struct {
	api.Config
}

//QueryChannels to query channels based on peer
func (txs *TxServiceImpl) QueryChannels(targetPeer sdkApi.Peer) ([]string, error) {
	channels, err := txs.FcClient.QueryChannels(targetPeer)
	if err != nil {
		return nil, errors.Errorf("Error querying channels on %v: %s", targetPeer, err)
	}
	return channels, nil
}

//ClientConfig to return client config
func (txs *TxServiceImpl) ClientConfig() sdkConfigApi.Config {
	return txs.FcClient.GetConfig()
}

//New creates new transaction snap service
func newTxService(channelID string) (*TxServiceImpl, error) {
	txService := &TxServiceImpl{}
	config, err := txsnapconfig.NewConfig(PeerConfigPath, channelID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to initialize config: %s", err)
		logger.Errorf(errMsg)
		return txService, err
	}

	if config == nil || config.GetConfigBytes() == nil {
		return nil, fmt.Errorf("config from ledger is nil")
	}

	fcClient, err := txnSnapClient.GetInstance(channelID, &apiConfig{config})
	if err != nil {
		return nil, errors.Errorf("Cannot initialize client %v", err)
	}

	membership := clientService.GetClientMembership(config)
	txService.Config = config
	txService.FcClient = fcClient
	txService.Membership = membership
	return txService, nil

}

//EndorseTransaction use to endorse the transaction
func (txs *TxServiceImpl) EndorseTransaction(snapTxRequest *api.SnapTransactionRequest, peers []sdkApi.Peer) ([]*apitxn.TransactionProposalResponse, error) {
	if snapTxRequest == nil {
		return nil, errors.Errorf("SnapTxRequest is required")

	}
	if snapTxRequest.ChaincodeID == "" {
		return nil, errors.Errorf("ChaincodeID is mandatory field of the SnapTransactionRequest")
	}
	if snapTxRequest.ChannelID == "" {
		return nil, errors.Errorf("ChannelID is mandatory field of the SnapTransactionRequest")
	}

	channel, err := txs.FcClient.NewChannel(snapTxRequest.ChannelID)
	if err != nil {
		return nil, errors.Errorf("Cannot create channel %v", err)
	}

	// //cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))
	}
	logger.Debug("Endorser args:", ccargs)

	var peerFilter api.PeerFilter
	if snapTxRequest.PeerFilter != nil {
		logger.Infof("Using peer filter [%s]\n", snapTxRequest.PeerFilter.Type)
		var err error
		peerFilter, err = peerfilter.New(snapTxRequest.PeerFilter)
		if err != nil {
			return nil, errors.Wrap(err, "error creating Peer Filter")
		}
	}

	request := &api.EndorseTxRequest{
		ChaincodeID:   snapTxRequest.ChaincodeID,
		Args:          ccargs,
		TransientData: snapTxRequest.TransientMap,
		ChaincodeIDs:  snapTxRequest.CCIDsForEndorsement,
		Targets:       peers,
		PeerFilter:    peerFilter,
	}
	tpxResponse, err := txs.FcClient.EndorseTransaction(channel, request)
	if err != nil {
		return nil, err
	}
	return tpxResponse, nil
}

//CommitTransaction use to comit the transaction
func (txs *TxServiceImpl) CommitTransaction(channelID string, tpResponses []*apitxn.TransactionProposalResponse, registerTxEvent bool, timeout time.Duration) (pb.TxValidationCode, error) {
	if channelID == "" {
		return pb.TxValidationCode(-1), errors.Errorf("ChannelID is mandatory field of the SnapTransactionRequest")
	}
	//	channel, err := txs.FcClient.NewChannel(channelID)
	channel, err := txs.getChannel(channelID)
	if err != nil {
		//what code should be returned here
		return pb.TxValidationCode(-1), errors.Errorf("Cannot create channel %v", err)
	}

	err = txs.FcClient.CommitTransaction(channel, tpResponses, registerTxEvent, registerTxEventTimeout)
	if err != nil {
		return pb.TxValidationCode(-1), errors.Errorf("CommitTransaction returned error: %v", err)
	}
	return pb.TxValidationCode(pb.TxValidationCode_VALID), nil

}

//EndorseAndCommitTransaction use to endorse and commit transaction
func (txs *TxServiceImpl) EndorseAndCommitTransaction(snapTxRequest *api.SnapTransactionRequest, peers []sdkApi.Peer, timeout time.Duration) (pb.TxValidationCode, error) {

	if snapTxRequest == nil {
		return pb.TxValidationCode(-1), errors.Errorf("SnapTxRequest is required")
	}

	if snapTxRequest.ChaincodeID == "" {
		return pb.TxValidationCode(-1), errors.Errorf("ChaincodeID is mandatory field of the SnapTransactionRequest")
	}
	if snapTxRequest.ChannelID == "" {
		return pb.TxValidationCode(-1), errors.Errorf("ChannelID is mandatory field of the SnapTransactionRequest")
	}

	tpxResponse, err := txs.EndorseTransaction(snapTxRequest, peers)
	if err != nil {
		return pb.TxValidationCode(-1), err
	}
	newTxID := tpxResponse[0].Proposal.TxnID
	logger.Debugf("newTxID: %s", newTxID)

	// Channel already checked in endorseTransaction
	//channel, _ := txs.FcClient.NewChannel(snapTxRequest.ChannelID)
	channel, err := txs.getChannel(snapTxRequest.ChannelID)
	if err != nil {
		return pb.TxValidationCode(-1), err
	}
	err = txs.FcClient.CommitTransaction(channel, tpxResponse, snapTxRequest.RegisterTxEvent, registerTxEventTimeout)

	if err != nil {
		return pb.TxValidationCode(-1), errors.Errorf("CommitTransaction returned error: %v", err)
	}
	return pb.TxValidationCode(pb.TxValidationCode_VALID), nil

}

//VerifyTxnProposalSignature use to verify transaction proposal signature
func (txs *TxServiceImpl) VerifyTxnProposalSignature(channelID string, signedProposal *pb.SignedProposal) error {

	if channelID == "" {
		return fmt.Errorf("ChannelID is mandatory field of the SnapTransactionRequest")
	}

	channel, err := txs.getChannel(channelID)
	if err != nil {
		return fmt.Errorf("Cannot create channel %v", err)
	}

	if signedProposal == nil {
		return fmt.Errorf("Signed proposal is missing")
	}
	err = txs.initializeChannel(channel)
	if err != nil {
		return fmt.Errorf("Cannot initialize channel %v", err)
	}

	proposalBytes, err := proto.Marshal(signedProposal)
	if err != nil {
		return fmt.Errorf("Cannot unmarshal proposlaBytes  %v", err)
	}
	err = txs.FcClient.VerifyTxnProposalSignature(channel, proposalBytes)
	if err != nil {
		return fmt.Errorf("VerifyTxnProposalSignature returned error: %v", err)
	}
	return nil
}

//GetPeersOfChannel use to get peers of channel
func (txs *TxServiceImpl) GetPeersOfChannel(args []string, membership api.MembershipManager) ([]byte, error) {
	if len(args) < 1 || args[0] == "" {
		return nil, fmt.Errorf("Channel name must be provided")
	}

	// First argument is channel
	channel := args[0]
	logger.Debugf("Retrieving peers on channel: %s", channel)

	channelMembership := membership.GetPeersOfChannel(channel)
	if channelMembership.QueryError != nil && channelMembership.Peers == nil {
		return nil, fmt.Errorf("Could not get peers on channel %s: %s", channel, channelMembership.QueryError)
	}
	if channelMembership.QueryError != nil && channelMembership.Peers != nil {
		logger.Warnf(
			"Error polling peers on channel %s, using last known configuration. Error: %s",
			channelMembership.QueryError)
	}

	logger.Debugf("Peers on channel(%s): %s", channel, channelMembership.Peers)

	// Construct list of endpoints
	endpoints := make([]protosPeer.PeerEndpoint, 0, len(channelMembership.Peers))
	for _, peer := range channelMembership.Peers {
		endpoints = append(endpoints, protosPeer.PeerEndpoint{Endpoint: peer.URL(), MSPid: []byte(peer.MSPID())})
	}

	peerBytes, err := json.Marshal(endpoints)
	if err != nil {
		return nil, err
	}

	return peerBytes, nil

}
func (txs *TxServiceImpl) getChannel(channelID string) (sdkApi.Channel, error) {
	if !DoIntializeChannel {
		channel, err := txs.FcClient.GetChannel(channelID)
		if err != nil {
			return nil, err
		}
		return channel, nil
	}
	channel, err := txs.FcClient.NewChannel(channelID)
	if err != nil {
		return nil, err
	}
	return channel, nil
}

func (txs *TxServiceImpl) initializeChannel(channel sdkApi.Channel) error {
	if !DoIntializeChannel {
		return nil
	}
	err := txs.FcClient.InitializeChannel(channel)
	if err != nil {
		return fmt.Errorf("Cannot initialize channel %v", err)
	}
	return nil
}

//utility methods
func newClientService() api.ClientService {
	return &clientServiceImpl{}
}

// GetFabricClient return fabric client
func (cs *clientServiceImpl) GetFabricClient(channelID string, config api.Config) (api.Client, error) {
	fcClient, err := txnSnapClient.GetInstance(channelID, config)
	if err != nil {
		return nil, fmt.Errorf("Cannot initialize client %v", err)
	}
	return fcClient, nil
}

// GetClientMembership return client membership
func (cs *clientServiceImpl) GetClientMembership(config api.Config) api.MembershipManager {
	// membership mananger
	membership := txnSnapClient.GetMembershipInstance(config)

	return membership
}

// getSnapTransactionRequest
func getSnapTransactionRequest(snapTransactionRequestbBytes []byte) (*api.SnapTransactionRequest, error) {
	var snapTxRequest api.SnapTransactionRequest
	err := json.Unmarshal(snapTransactionRequestbBytes, &snapTxRequest)
	if err != nil {
		return nil, fmt.Errorf("Cannot decode parameters from request to Snap Transaction Request %v", err)
	}
	return &snapTxRequest, nil
}
