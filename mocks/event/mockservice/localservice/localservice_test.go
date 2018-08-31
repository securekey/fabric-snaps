/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localservice

import (
	"testing"

	fabmocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/mocks"
)

func TestLocalService(t *testing.T) {
	channelID1 := "ch1"
	channelID2 := "ch2"

	service1 := fabmocks.NewMockEventService()
	service2 := fabmocks.NewMockEventService()

	if err := Register(channelID1, service1); err != nil {
		t.Fatalf("error registering localservice eventservice service for channel %s: %s", channelID1, err)
	}

	if err := Register(channelID2, service2); err != nil {
		t.Fatalf("error registering localservice eventservice service for channel %s: %s", channelID2, err)
	}

	// Register twice
	if err := Register(channelID2, service2); err == nil {
		t.Fatalf("expecting error registering localservice eventservice service twice for channel %s but got none", channelID2)
	}

	if s := Get(channelID1); s != service1 {
		t.Fatal("invalid service retrieved for channel")
	}
	if s := Get(channelID2); s != service2 {
		t.Fatal("invalid service retrieved for channel")
	}
	if s := Get("invalidchannel"); s != nil {
		t.Fatal("expecting nil service for invalid channel")
	}
}
