/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/chconfig"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/txn"

	"github.com/hyperledger/fabric-sdk-go/pkg/context"
	"github.com/pkg/errors"
)

// CustomEndorsementHandler ignores the channel in the ClientContext
// and instead sends the proposal to the given channel
type CustomEndorsementHandler struct {
	context contextApi.Client
	next    invoke.Handler
}

// Handle handles an endorsement proposal
func (h *CustomEndorsementHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	logger.Info("customEndorsementHandler - Invoking chaincode on system channel")

	grpcContext, cancel := context.NewRequest(h.context, context.WithTimeoutType(fabApi.PeerResponse))
	defer cancel()
	sysTransactor, err := channel.NewTransactor(grpcContext, chconfig.NewChannelCfg(""))
	if err != nil {
		requestContext.Error = err
		return
	}
	// Endorse Tx
	transactionProposalResponses, proposal, err := createAndSendTransactionProposal(sysTransactor, &requestContext.Request, peer.PeersToTxnProcessors(requestContext.Opts.Targets))

	requestContext.Response.Proposal = proposal
	requestContext.Response.TransactionID = proposal.TxnID

	if err != nil {
		requestContext.Error = err
		return
	}

	requestContext.Response.Responses = transactionProposalResponses
	if len(transactionProposalResponses) > 0 {
		requestContext.Response.Payload = transactionProposalResponses[0].ProposalResponse.GetResponse().Payload
	}

	//Delegate to next step if any
	if h.next != nil {
		h.next.Handle(requestContext, clientContext)
	}
}

// NewCustomEndorsementHandler creates a new instance of CustomEndorsementHandler
func NewCustomEndorsementHandler(context contextApi.Client, next invoke.Handler) *CustomEndorsementHandler {
	return &CustomEndorsementHandler{
		context: context,
		next:    next,
	}
}

func createAndSendTransactionProposal(transactor fabApi.Transactor, chrequest *invoke.Request, targets []fabApi.ProposalProcessor) ([]*fabApi.TransactionProposalResponse, *fabApi.TransactionProposal, error) {
	request := fabApi.ChaincodeInvokeRequest{
		ChaincodeID:  chrequest.ChaincodeID,
		Fcn:          chrequest.Fcn,
		Args:         chrequest.Args,
		TransientMap: chrequest.TransientMap,
	}

	txh, err := transactor.CreateTransactionHeader()
	if err != nil {
		return nil, nil, errors.WithMessage(err, "creating transaction header failed")
	}

	proposal, err := txn.CreateChaincodeInvokeProposal(txh, request)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "creating transaction proposal failed")
	}

	transactionProposalResponses, err := transactor.SendTransactionProposal(proposal, targets)
	return transactionProposalResponses, proposal, err
}
