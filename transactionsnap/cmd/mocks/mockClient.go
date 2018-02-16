/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"time"
)

import (
	sdkConfigApi "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
)

//GetNewClientWrapper returns wrapper mock object of client
func GetNewClientWrapper(fcClient transactionsnapApi.Client) *MockClient {
	return &MockClient{fcClient: fcClient}
}

//MockClient wrapper for client.Client which can be manipulated for desired results for tests
type MockClient struct {
	fcClient transactionsnapApi.Client
}

// EndorseTransaction request endorsement from the peers on this channel
// for a transaction with the given parameters
// @param {Channel} channel on which we want to transact
// @param {EndorseTxRequest} request identifies the chaincode to invoke
// @returns {[]TransactionProposalResponse} responses from endorsers
// @returns {error} error, if any
func (c *MockClient) EndorseTransaction(request *transactionsnapApi.EndorseTxRequest) (
	[]byte, error) {
	return c.fcClient.EndorseTransaction(request)
}

// CommitTransaction submits the given endorsements on the specified channel for
// commit
// @param {Channel} channel on which the transaction is taking place
// @param {[]TransactionProposalResponse} responses from endorsers
// @param {bool} register for Tx event
// @returns {error} error, if any
func (c *MockClient) CommitTransaction(request *transactionsnapApi.EndorseTxRequest) error {
	return c.fcClient.CommitTransaction(request, 1*time.Minute)
}

// QueryChannels joined by the given peer
// @param {Peer} The peer to query
// @returns {[]string} list of channels
// @returns {error} error, if any
func (c *MockClient) QueryChannels(peer sdkApi.Peer) ([]string, error) {
	return c.fcClient.QueryChannels(peer)
}

// VerifyTxnProposalSignature verify TxnProposalSignature against msp
// @param {Channel} channel on which the transaction is taking place
// @param {[]byte} Txn Proposal
// @returns {error} error, if any
func (c *MockClient) VerifyTxnProposalSignature(bytes []byte) error {
	return c.fcClient.VerifyTxnProposalSignature(bytes)
}

// GetConfig get client config
// @returns {Config} config
func (c *MockClient) GetConfig() sdkConfigApi.Config {
	return c.fcClient.GetConfig()
}

// GetSigningIdentity returns the signingIdentity (user) context from the client
// @retruns {sdkApi.IdentityContext} sdkApi.IdentityContext
func (c *MockClient) GetSigningIdentity() sdkApi.IdentityContext {
	return c.fcClient.GetSigningIdentity()
}
