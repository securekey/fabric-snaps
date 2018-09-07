/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/sorter/blockheightsorter"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	selectopts "github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/options"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("txnsnap")

var peerSorter = blockheightsorter.New() // TODO: Configurable options

//NewPeerFilterHandler returns a handler that filter peers
func NewPeerFilterHandler(chaincodeIDs []string, config api.Config, next ...invoke.Handler) *PeerFilterHandler {
	return &PeerFilterHandler{chaincodeIDs: chaincodeIDs, config: config, next: getNext(next)}
}

//PeerFilterHandler for handling peers filter
type PeerFilterHandler struct {
	next         invoke.Handler
	chaincodeIDs []string
	config       api.Config
}

//Handle selects proposal processors
func (p *PeerFilterHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	//Get proposal processor, if not supplied then use selection service to get available peers as endorser
	if len(requestContext.Opts.Targets) == 0 {
		remainingAttempts := p.config.GetEndorserSelectionMaxAttempts()
		logger.Debugf("Attempting to get endorsers - [%d] attempts...", remainingAttempts)
		var endorsers []fabApi.Peer
		for len(endorsers) == 0 && remainingAttempts > 0 {
			endorsers, err := p.getEndorsers(requestContext, clientContext)
			if err != nil {
				requestContext.Error = errors.WithMessage(err, "Failed to get endorsing peers")
				return
			}
			if len(endorsers) == 0 {
				remainingAttempts--
				logger.Warnf("No endorsers. [%d] remaining attempts...", remainingAttempts)
				time.Sleep(p.config.GetEndorserSelectionInterval())
			}
		}
		requestContext.Opts.Targets = endorsers
	}

	//Delegate to next step if any
	if p.next != nil {
		p.next.Handle(requestContext, clientContext)
	}
}

func (p *PeerFilterHandler) getEndorsers(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) ([]fabApi.Peer, error) {
	var selectionOpts []options.Opt
	if requestContext.SelectionFilter != nil {
		selectionOpts = append(selectionOpts, selectopts.WithPeerFilter(requestContext.SelectionFilter))
	}
	if requestContext.PeerSorter != nil {
		selectionOpts = append(selectionOpts, selectopts.WithPeerSorter(requestContext.PeerSorter))
	} else {
		logger.Debugf("Using block height sorter")
		selectionOpts = append(selectionOpts, selectopts.WithPeerSorter(peerSorter))
	}
	if len(p.chaincodeIDs) == 0 {
		p.chaincodeIDs = make([]string, 1)
		p.chaincodeIDs[0] = requestContext.Request.ChaincodeID
	}

	ccCalls := make([]*fabApi.ChaincodeCall, len(p.chaincodeIDs))
	for i, cid := range p.chaincodeIDs {
		ccCalls[i] = &fabApi.ChaincodeCall{ID: cid}
	}
	return clientContext.Selection.GetEndorsersForChaincode(ccCalls, selectionOpts...)
}

// filterTargets is helper method to filter peers
func (p *PeerFilterHandler) filterTargets(peers []fabApi.Peer, filter api.PeerFilter) []fabApi.Peer {

	if filter == nil {
		return peers
	}

	filteredPeers := []fabApi.Peer{}
	for _, peer := range peers {
		if filter.Accept(peer) {
			filteredPeers = append(filteredPeers, peer)
		}
	}

	return filteredPeers
}

func getNext(next []invoke.Handler) invoke.Handler {
	if len(next) > 0 {
		return next[0]
	}
	return nil
}
