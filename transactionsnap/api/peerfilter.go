/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"

// PeerFilterType is the type name of the Peer Filter
type PeerFilterType string

const (
	// MinBlockHeightPeerFilterType is a peer filter that selects peers
	// whose block height is at least the height of the local peer on which
	// the TxSnap is being invoked.
	// Required Args:
	// - arg[0]: Channel ID
	MinBlockHeightPeerFilterType PeerFilterType = "MinBlockHeight"
)

// PeerFilter is applied to peers selected for endorsement and removes
// those groups that don't pass the filter acceptance test
type PeerFilter interface {
	// Accept returns true if the given peer should be included in the set of endorsers
	Accept(peer sdkApi.Peer) bool
}

// PeerFilterOpts specifies the peer filter type and
// includes any args required by the peer filter
type PeerFilterOpts struct {
	Type PeerFilterType
	Args []string
}
