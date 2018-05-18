/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txsnapservice

import (
	"github.com/gogo/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	txnSnapClient "github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/peerfilter"
	txsnapconfig "github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

//PeerConfigPath use for testing
var PeerConfigPath = ""

// clientServiceImpl implements client service
type clientServiceImpl struct {
}

var clientService = newClientService()

//TxServiceImpl used to create transaction service
type TxServiceImpl struct {
	Config   api.Config
	FcClient api.Client
	// Callback is invoked after the endorsement
	// phase of EndorseAndCommitTransaction
	// (Used in unit tests.)
	Callback api.EndorsedCallback
}

//Get will return txService to caller
func Get(channelID string) (*TxServiceImpl, error) {
	return newTxService(channelID)
}

type apiConfig struct {
	api.Config
}

//GetTargetPeer to returns target peer for given peer config
func (txs *TxServiceImpl) GetTargetPeer(peerCfg *api.PeerConfig, opts ...peer.Option) (fabApi.Peer, error) {
	return txs.FcClient.GetTargetPeer(peerCfg, opts...)
}

//New creates new transaction snap service
func newTxService(channelID string) (*TxServiceImpl, error) {
	txService := &TxServiceImpl{}
	config, err := txsnapconfig.NewConfig(PeerConfigPath, channelID)
	if err != nil {
		return txService, errors.WithMessage(errors.GeneralError, err, "Failed to initialize config")
	}

	if config == nil || config.GetConfigBytes() == nil {
		return nil, errors.New(errors.GeneralError, "config from ledger is nil")
	}

	fcClient, err := txnSnapClient.GetInstance(channelID, &apiConfig{config})
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Cannot initialize client")
	}

	txService.Config = config
	txService.FcClient = fcClient
	return txService, nil

}

func (txs *TxServiceImpl) createEndorseTxRequest(snapTxRequest *api.SnapTransactionRequest, peers []fabApi.Peer) (*api.EndorseTxRequest, error) {

	if snapTxRequest == nil {
		return nil, errors.New(errors.GeneralError, "SnapTxRequest is required")

	}
	if snapTxRequest.ChaincodeID == "" {
		return nil, errors.New(errors.GeneralError, "ChaincodeID is mandatory field of the SnapTransactionRequest")
	}
	if snapTxRequest.ChannelID == "" {
		return nil, errors.New(errors.GeneralError, "ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))
	}
	logger.Debug("Endorser args:", ccargs)

	var peerFilter api.PeerFilter
	if snapTxRequest.PeerFilter != nil {
		logger.Debugf("Using peer filter [%s]\n", snapTxRequest.PeerFilter.Type)
		var err error
		peerFilter, err = peerfilter.New(snapTxRequest.PeerFilter)
		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "error creating Peer Filter")
		}
	}

	request := &api.EndorseTxRequest{
		ChaincodeID:          snapTxRequest.ChaincodeID,
		Args:                 ccargs,
		TransientData:        snapTxRequest.TransientMap,
		ChaincodeIDs:         snapTxRequest.CCIDsForEndorsement,
		Targets:              peers,
		PeerFilter:           peerFilter,
		RWSetIgnoreNameSpace: snapTxRequest.RWSetIgnoreNameSpace,
	}
	return request, nil
}

//EndorseTransaction use to endorse the transaction
func (txs *TxServiceImpl) EndorseTransaction(snapTxRequest *api.SnapTransactionRequest, peers []fabApi.Peer) (*channel.Response, error) {
	request, err := txs.createEndorseTxRequest(snapTxRequest, peers)
	if err != nil {
		return nil, err
	}
	value, err := txs.FcClient.EndorseTransaction(request)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Received error when endorsing Tx")
	}

	return value, nil
}

//CommitTransaction use to comit the transaction
func (txs *TxServiceImpl) CommitTransaction(snapTxRequest *api.SnapTransactionRequest, peers []fabApi.Peer) (*channel.Response, error) {
	request, err := txs.createEndorseTxRequest(snapTxRequest, peers)
	if err != nil {
		return nil, err
	}
	tpr, err := txs.FcClient.CommitTransaction(request, snapTxRequest.RegisterTxEvent, txs.Callback)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Received error from CommitTransaction")
	}

	return tpr, nil

}

//VerifyTxnProposalSignature use to verify transaction proposal signature
func (txs *TxServiceImpl) VerifyTxnProposalSignature(signedProposal *pb.SignedProposal) error {

	if signedProposal == nil {
		return errors.New(errors.GeneralError, "Signed proposal is missing")
	}

	proposalBytes, err := proto.Marshal(signedProposal)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "Cannot unmarshal proposlaBytes")
	}
	err = txs.FcClient.VerifyTxnProposalSignature(proposalBytes)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "VerifyTxnProposalSignature returned error")
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
		return nil, errors.WithMessage(errors.GeneralError, err, "Cannot initialize client")
	}
	return fcClient, nil
}
