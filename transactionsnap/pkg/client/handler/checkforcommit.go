/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	rwsetutil "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

//NewCheckForCommitHandler returns a handler that check if there is need to commit
func NewCheckForCommitHandler(rwSetIgnoreNameSpace []string, callback api.EndorsedCallback, next ...invoke.Handler) *CheckForCommitHandler {
	return &CheckForCommitHandler{rwSetIgnoreNameSpace: rwSetIgnoreNameSpace, callback: callback, next: getNext(next)}
}

//CheckForCommitHandler for checking need to commit
type CheckForCommitHandler struct {
	next                 invoke.Handler
	rwSetIgnoreNameSpace []string
	callback             api.EndorsedCallback
}

//Handle for endorsing transactions
func (c *CheckForCommitHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {

	txID := string(requestContext.Response.TransactionID)
	if c.callback != nil {
		if err := c.callback(requestContext.Response); err != nil {
			requestContext.Error = errors.WithMessage(err, "endorsed callback error")
			return
		}
	}

	logger.Debugf("Checking write sets to see if commit is necessary for Tx [%s]", txID)

	var err error

	// let's unmarshall one of the proposal responses to see if commit is needed
	prp := &pb.ProposalResponsePayload{}

	if requestContext.Response.Responses[0] == nil || requestContext.Response.Responses[0].ProposalResponse == nil || requestContext.Response.Responses[0].ProposalResponse.Payload == nil {
		requestContext.Error = errors.New("No proposal response payload")
		return
	}

	if err = proto.Unmarshal(requestContext.Response.Responses[0].ProposalResponse.Payload, prp); err != nil {
		requestContext.Error = errors.WithMessage(err, "Error unmarshaling to ProposalResponsePayload")
		return
	}

	ccAction := &pb.ChaincodeAction{}

	if err = proto.Unmarshal(prp.Extension, ccAction); err != nil {
		requestContext.Error = errors.WithMessage(err, "Error unmarshaling to ChaincodeAction")
		return
	}

	txRWSet := &rwsetutil.TxRwSet{}
	if err = txRWSet.FromProtoBytes(ccAction.Results); err != nil {
		requestContext.Error = errors.WithMessage(err, "Error unmarshaling to txRWSet")
		return
	}

	for _, nsRWSet := range txRWSet.NsRwSets {
		if contains(c.rwSetIgnoreNameSpace, nsRWSet.NameSpace) {
			// Ignore this writeset
			logger.Debugf("Ignoring writes to [%s] for Tx [%s]", nsRWSet.NameSpace, txID)
			continue
		}
		if nsRWSet.KvRwSet != nil && len(nsRWSet.KvRwSet.Writes) > 0 {
			logger.Debugf("Found writes to CC [%s] for Tx [%s]. A commit will be required.", nsRWSet.NameSpace, txID)
			c.next.Handle(requestContext, clientContext)
			return
		}
		for _, collRWSet := range nsRWSet.CollHashedRwSets {
			if collRWSet.HashedRwSet != nil && len(collRWSet.HashedRwSet.HashedWrites) > 0 {
				logger.Debugf("Found writes to private data collection [%s] in CC [%s] for Tx [%s]. A commit will be required.", collRWSet.CollectionName, nsRWSet.NameSpace, txID)
				c.next.Handle(requestContext, clientContext)
				return
			}
		}
	}

}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
