/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
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
