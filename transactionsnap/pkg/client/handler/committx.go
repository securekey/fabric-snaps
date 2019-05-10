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
	fmt.Printf("Parasu: reqContext.Response.len=%d; txId=%s\n",
		len(requestContext.Response.Responses),
		requestContext.Response.TransactionID)
	txnID := requestContext.Response.TransactionID

	//Register Tx event
	reg, statusNotifier, err := clientContext.EventService.RegisterTxStatusEvent(string(txnID)) // TODO: Change func to use TransactionID instead of string
	if err != nil {
		fmt.Printf("Parasu: Cannot register TxStatusEvent\n")
		requestContext.Error = errors.Wrap(err, "error registering for TxStatus event")
		return
	}
	fmt.Printf("Parasu: Successfully registered TxStatusEvent\n")
	defer clientContext.EventService.Unregister(reg)

	_, err = createAndSendTransaction(clientContext.Transactor, requestContext.Response.Proposal, requestContext.Response.Responses)
	if err != nil {
		fmt.Printf("Parasu: failed createAndSendTransaction\n")
		requestContext.Error = errors.Wrap(err, "CreateAndSendTransaction failed")
		return
	}
	fmt.Printf("Parasu: Successfully createAndSendTransaction; registerTxEvent=%b\n", l.registerTxEvent)
	if l.registerTxEvent {

		select {
		case txStatusEvent := <-statusNotifier:

			requestContext.Response.TxValidationCode = txStatusEvent.TxValidationCode
			if requestContext.Response.TxValidationCode != pb.TxValidationCode_VALID {
				requestContext.Error = status.New(status.EventServerStatus, int32(txStatusEvent.TxValidationCode),
					fmt.Sprintf("transaction [%s] did not commit successfully", txnID), nil)
				return
			}
		case <-requestContext.Ctx.Done():
			fmt.Printf("Parasu: time out=%b\n", l.registerTxEvent)
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
