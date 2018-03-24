/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	eventservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
)

//NewLocalEventCommitHandler returns a handler that commit txn
func NewLocalEventCommitHandler(registerTxEvent bool, channelID string, next ...invoke.Handler) *LocalEventCommitHandler {
	return &LocalEventCommitHandler{registerTxEvent: registerTxEvent, channelID: channelID, next: getNext(next)}
}

//LocalEventCommitHandler for commit txn
type LocalEventCommitHandler struct {
	next            invoke.Handler
	registerTxEvent bool
	channelID       string
}

//Handle for endorsing transactions
func (l *LocalEventCommitHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	txnID := string(requestContext.Response.TransactionID)
	var txStatusEventCh <-chan *fabApi.TxStatusEvent
	if l.registerTxEvent {
		//TODO
		events := eventservice.Get(l.channelID)
		reg, eventch, err := events.RegisterTxStatusEvent(txnID)
		if err != nil {
			requestContext.Error = errors.Wrapf(err, "unable to register for TxStatus event for TxID [%s] on channel [%s]", txnID, l.channelID)
			return
		}
		defer events.Unregister(reg)
		txStatusEventCh = eventch
	}
	_, err := createAndSendTransaction(clientContext.Transactor, requestContext.Response.Proposal, requestContext.Response.Responses)
	if err != nil {
		requestContext.Error = errors.Wrap(err, "CreateAndSendTransaction failed")
		return
	}
	if l.registerTxEvent {

		select {
		case txStatusEvent := <-txStatusEventCh:

			requestContext.Response.TxValidationCode = pb.TxValidationCode(txStatusEvent.TxValidationCode)
			if requestContext.Response.TxValidationCode != pb.TxValidationCode_VALID {
				requestContext.Error = status.New(status.EventServerStatus, int32(txStatusEvent.TxValidationCode),
					fmt.Sprintf("transaction [%s] did not commit successfully", txnID), nil)
				return
			}
		case <-time.After(requestContext.Opts.Timeouts[core.Execute]):
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
