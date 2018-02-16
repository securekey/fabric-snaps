/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

//SnapTransactionRequest type will be passed as argument to a transaction snap
//ChannelID and ChaincodeID are mandatory fields
type SnapTransactionRequest struct {
	ChannelID           string            // required channel ID
	ChaincodeID         string            // required chaincode ID
	TransientMap        map[string][]byte // optional transient Map
	EndorserArgs        [][]byte          // optional args for endorsement
	CCIDsForEndorsement []string          // optional ccIDs For endorsement selection
	PeerFilter          *PeerFilterOpts   // optional peer filter
}

// ClientService interface
type ClientService interface {
	GetFabricClient(channelID string, config Config) (Client, error)
}
