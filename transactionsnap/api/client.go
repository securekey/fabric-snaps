/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/securekey/fabric-snaps/util/errors"
)

// EndorsedCallback is a function that is invoked after the endorsement
// phase of CommitTransaction. (Used in unit tests.)
type EndorsedCallback func(invoke.Response) error

// CommitType specifies how commits should be handled
type CommitType int

const (
	// CommitOnWrite indicates that the transaction should be committed only if
	// the consumer chaincode produces a write-set
	CommitOnWrite CommitType = iota

	// Commit indicates that the transaction should be committed
	Commit

	// NoCommit indicates that the transaction should not be committed
	NoCommit
)

// String returns the string value of CommitType
func (ct CommitType) String() string {
	switch ct {
	case CommitOnWrite:
		return "commitOnWrite"
	case Commit:
		return "commit"
	case NoCommit:
		return "noCommit"
	default:
		return "unknown"
	}
}

// EndorseTxRequest contains the parameters for the EndorseTransaction function
type EndorseTxRequest struct {
	// ChaincodeID identifies the chaincode to invoke
	ChaincodeID string
	// Args to pass to the chaincode. Args[0] is the function name
	Args []string
	// TransientData map (optional)
	TransientData map[string][]byte
	// Targets for the transaction (optional)
	Targets []fabApi.Peer
	// ChaincodeIDs contains all of the chaincodes that should be included
	// when evaluating endorsement policy (including the chaincode being invoked).
	// If empty then only the invoked chaincode is included. (optional)
	ChaincodeIDs []string
	// PeerFilter filters out peers using application-specific logic (optional)
	PeerFilter PeerFilter
	// CommitType specifies how commits should be handled (default CommitOnWrite)
	CommitType CommitType
	// RWSetIgnoreNameSpace rw set ignore list
	RWSetIgnoreNameSpace []Namespace
	//TransactionID txn id
	TransactionID string
	//Nonce nonce
	Nonce []byte
}

// Client is a wrapper interface around the fabric client
// It enables multithreaded access to the client
type Client interface {
	// EndorseTransaction request endorsement from the peers on this channel
	// for a transaction with the given parameters
	// @param {EndorseTxRequest} request identifies the chaincode to invoke
	// @returns {Response} responses from endorsers
	// @returns {error} error, if any
	EndorseTransaction(endorseRequest *EndorseTxRequest) (*channel.Response, errors.Error)

	// CommitTransaction request commit from the peers on this channel
	// for a transaction with the given parameters
	// @param {EndorseTxRequest} request identifies the chaincode to invoke
	// @param {registerTxEvent} is bool to register tx event
	// @param {EndorsedCallback} is a function that is invoked after the endorsement
	// @returns {Response} responses from endorsers
	// @returns {bool} commit flag
	// @returns {error} error, if any
	CommitTransaction(endorseRequest *EndorseTxRequest, registerTxEvent bool, callback EndorsedCallback) (*channel.Response, bool, errors.Error)

	// VerifyTxnProposalSignature verify TxnProposalSignature against msp
	// @param {[]byte} Txn Proposal
	// @returns {error} error, if any
	VerifyTxnProposalSignature([]byte) errors.Error

	// GetLocalPeer gets the local fab api peer
	// @returns {fabApi.Peer} fab api peer
	GetLocalPeer() (fabApi.Peer, error)

	// ChannelConfig returns the channel configuration
	ChannelConfig() (fabApi.ChannelCfg, error)

	// EventService returns the event service
	EventService() (fabApi.EventService, error)

	// GetDiscoveredPeer returns the peer from the Discovery service that matches the given URL
	// Returns error if no matching peer is found
	GetDiscoveredPeer(url string) (fabApi.Peer, error)
}
