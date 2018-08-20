/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bcinfo

import (
	"fmt"
	"sync"
	"testing"
	"time"

	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/stretchr/testify/require"
)

type mockLedger struct {
	channelID    string
	bcinfoByChan map[string]*cb.BlockchainInfo
	mutex        *sync.RWMutex
}

func (l *mockLedger) GetBlockchainInfo() (*cb.BlockchainInfo, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	bcInfo, ok := l.bcinfoByChan[l.channelID]
	if !ok || bcInfo == nil {
		return nil, fmt.Errorf("blockchain info not found for channel [%s]", l.channelID)
	}
	return bcInfo, nil
}

func TestBCINFOProvider(t *testing.T) {
	ch1 := "channel1"
	ch2 := "channel2"
	ch3 := "channel3"

	bcInfo1 := &cb.BlockchainInfo{Height: 100}
	bcInfo1_1 := &cb.BlockchainInfo{Height: 101}
	bcInfo2 := &cb.BlockchainInfo{Height: 200}

	var mutex sync.RWMutex
	bcinfoByChan := map[string]*cb.BlockchainInfo{
		ch1: bcInfo1, ch2: bcInfo2, ch3: nil,
	}

	// mock out the ledger provider
	ledgerPrvdr = func(channelID string) ledger {
		return &mockLedger{
			channelID:    channelID,
			bcinfoByChan: bcinfoByChan,
			mutex:        &mutex,
		}
	}

	bcinfoProvider := NewProvider()

	bcInfo, err := bcinfoProvider.GetBlockchainInfo(ch1)
	require.NoError(t, err)
	require.Equal(t, bcInfo1, bcInfo)

	bcInfo, err = bcinfoProvider.GetBlockchainInfo(ch2)
	require.NoError(t, err)
	require.Equal(t, bcInfo2, bcInfo)

	bcInfo, err = bcinfoProvider.GetBlockchainInfo(ch3)
	require.Error(t, err)

	mutex.Lock()
	bcinfoByChan[ch1] = bcInfo1_1
	// Update to nil will cause an error and the caller should still get the old value
	bcinfoByChan[ch2] = nil
	mutex.Unlock()

	time.Sleep(1500 * time.Millisecond)

	bcInfo, err = bcinfoProvider.GetBlockchainInfo(ch1)
	require.NoError(t, err)
	require.Equal(t, bcInfo1_1, bcInfo)

	bcInfo, err = bcinfoProvider.GetBlockchainInfo(ch2)
	require.NoError(t, err)
	require.Equal(t, bcInfo2, bcInfo)
}
