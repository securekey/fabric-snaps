/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

var txnSnapConfig *viper.Viper
var coreConfig *viper.Viper

func TestIsTLSEnabled(t *testing.T) {
	value := IsTLSEnabled()
	if value == txnSnapConfig.GetBool("client.tls.enabled") {
		t.Fatalf("Expected IsTLSEnabled() return value %v but got %v", txnSnapConfig.GetBool("client.tls.enabled"), value)
	}
}

func TestGetMspID(t *testing.T) {
	value := GetMspID()
	if value != coreConfig.GetString("peer.localMspId") {
		t.Fatalf("Expected GetMspID() return value %v but got %v", coreConfig.GetString("peer.localMspId"), value)
	}
}

func TestGetTLSRootCertPath(t *testing.T) {
	value := GetTLSRootCertPath()
	if value != GetConfigPath(coreConfig.GetString("peer.tls.rootcert.file")) {
		t.Fatalf("Expected GetTLSRootCertPath() return value %v but got %v",
			GetConfigPath(coreConfig.GetString("peer.tls.rootcert.file")), value)
	}
}

func TestGetTLSCertPath(t *testing.T) {
	value := GetTLSCertPath()
	if value != GetConfigPath(coreConfig.GetString("peer.tls.cert.file")) {
		t.Fatalf("Expected GetTLSCertPath() return value %v but got %v",
			GetConfigPath(coreConfig.GetString("peer.tls.cert.file")), value)
	}
}

func TestGetTLSKeyPath(t *testing.T) {
	value := GetTLSKeyPath()
	if value != GetConfigPath(coreConfig.GetString("peer.tls.key.file")) {
		t.Fatalf("Expected GetTLSKeyPath() return value %v but got %v",
			GetConfigPath(coreConfig.GetString("peer.tls.key.file")), value)
	}
}

func TestGetEnrolmentCertPath(t *testing.T) {
	value := GetEnrolmentCertPath()
	if value != GetConfigPath(txnSnapConfig.GetString("txnsnap.enrolment.cert.file")) {
		t.Fatalf("Expected GetEnrolmentCertPath() return value %v but got %v",
			GetConfigPath(txnSnapConfig.GetString("txnsnap.enrolment.cert.file")), value)
	}
}

func TestGetEnrolmentKeyPath(t *testing.T) {
	value := GetEnrolmentKeyPath()
	if value != GetConfigPath(txnSnapConfig.GetString("txnsnap.enrolment.key.file")) {
		t.Fatalf("Expected GetEnrolmentKeyPath() return value %v but got %v",
			GetConfigPath(txnSnapConfig.GetString("txnsnap.enrolment.key.file")), value)
	}
}

func TestGetMembershipPollInterval(t *testing.T) {
	value := GetMembershipPollInterval()
	if value != txnSnapConfig.GetDuration("txnsnap.membership.pollinterval") {
		t.Fatalf("Expected GetMembershipPollInterval() return value %v but got %v",
			GetConfigPath(txnSnapConfig.GetString("txnsnap.membership.pollinterval")), value)
	}
}

func TestGetMembershipChannelPeers(t *testing.T) {
	membershipChannelPeers, err := GetMembershipChannelPeers("channel0")
	if err != nil {
		t.Fatalf("GetMembershipPeers return error %v", err)
	}
	var expectedMembershipChannelsPeers map[string]*MembershipChannelPeers
	txnSnapConfig.UnmarshalKey("txnsnap.membership.channels", &expectedMembershipChannelsPeers)
	expectedMembershipChannelPeers := expectedMembershipChannelsPeers["channel0"].Peers

	for key, value := range membershipChannelPeers {
		if value.Host != expectedMembershipChannelPeers[key].Host {
			t.Fatalf("Expected GetMembershipChannelPeers() Host return value %v but got %v",
				expectedMembershipChannelPeers[key].Host, value.Host)
		}
		if value.Port != expectedMembershipChannelPeers[key].Port {
			t.Fatalf("Expected GetMembershipChannelPeers() Port return value %v but got %v",
				expectedMembershipChannelPeers[key].Port, value.Port)
		}
		if value.MspID != expectedMembershipChannelPeers[key].MspID {
			t.Fatalf("Expected GetMembershipChannelPeers() MspID return value %v but got %v",
				expectedMembershipChannelPeers[key].MspID, value.MspID)
		}
	}

}

func TestGetLocalPeer(t *testing.T) {
	peerConfig.Set("peer.address", "")
	_, err := GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != "Peer address not found in config" {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	peerConfig.Set("peer.address", "peer:Address")
	peerConfig.Set("peer.events.address", "")
	_, err = GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != "Peer event address not found in config" {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	peerConfig.Set("peer.events.address", "peer:EventAddress")
	_, err = GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != `strconv.ParseInt: parsing "Address": invalid syntax` {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	peerConfig.Set("peer.address", "peer:5050")
	_, err = GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != `strconv.ParseInt: parsing "EventAddress": invalid syntax` {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	peerConfig.Set("peer.events.address", "event:5151")
	peerConfig.Set("peer.localMspId", "")
	_, err = GetLocalPeer()
	if err == nil {
		t.Fatal("GetLocalPeer() didn't return error")
	}
	if err.Error() != "Peer localMspId not found in config" {
		t.Fatal("GetLocalPeer() didn't return expected error msg")
	}
	peerConfig.Set("peer.localMspId", "mspID")
	localPeer, err := GetLocalPeer()
	if err != nil {
		t.Fatalf("GetLocalPeer() return error %v", err)
	}
	if localPeer.Host != "peer" {
		t.Fatalf("Expected localPeer.Host value %s but got %s",
			"peer", localPeer.Host)
	}
	if localPeer.Port != 5050 {
		t.Fatalf("Expected localPeer.Port value %d but got %d",
			5050, localPeer.Port)
	}
	if localPeer.EventHost != "event" {
		t.Fatalf("Expected localPeer.EventHost value %s but got %s",
			"event", localPeer.Host)
	}
	if localPeer.EventPort != 5151 {
		t.Fatalf("Expected localPeer.EventPort value %d but got %d",
			5151, localPeer.EventPort)
	}
	if string(localPeer.MSPid) != "mspID" {
		t.Fatalf("Expected localPeer.MSPid value %s but got %s",
			"mspID", localPeer.MSPid)
	}

}

func TestGetConfigPath(t *testing.T) {
	if GetConfigPath("/") != "/" {
		t.Fatalf(`Expected GetConfigPath("/") value %s but got %s`,
			"/", "/")
	}
}

func TestInitializeLogging(t *testing.T) {
	viper.Set("txnsnap.loglevel", "wrongLogValue")
	defer viper.Set("txnsnap.loglevel", "info")
	err := initializeLogging()
	if err == nil {
		t.Fatal("initializeLogging() didn't return error")
	}
	if err.Error() != "Error initializing log level: logger: invalid log level" {
		t.Fatal("initializeLogging() didn't return expected error msg")
	}
}

func TestMain(m *testing.M) {
	err := Init("../sampleconfig")
	if err != nil {
		panic(err.Error())
	}
	txnSnapConfig = viper.New()
	txnSnapConfig.SetConfigFile("../sampleconfig/config.yaml")
	txnSnapConfig.ReadInConfig()
	coreConfig = viper.New()
	coreConfig.SetConfigFile("../sampleconfig/core.yaml")
	coreConfig.ReadInConfig()
	os.Exit(m.Run())
}
