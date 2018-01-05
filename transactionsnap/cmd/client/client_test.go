/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"testing"

	"github.com/securekey/fabric-snaps/transactionsnap/cmd/config"
)

func TestGetInstance(t *testing.T) {
	channelID := "testChannel"
	//create config
	config, err := config.NewConfig("../sampleconfig", channelID)
	if err != nil {
		t.Fatalf("Error initializing config: %s", err)
	}
	//Get instance of client - this will create cache
	_, err = GetInstance("", &sampleConfig{config})
	if err == nil {
		t.Fatalf("Expected error: 'Channel is required'")
	}
	//Get instance of client - client from cache
	_, err = GetInstance(channelID, &sampleConfig{config})
	if err != nil {
		t.Fatalf("Client GetInstance return error %v", err)
	}
	//Another channel - this will add to existing cache
	_, err = GetInstance("testTwo", &sampleConfig{config})
	if err != nil {
		t.Fatalf("Client GetInstance return error %v", err)
	}
	//read from cache
	clientMutex.RLock()
	c := cachedClient[channelID]
	c1 := cachedClient["doesnotexist"]
	c2 := cachedClient[""]
	c3 := cachedClient["testTwo"]
	clientMutex.RUnlock()
	if c == nil {
		t.Fatalf("Client for channel %s should have been cached ", channelID)
	}
	if c3 == nil {
		t.Fatalf("Client for channel %s should have been cached ", channelID)
	}
	if c1 != nil {
		t.Fatalf("Client for channel %s should NOT be in cache ", channelID)
	}
	if c2 != nil {
		t.Fatalf("Client for channel %s should NOT be in cache ", channelID)
	}

}
