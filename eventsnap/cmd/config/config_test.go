/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"
	"time"

	"os"

	"github.com/hyperledger/fabric/bccsp/factory"
	configmocks "github.com/securekey/fabric-snaps/configmanager/pkg/mocks"
	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
)

const (
	TxnSnapAppName = "txnsnap"
	channelID      = "testChannel"
	mspID          = "Org1MSP"
	peerID         = "peer1"
)

func TestInvalidConfig(t *testing.T) {
	_, err := New("", "./invalid")
	if err == nil {
		t.Fatalf("Expecting error for invalid config but received none")
	}
}

func TestConfig(t *testing.T) {

	configStub1 := configmocks.NewMockStub(channelID)
	service.Initialize(configStub1, mspID)

	// Test with no channel config
	config, err := New("", "../sampleconfig")
	if err == nil {
		t.Fatalf("Expecting error creating new config with no channel")
	}

	// Test config on channel1
	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, EventSnapAppName, "../sampleconfig/config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, TxnSnapAppName, "../sampleconfig/txnsnap-config.yaml"); err != nil {
		t.Fatalf("Error saving config: %s", err)
	}
	config, err = New(channelID, "../sampleconfig")
	if err != nil {
		t.Fatalf("Error creating new config: %s", err)
	}
	checkUint(t, "EventConsumerBufferSize", config.EventConsumerBufferSize, 101)
	checkUint(t, "EventDispatcherBufferSize", config.EventDispatcherBufferSize, 101)
	checkDuration(t, "EventConsumerTimeout", config.EventConsumerTimeout, 11*time.Millisecond)
	checkDuration(t, "EventConsumerTimeout", config.ResponseTimeout, 3*time.Second)
	checkString(t, "EventDispatcherBufferSize", config.URL, "0.0.0.0:7051")

	if len(config.Bytes) == 0 {
		t.Fatal("config bytes are not supposed to be empty")
	}

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

func TestMain(m *testing.M) {

	txsnapservice.PeerConfigPath = "../sampleconfig"

	opts := &factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily:   "SHA2",
			SecLevel:     256,
			Ephemeral:    false,
			FileKeystore: &factory.FileKeystoreOpts{KeyStorePath: "../sampleconfig/msp/keystore"},
		},
	}
	factory.InitFactories(opts)

	os.Exit(m.Run())
}
