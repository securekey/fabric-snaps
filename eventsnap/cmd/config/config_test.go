/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"
	"time"
)

func TestInvalidConfig(t *testing.T) {
	_, err := New("./invalid")
	if err == nil {
		t.Fatalf("Expecting error for invalid config but received none")
	}
}

func TestConfig(t *testing.T) {
	config, err := New("../sampleconfig")
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
	checkDuration(t, "EventServerTimeWindow", config.EventServerTimeWindow, 15*time.Second)
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
