/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	sdkConfigApi "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
)

// EndorsedCallback is a function that is invoked after the endorsement
// phase of CommitTransaction. (Used in unit tests.)
type EndorsedCallback func([]*sdkApi.TransactionProposalResponse) error

// EndorseTxRequest contains the parameters for the EndorseTransaction function
type EndorseTxRequest struct {
	// ChaincodeID identifies the chaincode to invoke
	ChaincodeID string
	// Args to pass to the chaincode. Args[0] is the function name
	Args []string
	// TransientData map (optional)
	TransientData map[string][]byte
	// Targets for the transaction (optional)
	Targets []sdkApi.Peer
	// ChaincodeIDs contains all of the chaincodes that should be included
	// when evaluating endorsement policy (including the chaincode being invoked).
	// If empty then only the invoked chaincode is included. (optional)
	ChaincodeIDs []string
	// PeerFilter filters out peers using application-specific logic (optional)
	PeerFilter PeerFilter
	// RWSetIgnoreNameSpace rw set ignore list
	RWSetIgnoreNameSpace []string
}

// Client is a wrapper interface around the fabric client
// It enables multithreaded access to the client
type Client interface {
	// EndorseTransaction request endorsement from the peers on this channel
	// for a transaction with the given parameters
	// @param {EndorseTxRequest} request identifies the chaincode to invoke
	// @returns {[]TransactionProposalResponse} responses from endorsers
	// @returns {error} error, if any
	EndorseTransaction(endorseRequest *EndorseTxRequest) ([]*sdkApi.TransactionProposalResponse, error)

	// CommitTransaction request commit from the peers on this channel
	// for a transaction with the given parameters
	// @param {EndorseTxRequest} request identifies the chaincode to invoke
	// @param {registerTxEvent} is bool to register tx event
	// @param {EndorsedCallback} is a function that is invoked after the endorsement
	// @returns {[]TransactionProposalResponse} responses from endorsers
	// @returns {error} error, if any
	CommitTransaction(endorseRequest *EndorseTxRequest, registerTxEvent bool, callback EndorsedCallback) ([]*sdkApi.TransactionProposalResponse, error)

	// QueryChannels joined by the given peer
	// @param {Peer} The peer to query
	// @returns {[]string} list of channels
	// @returns {error} error, if any
	QueryChannels(sdkApi.Peer) ([]string, error)

	// VerifyTxnProposalSignature verify TxnProposalSignature against msp
	// @param {[]byte} Txn Proposal
	// @returns {error} error, if any
	VerifyTxnProposalSignature([]byte) error

	// GetConfig get client config
	// @returns {Config} config
	GetConfig() sdkConfigApi.Config
}
