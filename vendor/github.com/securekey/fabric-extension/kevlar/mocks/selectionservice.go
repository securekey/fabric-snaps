/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package mocks

import (
	"fmt"

	sdkApi "github.com/hyperledger/fabric-sdk-go/api"
	"github.com/securekey/fabric-extension/kevlar/config"
)

type MockSelectionService struct {
	TestPeer       config.PeerConfig
	TestEndorsers  []sdkApi.Peer
	InvalidChannel string
}

func (m *MockSelectionService) GetEndorsersForChaincode(channelID string,
	chaincodeIDs ...string) ([]sdkApi.Peer, error) {
	if channelID == m.InvalidChannel {
		return nil, fmt.Errorf("Invalid channel")
	}
	return m.TestEndorsers, nil
}

func (m *MockSelectionService) GetPeerForEvents(channelID string) (*config.PeerConfig, error) {
	if channelID == m.InvalidChannel {
		return &config.PeerConfig{}, fmt.Errorf("Invalid channel")
	}
	return &m.TestPeer, nil
}
