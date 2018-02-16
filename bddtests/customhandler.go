/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	"github.com/pkg/errors"
)

// CustomEndorsementHandler ignores the channel in the ClientContext
// and instead sends the proposal to the given channel
type CustomEndorsementHandler struct {
	channel apifabclient.Channel
	next    chclient.Handler
}

// Handle handles an endorsement proposal
func (h *CustomEndorsementHandler) Handle(requestContext *chclient.RequestContext, clientContext *chclient.ClientContext) {
	logger.Info("customEndorsementHandler - Invoking chaincode on system channel")

	if !clientContext.EventHub.IsConnected() {
		err := clientContext.EventHub.Connect()
		if err != nil {
			requestContext.Error = err
			return
		}
	}

	transactionProposalResponses, txnID, err := createAndSendTransactionProposal(h.channel, &requestContext.Request, requestContext.Opts.ProposalProcessors)

	requestContext.Response.TransactionID = txnID

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
func NewCustomEndorsementHandler(channel apifabclient.Channel, next chclient.Handler) *CustomEndorsementHandler {
	return &CustomEndorsementHandler{
		channel: channel,
		next:    next,
	}
}

func createAndSendTransactionProposal(sender apifabclient.ProposalSender, chrequest *chclient.Request, targets []apifabclient.ProposalProcessor) ([]*apifabclient.TransactionProposalResponse, apifabclient.TransactionID, error) {
	request := apifabclient.ChaincodeInvokeRequest{
		ChaincodeID:  chrequest.ChaincodeID,
		Fcn:          chrequest.Fcn,
		Args:         chrequest.Args,
		TransientMap: chrequest.TransientMap,
	}

	//logger.Debugf("sending transaction proposal with ChaincodeID: [%s] Fcn: [%s] Args: [%s]", request.ChaincodeID, request.Fcn, request.Args)

	transactionProposalResponses, txnID, err := sender.SendTransactionProposal(request, targets)
	if err != nil {
		return nil, txnID, err
	}

	for _, v := range transactionProposalResponses {
		if v.Err != nil {
			return nil, txnID, errors.WithMessage(v.Err, "SendTransactionProposal failed")
		}
	}
	return transactionProposalResponses, txnID, nil
}
