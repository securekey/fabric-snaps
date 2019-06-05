/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package txsnapservice

import (
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	pb "github.com/hyperledger/fabric/protos/peer"
	metricsutil "github.com/securekey/fabric-snaps/metrics/pkg/util"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/api/endorse"
	txnSnapClient "github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/peerfilter"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

//PeerConfigPath use for testing
var PeerConfigPath = ""
var metrics *txnSnapClient.Metrics
var once sync.Once

//TxServiceImpl used to create transaction service
type TxServiceImpl struct {
	FcClient api.Client
	// Callback is invoked after the endorsement
	// phase of EndorseAndCommitTransaction
	// (Used in unit tests.)
	Callback api.EndorsedCallback
}

//Get will return txService to caller
func Get(channelID string) (*TxServiceImpl, errors.Error) {
	return newTxService(channelID)
}

//GetLocalPeer to returns target peer for given peer config
func (txs *TxServiceImpl) GetLocalPeer() (fabApi.Peer, error) {
	return txs.FcClient.GetLocalPeer()
}

//GetDiscoveredPeer the peer from the Discovery service that matches the given URL
//Returns error if no matching peer is found
func (txs *TxServiceImpl) GetDiscoveredPeer(url string) (fabApi.Peer, error) {
	return txs.FcClient.GetDiscoveredPeer(url)
}

//New creates new transaction snap service
func newTxService(channelID string) (*TxServiceImpl, errors.Error) {
	// we need to use once here to initialize metrics because newTxService is called from multiple snaps
	once.Do(func() { metrics = txnSnapClient.NewMetrics(metricsutil.GetMetricsInstance()) })
	client, err := txnSnapClient.GetInstance(channelID, metrics)
	if err != nil {
		return nil, errors.WithMessage(errors.TxClientInitError, err, "Cannot initialize client")
	}
	return &TxServiceImpl{
		FcClient: client,
	}, nil
}

func (txs *TxServiceImpl) createEndorseTxRequest(snapTxRequest *api.SnapTransactionRequest, peers []fabApi.Peer) (*api.EndorseTxRequest, errors.Error) {

	if snapTxRequest == nil {
		return nil, errors.New(errors.MissingRequiredParameterError, "SnapTxRequest is required")

	}
	if snapTxRequest.ChaincodeID == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "ChaincodeID is mandatory field of the SnapTransactionRequest")
	}
	if snapTxRequest.ChannelID == "" {
		return nil, errors.New(errors.MissingRequiredParameterError, "ChannelID is mandatory field of the SnapTransactionRequest")
	}

	//cc code args
	endorserArgs := snapTxRequest.EndorserArgs
	var ccargs []string
	for _, ccArg := range endorserArgs {
		ccargs = append(ccargs, string(ccArg))
	}

	fn := ""
	endorseMsg := "Endorser with empty args %s"
	if len(ccargs) > 0 {
		fn = ccargs[0]
		endorseMsg = "Endorser args[0]: %s"
	}

	logger.Debugf(endorseMsg, fn)

	var peerFilter api.PeerFilter
	if snapTxRequest.PeerFilter != nil {
		logger.Debugf("Using peer filter [%s]\n", snapTxRequest.PeerFilter.Type)
		var err error
		peerFilter, err = peerfilter.New(snapTxRequest.PeerFilter)
		if err != nil {
			return nil, errors.Wrap(errors.SystemError, err, "error creating Peer Filter")
		}
	}

	request := &api.EndorseTxRequest{
		ChaincodeID:          snapTxRequest.ChaincodeID,
		Args:                 ccargs,
		TransientData:        snapTxRequest.TransientMap,
		ChaincodeIDs:         snapTxRequest.CCIDsForEndorsement,
		Targets:              peers,
		PeerFilter:           peerFilter,
		CommitType:           snapTxRequest.CommitType,
		RWSetIgnoreNameSpace: snapTxRequest.RWSetIgnoreNameSpace,
		Nonce:                snapTxRequest.Nonce,
		TransactionID:        snapTxRequest.TransactionID,
	}
	return request, nil
}

//EndorseTransaction use to endorse the transaction
func (txs *TxServiceImpl) EndorseTransaction(snapTxRequest *api.SnapTransactionRequest, peers []fabApi.Peer, options ...endorse.RequestOption) (*channel.Response, errors.Error) {
	request, err := txs.createEndorseTxRequest(snapTxRequest, peers)
	if err != nil {
		return nil, err
	}
	value, err := txs.FcClient.EndorseTransaction(request, options...)
	if err != nil {
		return nil, err
	}

	return value, nil
}

//CommitTransaction use to comit the transaction
func (txs *TxServiceImpl) CommitTransaction(snapTxRequest *api.SnapTransactionRequest, peers []fabApi.Peer) (*channel.Response, bool, errors.Error) {
	request, err := txs.createEndorseTxRequest(snapTxRequest, peers)
	if err != nil {
		return nil, false, err
	}

	return txs.FcClient.CommitTransaction(request, snapTxRequest.RegisterTxEvent, txs.Callback)
}

//CommitOnlyTransaction just commits the data without endorsement
func (txs *TxServiceImpl) CommitOnlyTransaction(rwSetIgnoreNameSpace []api.Namespace, commitType api.CommitType, endorserResponse *channel.Response) (*channel.Response, bool, errors.Error) {
	return txs.FcClient.CommitOnlyTransaction(rwSetIgnoreNameSpace, commitType, endorserResponse, true, txs.Callback)
}

//InvokeSDKHandler invoke sdk handler
func (txs *TxServiceImpl) InvokeSDKHandler(handler invoke.Handler, request channel.Request, options ...channel.RequestOption) (*channel.Response, error) {
	return txs.FcClient.InvokeHandler(handler, request, options...)
}

//VerifyTxnProposalSignature use to verify transaction proposal signature
func (txs *TxServiceImpl) VerifyTxnProposalSignature(signedProposal *pb.SignedProposal) errors.Error {

	if signedProposal == nil {
		return errors.New(errors.MissingRequiredParameterError, "Signed proposal is missing")
	}

	proposalBytes, e := proto.Marshal(signedProposal)
	if e != nil {
		return errors.Wrap(errors.SystemError, e, "Cannot marshal signed proposal")
	}
	err := txs.FcClient.VerifyTxnProposalSignature(proposalBytes)
	if err != nil {
		return err
	}
	return nil
}

//VerifyEndorsements use to verify endorsements
func (txs *TxServiceImpl) VerifyEndorsements(endorsements []byte) errors.Error {
	if endorsements == nil {
		return errors.New(errors.MissingRequiredParameterError, "Signed proposal is missing")
	}

	err := txs.FcClient.VerifyEndorsements(endorsements)
	if err != nil {
		logger.Debugf("VerifyEndorsements failed %s", err)
		return err
	}
	logger.Debugf("VerifyEndorsements succeed")

	return nil
}
