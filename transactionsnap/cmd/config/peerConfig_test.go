/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"testing"
)

func TestLenFunc(t *testing.T) {
	peerConfig := PeerConfig{Host: "localhost", Port: 0, EventHost: "localhost", EventPort: 0, MSPid: nil}
	var peerConfigs PeerConfigs
	peerConfigs = append(peerConfigs, peerConfig)
	if peerConfigs.Len() != 1 {
		t.Fatal("peerConfigs.Len() return wrong value")
	}
}

func TestLessFunc(t *testing.T) {
	peerConfig := PeerConfig{Host: "a", Port: 0, EventHost: "a", EventPort: 0, MSPid: nil}
	peerConfig1 := PeerConfig{Host: "b", Port: 0, EventHost: "b", EventPort: 0, MSPid: nil}
	var peerConfigs PeerConfigs
	peerConfigs = append(peerConfigs, peerConfig)
	peerConfigs = append(peerConfigs, peerConfig1)
	if peerConfigs.Less(0, 1) != true {
		t.Fatal("peerConfigs.less return wrong value")
	}
}

func TestSwapFunc(t *testing.T) {
	peerConfig := PeerConfig{Host: "a", Port: 0, EventHost: "a", EventPort: 0, MSPid: nil}
	peerConfig1 := PeerConfig{Host: "b", Port: 0, EventHost: "b", EventPort: 0, MSPid: nil}
	var peerConfigs PeerConfigs
	peerConfigs = append(peerConfigs, peerConfig)
	peerConfigs = append(peerConfigs, peerConfig1)
	peerConfigs.Swap(0, 1)
	if peerConfigs[0].Host != "b" {
		t.Fatal("peerConfigs.Swap didn't swap correctly")
	}
	if peerConfigs[1].Host != "a" {
		t.Fatal("peerConfigs.Swap didn't swap correctly")
	}
}
