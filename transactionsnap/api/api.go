/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

// Namespace contains a chaincode name and an optional set of private data collections to ignore
type Namespace struct {
	Name        string
	Collections []string
}

//SnapTransactionRequest type will be passed as argument to a transaction snap
//ChannelID and ChaincodeID are mandatory fields
type SnapTransactionRequest struct {
	ChannelID            string            // required channel ID
	ChaincodeID          string            // required chaincode ID
	TransientMap         map[string][]byte // optional transient Map
	EndorserArgs         [][]byte          // optional args for endorsement
	CCIDsForEndorsement  []string          // optional ccIDs For endorsement selection
	RegisterTxEvent      bool              // optional args for register Tx event (default is false)
	PeerFilter           *PeerFilterOpts   // optional peer filter
	CommitType           CommitType        // optional specifies how commits should be handled (default CommitOnWrite)
	RWSetIgnoreNameSpace []Namespace       // RWSetIgnoreNameSpace rw set ignore list
	TransactionID        string            // TransactionID txn id
	Nonce                []byte            // Nonce nonce

}

// Creator is received from the delegate when its identity
// doesn't match the TxID pre-calculated by this handler.
// It is received as JSON in the proposal response payload.
type Creator struct {
	// base64 encoded identity of the delegate
	Identity string `json:"identity,omitempty"`
}

// ClientService interface
type ClientService interface {
	GetFabricClient(channelID string) (Client, error)
}

// ValidationRequest holds the info for the validation request
type ValidationRequest struct {
	ChannelID         string `json:"channelID"`
	Proposal          []byte `json:"proposal"`
	ProposalResponses []byte `json:"proposalResponses"`
}
