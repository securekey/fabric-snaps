/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	rwsetutil "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

//NewCheckForCommitHandler returns a handler that check if there is need to commit
func NewCheckForCommitHandler(rwSetIgnoreNameSpace []api.Namespace, callback api.EndorsedCallback, commitType api.CommitType, next ...invoke.Handler) *CheckForCommitHandler {
	return &CheckForCommitHandler{rwSetIgnoreNameSpace: rwSetIgnoreNameSpace, callback: callback, commitType: commitType, next: getNext(next)}
}

//CheckForCommitHandler for checking need to commit
type CheckForCommitHandler struct {
	next                 invoke.Handler
	rwSetIgnoreNameSpace []api.Namespace
	callback             api.EndorsedCallback
	commitType           api.CommitType
	ShouldCommit         bool
}

//Handle for endorsing transactions
func (c *CheckForCommitHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	response := requestContext.Response
	txID := string(response.TransactionID)

	if c.callback != nil {
		if err := c.callback(response); err != nil {
			requestContext.Error = errors.WithMessage(err, "endorsed callback error")
			return
		}
	}

	if c.commitType == api.NoCommit {
		logger.Debugf("[txID %s] No commit is necessary since commit type is [%s]", txID, c.commitType)
		return
	}

	if c.commitType == api.Commit {
		logger.Debugf("[txID %s] Commit is necessary since commit type is [%s]", txID, c.commitType)
		c.next.Handle(requestContext, clientContext)
		return
	}
	logger.Debugf("1.c.ShouldCommit=%b", c.ShouldCommit)
	c.commitIfHasWriteSet(txID, requestContext, response, clientContext)
}
func (c *CheckForCommitHandler) commitIfHasWriteSet(txID string, requestContext *invoke.RequestContext, response invoke.Response, clientContext *invoke.ClientContext) {
	var err error

	// let's unmarshall one of the proposal responses to see if commit is needed
	prp := &pb.ProposalResponsePayload{}

	if response.Responses[0] == nil || response.Responses[0].ProposalResponse == nil || response.Responses[0].ProposalResponse.Payload == nil {
		requestContext.Error = errors.New("No proposal response payload")
		return
	}

	if err = proto.Unmarshal(response.Responses[0].ProposalResponse.Payload, prp); err != nil {
		requestContext.Error = errors.WithMessage(err, "Error unmarshaling to ProposalResponsePayload")
		return
	}
	plArr, _ := json.Marshal(prp)
	logger.Debugf("Parasu: response.Responses[0].ProposalResponse.Payload=\n%s\n", string(plArr))

	ccAction := &pb.ChaincodeAction{}
	if err = proto.Unmarshal(prp.Extension, ccAction); err != nil {
		requestContext.Error = errors.WithMessage(err, "Error unmarshaling to ChaincodeAction")
		return
	}

	plArr, _ = json.Marshal(ccAction)
	logger.Debugf("Parasu: prp.Extension=\n%s\n", string(plArr))

	if len(ccAction.Events) > 0 {
		logger.Debugf("[txID %s] Commit is necessary since commit type is [%s] and chaincode event exists in proposal response", txID, api.CommitOnWrite)
		c.ShouldCommit = true
	} else {
		txRWSet := &rwsetutil.TxRwSet{}
		if err = txRWSet.FromProtoBytes(ccAction.Results); err != nil {
			requestContext.Error = errors.WithMessage(err, "Error unmarshaling to txRWSet")
			return
		}
		logger.Debugf("2.c.ShouldCommit=%b", c.ShouldCommit)
		if c.hasWriteSet(txRWSet, txID) {
			logger.Debugf("[txID %s] Commit is necessary since commit type is [%s] and write set exists in proposal response", txID, api.CommitOnWrite)
			c.ShouldCommit = true
		}
	}

	logger.Debugf("c.ShouldCommit=%b", c.ShouldCommit)

	if c.ShouldCommit {
		c.next.Handle(requestContext, clientContext)
	} else {
		logger.Debugf("[txID %s] Commit is NOT necessary since commit type is [%s] and NO write set exists in proposal response", txID, api.CommitOnWrite)
	}
}

func (c *CheckForCommitHandler) hasWriteSet(txRWSet *rwsetutil.TxRwSet, txID string) bool {
	set, err := json.Marshal(*txRWSet)
	if err != nil {
		fmt.Println("Parasu: could not marshall the rwset.")
	}
	logger.Debugf("2.1.c.ShouldCommit=%b, rwSetLen=%d, txRWSet=%s", c.ShouldCommit, len(txRWSet.NsRwSets), string(set))
	for _, nsRWSet := range txRWSet.NsRwSets {
		if ignoreCC(c.rwSetIgnoreNameSpace, nsRWSet.NameSpace) {
			// Ignore this writeset
			logger.Debugf("[txID %s] Ignoring writes to [%s]", txID, nsRWSet.NameSpace)
			continue
		}
		if nsRWSet.KvRwSet != nil && len(nsRWSet.KvRwSet.Writes) > 0 {
			logger.Debugf("[txID %s] Found writes to CC [%s]. A commit will be required.", txID, nsRWSet.NameSpace)
			return true
		}

		for _, collRWSet := range nsRWSet.CollHashedRwSets {
			if ignoreCollection(c.rwSetIgnoreNameSpace, nsRWSet.NameSpace, collRWSet.CollectionName) {
				// Ignore this writeset
				logger.Debugf("[txID %s] Ignoring writes to private data collection [%s] in CC [%s]", txID, collRWSet.CollectionName, nsRWSet.NameSpace)
				continue
			}
			if collRWSet.HashedRwSet != nil && len(collRWSet.HashedRwSet.HashedWrites) > 0 {
				logger.Debugf("[txID %s] Found writes to private data collection [%s] in CC [%s]. A commit will be required.", txID, collRWSet.CollectionName, nsRWSet.NameSpace)
				return true
			}
		}
	}
	logger.Debugf("3.c.ShouldCommit=%b", c.ShouldCommit)
	return false
}

func ignoreCC(namespaces []api.Namespace, ccName string) bool {
	for _, ns := range namespaces {
		if ns.Name == ccName {
			// Ignore entire chaincode only if no collections specified
			return len(ns.Collections) == 0
		}
	}
	return false
}

func ignoreCollection(namespaces []api.Namespace, ccName, collName string) bool {
	for _, ns := range namespaces {
		if ns.Name == ccName && contains(ns.Collections, collName) {
			return true
		}
	}
	return false
}

func contains(namespaces []string, name string) bool {
	for _, ns := range namespaces {
		if ns == name {
			return true
		}
	}
	return false
}
