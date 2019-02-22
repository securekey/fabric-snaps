/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rolesmgr

import "github.com/hyperledger/fabric/protos/gossip"

//HasEndorserRole returns if given ledger config has endorser role for current peer
// it is always true for vanilla fabric
func HasEndorserRole() bool {
	return true
}

//AllRoles returns all roles from given gossip properties
func AllRoles(properties *gossip.Properties) []string {
	return []string{}
}
