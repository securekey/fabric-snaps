/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
)

//MockSelectionService type used in testing
type MockSelectionService struct {
	TestPeer       api.PeerConfig
	TestEndorsers  []sdkApi.Peer
	InvalidChannel string
}

//GetEndorsersForChaincode return endorsers for chaincode
func (m *MockSelectionService) GetEndorsersForChaincode(channelID string,
	chaincodeIDs ...string) ([]sdkApi.Peer, error) {
	if channelID == m.InvalidChannel {
		return nil, fmt.Errorf("Invalid channel")
	}
	return m.TestEndorsers, nil
}

//GetPeerForEvents get peers for events
func (m *MockSelectionService) GetPeerForEvents(channelID string) (*api.PeerConfig, error) {
	if channelID == m.InvalidChannel {
		return &api.PeerConfig{}, fmt.Errorf("Invalid channel")
	}
	return &m.TestPeer, nil
}
