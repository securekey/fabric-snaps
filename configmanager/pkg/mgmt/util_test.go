/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"testing"

	"github.com/securekey/fabric-snaps/configmanager/api"
)

func TestCreateConfigKey(t *testing.T) {
	if _, err := CreateConfigKey("", "asv", "aaa"); err == nil {
		t.Fatalf("Expected error ")
	}
	if _, err := CreateConfigKey("safsdf", "", "aaa"); err == nil {
		t.Fatalf("Expected error ")
	}
	if _, err := CreateConfigKey("sdfsdf", "asv", ""); err == nil {
		t.Fatalf("Expected error ")
	}

}

func TestValidateConfigKey(t *testing.T) {
	key := api.ConfigKey{MspID: "", PeerID: "aaa", AppName: "bbbb"}
	if err := ValidateConfigKey(key); err == nil {
		t.Fatalf("Expected error ")
	}
	key.PeerID = ""
	if err := ValidateConfigKey(key); err == nil {
		t.Fatalf("Expected error ")
	}
	key.PeerID = "abc"
	key.AppName = ""
	if err := ValidateConfigKey(key); err == nil {
		t.Fatalf("Expected error ")
	}
}

func TestConfigKeyToString(t *testing.T) {
	key := api.ConfigKey{MspID: "abc", PeerID: "peer.zero.sk.example", AppName: "testApp"}
	keyStr, _ := ConfigKeyToString(key)
	expectedKeyString := "abc!peer.zero.sk.example!testApp"
	if keyStr != expectedKeyString {
		t.Fatalf("Expected key string %s. Got %s", expectedKeyString, keyStr)
	}
}

func TestStringToConfigKey(t *testing.T) {
	key := "abc!peer.zero.sk.example!testApp"
	if _, err := StringToConfigKey(key); err != nil {
		t.Fatalf("Error %s", err)
	}

	key = "abc!peer.zero.sk.exampletestApp"
	if _, err := StringToConfigKey(key); err == nil {
		t.Fatalf("Expecting error 'Invalid config key abc!peer.zero.sk.exampletestApp'")
	}
	key = ""
	if _, err := StringToConfigKey(key); err == nil {
		t.Fatalf("Expecting error 'Invalid config key '")
	}

}
