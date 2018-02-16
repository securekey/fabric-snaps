/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	eventservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
)

//NewLocalEventCommitHandler returns a handler that commit txn
func NewLocalEventCommitHandler(next ...chclient.Handler) *LocalEventCommitHandler {
	return &LocalEventCommitHandler{next: getNext(next)}
}

//LocalEventCommitHandler for commit txn
type LocalEventCommitHandler struct {
	next chclient.Handler
}

//Handle for endorsing transactions
func (l *LocalEventCommitHandler) Handle(requestContext *chclient.RequestContext, clientContext *chclient.ClientContext) {
	txnID := requestContext.Response.TransactionID

	events := eventservice.Get(clientContext.Channel.Name())
	reg, eventch, err := events.RegisterTxStatusEvent(txnID.ID)
	if err != nil {
		requestContext.Error = errors.Wrapf(err, "unable to register for TxStatus event for TxID [%s] on channel [%s]", txnID, clientContext.Channel.Name())
		return
	}
	defer events.Unregister(reg)
	txStatusEventCh := eventch

	_, err = createAndSendTransaction(clientContext.Channel, requestContext.Response.Responses)
	if err != nil {
		requestContext.Error = errors.Wrap(err, "CreateAndSendTransaction failed")
		return
	}

	select {
	case txStatusEvent := <-txStatusEventCh:

		requestContext.Response.TxValidationCode = pb.TxValidationCode(txStatusEvent.TxValidationCode)
		if requestContext.Response.TxValidationCode != pb.TxValidationCode_VALID {
			requestContext.Error = errors.Errorf("transaction [%s] did not commit successfully. Code: [%s]", txnID.ID, txStatusEvent.TxValidationCode)
			return
		}
	case <-time.After(requestContext.Opts.Timeout):
		requestContext.Error = errors.New("Execute didn't receive block event")
		return
	}

	//Delegate to next step if any
	if l.next != nil {
		l.next.Handle(requestContext, clientContext)
	}
}

func createAndSendTransaction(sender apifabclient.Sender, resps []*apifabclient.TransactionProposalResponse) (*apifabclient.TransactionResponse, error) {

	tx, err := sender.CreateTransaction(resps)
	if err != nil {
		return nil, errors.WithMessage(err, "CreateTransaction failed")
	}

	transactionResponse, err := sender.SendTransaction(tx)
	if err != nil {
		return nil, errors.WithMessage(err, "SendTransaction failed")

	}
	if transactionResponse.Err != nil {
		logger.Debugf("orderer %s failed (%s)", transactionResponse.Orderer, transactionResponse.Err.Error())
		return nil, errors.Wrap(transactionResponse.Err, "orderer failed")
	}

	return transactionResponse, nil
}
