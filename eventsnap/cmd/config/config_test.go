/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"
	"time"

	configmocks "github.com/securekey/fabric-snaps/configmanager/pkg/mocks"
	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
)

func TestInvalidConfig(t *testing.T) {
	_, err := New("", "./invalid")
	if err == nil {
		t.Fatalf("Expecting error for invalid config but received none")
	}
}

func TestConfigNoChannel(t *testing.T) {
	config, err := New("", "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}

	checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
	checkUint(t, "EventServerBufferSize", config.EventServerBufferSize, 100)
	checkDuration(t, "EventServerTimeout", config.EventServerTimeout, 10*time.Millisecond)
	checkDuration(t, "EventServerTimeWindow", config.EventServerTimeWindow, 15*time.Minute)
}

func TestConfig(t *testing.T) {
	mspID := "Org1MSP"
	peerID := "peer1"
	channelID1 := "ch1"
	channelID2 := "ch2"

	configStub1 := configmocks.NewMockStub(channelID1)
	service.Initialize(configStub1, mspID)

	// Test default values
	config, err := New(channelID1, "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}
	checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
	checkUint(t, "EventConsumerBufferSize", config.EventConsumerBufferSize, defaultEventConsumerBufferSize)
	checkDuration(t, "EventHubRegTimeout", config.EventHubRegTimeout, defaultEventHubRegTimeout)
	checkDuration(t, "EventRelayTimeout", config.EventRelayTimeout, defaultEventRelayTimeout)
	checkUint(t, "EventDispatcherBufferSize", config.EventDispatcherBufferSize, defaultEventDispatcherBufferSize)
	checkDuration(t, "EventConsumerTimeout", config.EventConsumerTimeout, defaultEventConsumerTimeout)
	checkUint(t, "EventServerBufferSize", config.EventServerBufferSize, 100)
	checkDuration(t, "EventServerTimeout", config.EventServerTimeout, 10*time.Millisecond)
	checkDuration(t, "EventServerTimeWindow", config.EventServerTimeWindow, 15*time.Minute)

	// Test config on channel1
	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, EventSnapAppName, "../sampleconfig/configch1.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	config, err = New(channelID1, "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}
	checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
	checkUint(t, "EventConsumerBufferSize", config.EventConsumerBufferSize, 100)
	checkDuration(t, "EventHubRegTimeout", config.EventHubRegTimeout, 1*time.Second)
	checkDuration(t, "EventRelayTimeout", config.EventRelayTimeout, 1*time.Second)
	checkUint(t, "EventDispatcherBufferSize", config.EventDispatcherBufferSize, 100)
	checkDuration(t, "EventConsumerTimeout", config.EventConsumerTimeout, 10*time.Millisecond)
	checkUint(t, "EventServerBufferSize", config.EventServerBufferSize, 100)
	checkDuration(t, "EventServerTimeout", config.EventServerTimeout, 10*time.Millisecond)
	checkDuration(t, "EventServerTimeWindow", config.EventServerTimeWindow, 15*time.Minute)

	// Test config on channel2
	configStub2 := configmocks.NewMockStub(channelID2)
	if err := configmocks.SaveConfigFromFile(configStub2, mspID, peerID, EventSnapAppName, "../sampleconfig/configch2.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	config, err = New(channelID2, "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}
	checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
	checkUint(t, "EventConsumerBufferSize", config.EventConsumerBufferSize, 200)
	checkDuration(t, "EventHubRegTimeout", config.EventHubRegTimeout, 2*time.Second)
	checkDuration(t, "EventRelayTimeout", config.EventRelayTimeout, 2*time.Second)
	checkUint(t, "EventDispatcherBufferSize", config.EventDispatcherBufferSize, 200)
	checkDuration(t, "EventConsumerTimeout", config.EventConsumerTimeout, 20*time.Millisecond)
	checkUint(t, "EventServerBufferSize", config.EventServerBufferSize, 100)
	checkDuration(t, "EventServerTimeout", config.EventServerTimeout, 10*time.Millisecond)
	checkDuration(t, "EventServerTimeWindow", config.EventServerTimeWindow, 15*time.Minute)
}

func checkString(t *testing.T, field string, value string, expectedValue string) {
	if value != expectedValue {
		t.Fatalf("Expecting [%s] for [%s] but got [%s]", expectedValue, field, value)
	}
}

func checkUint(t *testing.T, field string, value, expectedValue uint) {
	if value != expectedValue {
		t.Fatalf("Expecting [%d] for [%s] but got [%d]", expectedValue, field, value)
	}
}

func checkDuration(t *testing.T, field string, value, expectedValue time.Duration) {
	if value != expectedValue {
		t.Fatalf("Expecting %d for %s but got %d", expectedValue, field, value)
	}
}
