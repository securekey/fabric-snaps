/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"time"

	sdkConfigApi "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	common "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/core/common/ccprovider"
)

// EndorseTxRequest contains the parameters for the EndoreseTransaction function
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
}

// Client is a wrapper interface around the fabric client
// It enables multithreaded access to the client
type Client interface {
	// NewChannel registers a channel object with the fabric client
	// this object represents a channel on the fabric network
	// @param {string} name of the channel
	// @returns {Channel} channel object
	// @returns {error} error, if any
	NewChannel(string) (sdkApi.Channel, error)

	// GetChannel returns a channel object that has been added to the fabric client
	// @param {string} name of the channel
	// @returns {Channel} channel that was requested
	// @returns {error} error, if any
	GetChannel(string) (sdkApi.Channel, error)

	// EndorseTransaction request endorsement from the peers on this channel
	// for a transaction with the given parameters
	// @param {Channel} channel on which we want to transact
	// @param {EndorseTxRequest} reuest identifies the chaincode to invoke
	// @returns {[]TransactionProposalResponse} responses from endorsers
	// @returns {error} error, if any
	EndorseTransaction(channel sdkApi.Channel, request *EndorseTxRequest) ([]*apitxn.TransactionProposalResponse, error)

	// CommitTransaction submits the given endorsements on the specified channel for
	// commit
	// @param {Channel} channel on which the transaction is taking place
	// @param {[]TransactionProposalResponse} responses from endorsers
	// @param {bool} register for Tx event
	// @param {time.Duration} register for Tx event timeout in seconds
	// @returns {error} error, if any
	CommitTransaction(sdkApi.Channel, []*apitxn.TransactionProposalResponse, bool, time.Duration) error

	// QueryChannels joined by the given peer
	// @param {Peer} The peer to query
	// @returns {[]string} list of channels
	// @returns {error} error, if any
	QueryChannels(sdkApi.Peer) ([]string, error)

	// VerifyTxnProposalSignature verify TxnProposalSignature against msp
	// @param {Channel} channel on which the transaction is taking place
	// @param {[]byte} Txn Proposal
	// @returns {error} error, if any
	VerifyTxnProposalSignature(sdkApi.Channel, []byte) error

	// SetSelectionService is used to inject a selection service for testing
	// @param {SelectionService} SelectionService
	SetSelectionService(SelectionService)

	// GetSelectionService returns the SelectionService
	GetSelectionService() SelectionService

	// GetEventHub returns the GetEventHub
	// @returns {EventHub} EventHub
	// @returns {error} error, if any
	GetEventHub() (sdkApi.EventHub, error)

	// Hash message
	// @param {[]byte} message to hash
	// @returns {[[]byte} hash
	// @returns {error} error, if any
	Hash([]byte) ([]byte, error)

	// InitializeChannel initializes the given channel
	// @param {Channel} Channel that needs to be initialized
	// @returns {error} error, if any
	InitializeChannel(channel sdkApi.Channel) error

	// GetConfig get client config
	// @returns {Config} config
	GetConfig() sdkConfigApi.Config

	// GetUser returns the user from the client context
	// @retruns {User} user
	GetUser() sdkApi.User
}

// CCDataProvider retrieves Chaincode Data for the given chaincode ID on the given channel
type CCDataProvider interface {
	QueryChaincodeData(channelID string, chaincodeID string) (*ccprovider.ChaincodeData, error)
}

// SelectionService selects peers for endorsement and commit events
type SelectionService interface {
	// GetEndorsersForChaincode returns a set of peers that should satisfy the endorsement
	// policies of all of the given chaincodes
	GetEndorsersForChaincode(channelID string, peerFilter PeerFilter, chaincodeIDs ...string) ([]sdkApi.Peer, error)
	GetPeerForEvents(channelID string) (*PeerConfig, error)
}

// SignaturePolicyFunc is a function that evaluates a signature policy and returns a peer group hierarchy
type SignaturePolicyFunc func() (GroupOfGroups, error)

