/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import
(
	"github.com/pkg/errors"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/txn"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric/protos/utils"
)

// CustomEndorsementHandler for handling endorsement of transactions
type CustomEndorsementHandler struct {
	next         invoke.Handler
}

// Handle for endorsing transactions
func (e *CustomEndorsementHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {

	if len(requestContext.Opts.Targets) == 0 {
		requestContext.Error = status.New(status.ClientStatus, status.NoPeersFound.ToInt32(), "targets were not provided", nil)
		return
	}

	// Endorse Tx
	transactionProposalResponses, proposal, err := createAndSendTransactionProposal(clientContext.Transactor, &requestContext.Request, peer.PeersToTxnProcessors(requestContext.Opts.Targets))

	requestContext.Response.Proposal = proposal
	requestContext.Response.TransactionID = proposal.TxnID // TODO: still needed?

	if err != nil {
		requestContext.Error = err
		return
	}

	requestContext.Response.Responses = transactionProposalResponses
	if len(transactionProposalResponses) > 0 {
		requestContext.Response.Payload = transactionProposalResponses[0].ProposalResponse.GetResponse().Payload
		requestContext.Response.ChaincodeStatus = transactionProposalResponses[0].ChaincodeStatus
	}

	//Delegate to next step if any
	if e.next != nil {
		e.next.Handle(requestContext, clientContext)
	}
}

// createAndSendTransactionProposal has the client sign the proposal payload without transients and includes that signature in the proposal
func createAndSendTransactionProposal(transactor fab.ProposalSender, chrequest *invoke.Request, targets []fab.ProposalProcessor) ([]*fab.TransactionProposalResponse, *fab.TransactionProposal, error) {
	request := fab.ChaincodeInvokeRequest{
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

	// remove transients so client can sign payload
	ccProposalPayload, err := utils.GetChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "get chaincode proposal payload failed")
	}

	ccProposalPayload.TransientMap = nil

	payloadBytesWithoutTransients, err := utils.GetBytesChaincodeProposalPayload(ccProposalPayload)
	if err != nil {
		return nil, nil, errors.WithMessage(err, "marshal of chaincode proposal payload to bytes failed")
	}


	// client needs to sign payloadBytesWithoutTransients
	// get signingManager
	// signature, err := signingManager.Sign(payloadBytesWithoutTransients, privateKey)

	transactionProposalResponses, err := transactor.SendTransactionProposal(proposal, targets)

	return transactionProposalResponses, proposal, err
}


