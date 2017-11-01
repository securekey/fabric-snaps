/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	transactionsnapApi "github.com/securekey/fabric-snaps/transactionsnap/api"
)

// PeerConfigs represents a list of peers. It implements the sort interface
type PeerConfigs []transactionsnapApi.PeerConfig

func (p PeerConfigs) Len() int {
	return len(p)
}

func (p PeerConfigs) Less(i, j int) bool {
	return p[i].Host < p[j].Host
}

func (p PeerConfigs) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
