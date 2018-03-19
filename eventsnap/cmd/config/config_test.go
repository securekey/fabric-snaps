/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

//import (
//	"fmt"
//	"io/ioutil"
//	"testing"
//	"time"
//
//	configmocks "github.com/securekey/fabric-snaps/configmanager/pkg/mocks"
//	"github.com/securekey/fabric-snaps/configmanager/pkg/service"
//	"github.com/spf13/viper"
//)
//
//func TestInvalidConfig(t *testing.T) {
//	_, err := New("", "./invalid")
//	if err == nil {
//		t.Fatalf("Expecting error for invalid config but received none")
//	}
//}
//
//func TestConfig(t *testing.T) {
//	mspID := "Org1MSP"
//	peerID := "peer1"
//	channelID1 := "ch1"
//	channelID2 := "ch2"
//	channelID3 := "ch3"
//
//	configStub1 := configmocks.NewMockStub(channelID1)
//	service.Initialize(configStub1, mspID)
//
//	// Test with no channel config
//	config, err := New("", "../sampleconfig")
//	if err == nil {
//		t.Fatalf("Expecting error creating new config with no channel")
//	}
//	// if config.ChannelConfigLoaded {
//	// 	t.Fatalf("Expecting that channel config is not loaded")
//	// }
//	// checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
//
//	// Test config on channel1
//	if err := configmocks.SaveConfigFromFile(configStub1, mspID, peerID, EventSnapAppName, "../sampleconfig/configch1.yaml"); err != nil {
//		t.Fatalf("Error saving config: %s", err)
//	}
//	config, err = New(channelID1, "../sampleconfig")
//	if err != nil {
//		t.Fatalf("Error creating new config: %s", err)
//	}
//	checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
//	checkUint(t, "EventConsumerBufferSize", config.EventConsumerBufferSize, 100)
//	checkDuration(t, "EventHubRegTimeout", config.EventHubRegTimeout, 1*time.Second)
//	checkUint(t, "EventDispatcherBufferSize", config.EventDispatcherBufferSize, 100)
//	checkDuration(t, "EventConsumerTimeout", config.EventConsumerTimeout, 10*time.Millisecond)
//
//	// Test config on channel2
//	configStub2 := configmocks.NewMockStub(channelID2)
//	if err := configmocks.SaveConfigFromFile(configStub2, mspID, peerID, EventSnapAppName, "../sampleconfig/configch2.yaml"); err != nil {
//		t.Fatalf("Error saving config: %s", err)
//	}
//	config, err = New(channelID2, "../sampleconfig")
//	if err != nil {
//		t.Fatalf("Error creating new config: %s", err)
//	}
//	checkString(t, "EventHubAddress", config.EventHubAddress, "0.0.0.0:7053")
//	checkUint(t, "EventConsumerBufferSize", config.EventConsumerBufferSize, 200)
//	checkDuration(t, "EventHubRegTimeout", config.EventHubRegTimeout, 2*time.Second)
//	checkUint(t, "EventDispatcherBufferSize", config.EventDispatcherBufferSize, 200)
//	checkDuration(t, "EventConsumerTimeout", config.EventConsumerTimeout, 20*time.Millisecond)
//
//	// Test config on channel3
//	configStub3 := configmocks.NewMockStub(channelID3)
//	if err := configmocks.SaveConfigFromFile(configStub3, mspID, peerID, EventSnapAppName, "../sampleconfig/configch3.yaml"); err != nil {
//		t.Fatalf("Error saving config: %s", err)
//	}
//	config, err = New(channelID3, "../sampleconfig")
//	if err != nil {
//		t.Fatalf("Error creating new config: %s", err)
//	}
//	// try to load the config in a new viper instance and verify the cert pem is loaded
//	// as we don't have access to the cert/key in config.TransportCredentials
//	v := viper.New()
//	v.SetConfigFile("../sampleconfig/configch3.yaml")
//	v.ReadInConfig()
//	p := v.Get("eventsnap.eventhub.tlsCerts.client.certpem")
//
//	if p == "" {
//		t.Fatalf("certpem is empty when loading from viper")
//	}
//
//	cp, err := ioutil.ReadFile("tls/client_sdk_go.pem")
//	checkString(t, "eventhub.tlsCerts.client.certpem", p.(string), fmt.Sprintf("%s", cp))
//}
//
//func checkString(t *testing.T, field string, value string, expectedValue string) {
//	if value != expectedValue {
//		t.Fatalf("Expecting [%s] for [%s] but got [%s]", expectedValue, field, value)
//	}
//}
//
//func checkUint(t *testing.T, field string, value, expectedValue uint) {
//	if value != expectedValue {
//		t.Fatalf("Expecting [%d] for [%s] but got [%d]", expectedValue, field, value)
//	}
//}
//
//func checkDuration(t *testing.T, field string, value, expectedValue time.Duration) {
//	if value != expectedValue {
//		t.Fatalf("Expecting %d for %s but got %d", expectedValue, field, value)
//	}
//}