// SignaturePolicyCompiler compiles a signature policy envelope and returns a peer group hierarchy
type SignaturePolicyCompiler interface {
	Compile(sigPolicyEnv *common.SignaturePolicyEnvelope) (GroupOfGroups, error)
}

// PeerRetriever is a function that retuens a set of peers for the given MSP ID
type PeerRetriever func(mspID string) []sdkApi.Peer

// PeerGroupResolver resolves a group of peers that would (exactly) satisfy
// a chaincode's endorsement policy.
type PeerGroupResolver interface {
	// Resolve returns a PeerGroup ensuring that all of the peers in the group are
	// in the given set of available peers
	// This method should never return nil but may return a PeerGroup that contains no peers.
	Resolve(peerFilter PeerFilter) PeerGroup
}

// LoadBalancePolicy is used to pick a peer group from a given set of peer groups
type LoadBalancePolicy interface {
	// Choose returns one of the peer groups from the given set of peer groups.
	// This method should never return nil but may return a PeerGroup that contains no peers.
	Choose(peerGroups []PeerGroup) PeerGroup
}

// Item represents any item
type Item interface {
}

// Group contains a group of Items
type Group interface {
	// Items returns all of the items
	Items() []Item

	// Equals returns true if this Group contains the same items as the given Group
	Equals(other Group) bool

	// Reduce reduces the group (which may be a hierarchy of groups) into a simple, non-hierarchical set of groups.
	// For example, given the group, G=(A and (B or C or D))
	// then G.Reduce() = [(A and B) or (A and C) or (A and D)]
	Reduce() []Group
}

// GroupOfGroups contains a set of groups.
type GroupOfGroups interface {
	// GroupOfGroups is also a Group
	Group

	// Groups returns all of the groups in this container
	Groups() []Group

	// Nof returns a set of groups that includes all possible combinations for the given threshold.
	// For example, given the group-of-groups, G=(G1, G2, G3), where G1=(A or B), G2=(C or D), G3=(E or F),
	// then:
	// - G.Nof(1) = (G1 or G2 or G3)
	// - G.Nof(2) = ((G1 and G2) or (G1 and G3) or (G2 and G3)
	// - G.Nof(3) = (G1 and G2 and G3)
	Nof(threshold int32) (GroupOfGroups, error)
}

// PeerGroup contains a group of Peers
type PeerGroup interface {
	Group
	Peers() []sdkApi.Peer
}

// Collapsable is implemented by any group that can collapse into a simple (non-hierarchical) Group
type Collapsable interface {
	// Collapse converts a hierarchical group into a single-level group (if possible).
	// For example, say G = (A and (B and C) and (D and E) and (F or G))
	// then G.Collapse() = (A and B and C and D and E and (F or G))
	Collapse() Group
}

// ChannelPeer extends Peer and adds channel-specific information
type ChannelPeer interface {
	sdkApi.Peer

	// ChannelID returns the channel ID
	ChannelID() string

	// BlockHeight returns the block height of the peer
	// for the current channel.
	BlockHeight() uint64

	// GetBlockHeight returns the block height of the peer for
	// the given channel. Returns 0 if the peer is not joined
	// to the channel or if the info is not available.
	GetBlockHeight(channelID string) uint64
}

// ChannelMembership defines membership for a channel
type ChannelMembership struct {
	// Peers on the channel
	Peers []ChannelPeer
	// QueryError Error from the last query/polling operation
	QueryError error
}

// MembershipManager maintains a peer membership lists on channels
type MembershipManager interface {
	// GetPeersOfChannel returns the peers on the given channel. It returns
	// ChannelMembership.QueryError is there was an error querying or polling
	// peers on the channel. It also returns the last known membership list
	// in case there was a polling error
	// @param {string} name of the channel
	// @returns {ChannelMembership} channel membership object
	GetPeersOfChannel(string) ChannelMembership
}
