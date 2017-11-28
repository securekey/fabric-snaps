/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localservice

import (
	"testing"

	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

func TestLocalService(t *testing.T) {
	channelID1 := "ch1"
	channelID2 := "ch2"

	service1 := newMockEventService()
	service2 := newMockEventService()

	if err := Register(channelID1, service1); err != nil {
		t.Fatalf("error registering local event service for channel %s: %s", channelID1, err)
	}

	if err := Register(channelID2, service2); err != nil {
		t.Fatalf("error registering local event service for channel %s: %s", channelID2, err)
	}

	// Register twice
	if err := Register(channelID2, service2); err == nil {
		t.Fatalf("expecting error registering local event service twice for channel %s but got none", channelID2)
	}

	if s := Get(channelID1); s != service1 {
		t.Fatalf("invalid service retrieved for channel")
	}
	if s := Get(channelID2); s != service2 {
		t.Fatalf("invalid service retrieved for channel")
	}
	if s := Get("invalidchannel"); s != nil {
		t.Fatalf("expecting nil service for invalid channel")
	}
}

func newMockEventService() *mockService {
	return &mockService{}
}

type mockService struct {
}

func (m *mockService) RegisterFilteredBlockEvent() (eventapi.Registration, <-chan *eventapi.FilteredBlockEvent, error) {
	panic("not implemented")
}

func (m *mockService) RegisterChaincodeEvent(ccID, eventFilter string) (eventapi.Registration, <-chan *eventapi.CCEvent, error) {
	panic("not implemented")
}

func (m *mockService) RegisterTxStatusEvent(txID string) (eventapi.Registration, <-chan *eventapi.TxStatusEvent, error) {
	panic("not implemented")
}

func (m *mockService) Unregister(reg eventapi.Registration) {
	panic("not implemented")
}

func (m *mockService) RegisterBlockEvent() (eventapi.Registration, <-chan *eventapi.BlockEvent, error) {
	panic("not implemented")
}
