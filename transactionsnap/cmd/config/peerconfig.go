/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

// PeerConfig represents the server addresses of a fabric peer
type PeerConfig struct {
	Host      string
	Port      int
	EventHost string
	EventPort int
	MSPid     []byte
}

// PeerConfigs represents a list of peers. It implements the sort interface
type PeerConfigs []PeerConfig

func (p PeerConfigs) Len() int {
	return len(p)
}

func (p PeerConfigs) Less(i, j int) bool {
	return p[i].Host < p[j].Host
}

func (p PeerConfigs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
