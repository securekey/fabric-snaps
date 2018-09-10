/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package balancer

import (
	"strings"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client/lbp"
	"github.com/pkg/errors"
)

var logger = logging.NewLogger("txnsnap")

// PreferPeerBalancer is a balancer that chooses the provided peer if it's in
// the provided list of peers, otherwise it load balances the peers.
type PreferPeerBalancer struct {
	lbp.LoadBalancePolicy
	peerURL string
}

// NewPreferPeer creates a balancer that chooses the provided peer if it's in
// the provided list of peers, otherwise it load balances the peers.
func NewPreferPeer(peerURL string, balancer lbp.LoadBalancePolicy) *PreferPeerBalancer {
	return &PreferPeerBalancer{
		LoadBalancePolicy: balancer,
		peerURL:           peerURL,
	}
}

// Choose chooses from the list of peers using the provided balancer but prefers the local peer if it's in the list
func (b *PreferPeerBalancer) Choose(peers []fab.Peer) (fab.Peer, error) {
	if len(peers) == 0 {
		return nil, errors.New("no peers to choose from")
	}

	for _, peer := range peers {
		logger.Debugf("Checking if the peer [%s] is the preferred peer [%s] ...", peer.URL(), b.peerURL)
		if strings.Contains(peer.URL(), b.peerURL) {
			logger.Debugf("... choosing preferred peer [%s]", b.peerURL)
			return peer, nil
		}
	}

	logger.Debugf("Preferred peer [%s] is not in the list of peers. Choosing from the list using the provided balancer.", b.peerURL)
	return b.LoadBalancePolicy.Choose(peers)
}
