/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("txnsnap")

//NewPeerFilterHandler returns a handler that filter peers
func NewPeerFilterHandler(peerFilter api.PeerFilter, chaincodeIDs []string, config api.Config, next ...chclient.Handler) *PeerFilterHandler {
	return &PeerFilterHandler{peerFilter: peerFilter, chaincodeIDs: chaincodeIDs, config: config, next: getNext(next)}
}

//PeerFilterHandler for handling peers filter
type PeerFilterHandler struct {
	next         chclient.Handler
	peerFilter   api.PeerFilter
	chaincodeIDs []string
	config       api.Config
}

//Handle for endorsing transactions
func (p *PeerFilterHandler) Handle(requestContext *chclient.RequestContext, clientContext *chclient.ClientContext) {
	//Get proposal processor, if not supplied then use discovery service to get available peers as endorser
	//If selection service available then get endorser peers for this chaincode
	if len(requestContext.Opts.ProposalProcessors) == 0 {
		// Select endorsers
		remainingAttempts := p.config.GetEndorserSelectionMaxAttempts()
		logger.Infof("Attempting to get endorsers - [%d] attempts...", remainingAttempts)
		var peers []apifabclient.Peer
		for len(peers) == 0 && remainingAttempts > 0 {
			var err error
			// Use discovery service to figure out proposal processors
			peers, err = clientContext.Discovery.GetPeers()
			if err != nil {
				requestContext.Error = errors.WithMessage(err, "GetPeers failed")
				return
			}
			logger.Debugf("Discovery.GetPeers() return peers:%v", peers)
			if clientContext.Selection != nil {
				if len(p.chaincodeIDs) == 0 {
					p.chaincodeIDs = make([]string, 1)
					p.chaincodeIDs[0] = requestContext.Request.ChaincodeID
				}
				peers, err = clientContext.Selection.GetEndorsersForChaincode(peers, p.chaincodeIDs...)
				if err != nil {
					requestContext.Error = errors.WithMessage(err, "Failed to get endorsing peers")
					return
				}
				logger.Debugf("Selection GetEndorsersForChaincode return peers:%v", peers)

			}
			peers := p.filterTargets(peers, p.peerFilter)
			if len(peers) == 0 {
				remainingAttempts--
				logger.Warnf("No endorsers. [%d] remaining attempts...", remainingAttempts)
				time.Sleep(p.config.GetEndorserSelectionInterval())
			}
		}

		requestContext.Opts.ProposalProcessors = peer.PeersToTxnProcessors(peers)
	}

	//Delegate to next step if any
	if p.next != nil {
		p.next.Handle(requestContext, clientContext)
	}
}

// filterTargets is helper method to filter peers
func (p *PeerFilterHandler) filterTargets(peers []apifabclient.Peer, filter api.PeerFilter) []apifabclient.Peer {

	if filter == nil {
		return peers
	}

	filteredPeers := []apifabclient.Peer{}
	for _, peer := range peers {
		if filter.Accept(peer) {
			filteredPeers = append(filteredPeers, peer)
		}
	}
	logger.Debugf("filterTargets return peers:%v", filteredPeers)

	return filteredPeers
}

func getNext(next []chclient.Handler) chclient.Handler {
	if len(next) > 0 {
		return next[0]
	}
	return nil
}
