/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"fmt"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
)

//NewCommitTxHandler returns a handler that commit txn
func NewCommitTxHandler(registerTxEvent bool, channelID string, next ...invoke.Handler) *CommitTxHandler {
	return &CommitTxHandler{registerTxEvent: registerTxEvent, channelID: channelID, next: getNext(next)}
}

//CommitTxHandler for commit txn
type CommitTxHandler struct {
	next            invoke.Handler
	registerTxEvent bool
	channelID       string
}

//Handle for endorsing transactions
func (l *CommitTxHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	txnID := requestContext.Response.TransactionID

	//Register Tx event
	reg, statusNotifier, err := clientContext.EventService.RegisterTxStatusEvent(string(txnID)) // TODO: Change func to use TransactionID instead of string
	if err != nil {
		requestContext.Error = errors.Wrap(err, "error registering for TxStatus event")
		return
	}
	defer clientContext.EventService.Unregister(reg)

	_, err = createAndSendTransaction(clientContext.Transactor, requestContext.Response.Proposal, requestContext.Response.Responses)
	if err != nil {
		requestContext.Error = errors.Wrap(err, "CreateAndSendTransaction failed")
		return
	}
	if l.registerTxEvent {

		select {
		case txStatusEvent := <-statusNotifier:

			requestContext.Response.TxValidationCode = pb.TxValidationCode(txStatusEvent.TxValidationCode)
			if requestContext.Response.TxValidationCode != pb.TxValidationCode_VALID {
				requestContext.Error = status.New(status.EventServerStatus, int32(txStatusEvent.TxValidationCode),
					fmt.Sprintf("transaction [%s] did not commit successfully", txnID), nil)
				return
			}
		case <-requestContext.Ctx.Done():
			requestContext.Error = errors.New("Execute didn't receive block event")
			return
		}
	}
	//Delegate to next step if any
	if l.next != nil {
		l.next.Handle(requestContext, clientContext)
	}
}

func createAndSendTransaction(sender fabApi.Sender, proposal *fabApi.TransactionProposal, resps []*fabApi.TransactionProposalResponse) (*fabApi.TransactionResponse, error) {

	txnRequest := fabApi.TransactionRequest{
		Proposal:          proposal,
		ProposalResponses: resps,
	}

	tx, err := sender.CreateTransaction(txnRequest)
	if err != nil {
		return nil, errors.WithMessage(err, "CreateTransaction failed")
	}

	transactionResponse, err := sender.SendTransaction(tx)
	if err != nil {
		return nil, errors.WithMessage(err, "SendTransaction failed")

	}

	return transactionResponse, nil
}
