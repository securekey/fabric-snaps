// +build kevlar

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rolesmgr

import (
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
	"github.com/hyperledger/fabric/protos/gossip"
)

//HasEndorserRole returns if given ledger config has endorser role for current peer
//temporary hook for kevlar "ledgerconfig.HasRole(ledgerconfig.EndorserRole)"
func HasEndorserRole() bool {
	return !ledgerconfig.HasRole(ledgerconfig.EndorserRole)
}

//AllRoles returns all roles from given gossip properties
func AllRoles(properties *gossip.Properties) []string {
	return properties.Roles
}
