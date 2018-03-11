/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package handler

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

var logger = logging.NewLogger("txnsnap")

//NewPeerFilterHandler returns a handler that filter peers
func NewPeerFilterHandler(peerFilter api.PeerFilter, chaincodeIDs []string, config api.Config, next ...invoke.Handler) *PeerFilterHandler {
	return &PeerFilterHandler{peerFilter: peerFilter, chaincodeIDs: chaincodeIDs, config: config, next: getNext(next)}
}

//PeerFilterHandler for handling peers filter
type PeerFilterHandler struct {
	next         invoke.Handler
	peerFilter   api.PeerFilter
	chaincodeIDs []string
	config       api.Config
}

//Handle for endorsing transactions
func (p *PeerFilterHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {
	//Get proposal processor, if not supplied then use discovery service to get available peers as endorser
	//If selection service available then get endorser peers for this chaincode
	if len(requestContext.Opts.Targets) == 0 {
		// Select endorsers
		remainingAttempts := p.config.GetEndorserSelectionMaxAttempts()
		logger.Infof("Attempting to get endorsers - [%d] attempts...", remainingAttempts)
		var endorsers []fabApi.Peer
		for len(endorsers) == 0 && remainingAttempts > 0 {
			var err error
			// Use discovery service to figure out proposal processors
			peersFromDiscovery, err := clientContext.Discovery.GetPeers()
			if err != nil {
				requestContext.Error = errors.WithMessage(err, "GetPeers failed")
				return
			}
			logger.Debugf("Discovery.GetPeers() return peers:%v", peersFromDiscovery)
			filterPeers := p.filterTargets(peersFromDiscovery, p.peerFilter)
			logger.Debugf("filterTargets return peers:%v", filterPeers)
			endorsers = filterPeers
			if clientContext.Selection != nil && len(endorsers) != 0 {
				if len(p.chaincodeIDs) == 0 {
					p.chaincodeIDs = make([]string, 1)
					p.chaincodeIDs[0] = requestContext.Request.ChaincodeID
				}
				endorsers, err = clientContext.Selection.GetEndorsersForChaincode(filterPeers, p.chaincodeIDs...)
				if err != nil {
					requestContext.Error = errors.WithMessage(err, "Failed to get endorsing peers")
					return
				}
				logger.Debugf("Selection GetEndorsersForChaincode return peers:%v", endorsers)

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
