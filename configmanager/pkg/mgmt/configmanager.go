/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mgmt

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

//ConfigClientImpl ...
type ConfigClientImpl struct {
	stub shim.ChaincodeStubInterface
}

// Save to save configuration in ledger
func (ccImpl *ConfigClientImpl) Save(jsonConfig string) (bool, error) {

	if len(jsonConfig) == 0 {
		return false, fmt.Errorf("Configuration must be provided")
	}
	//parse json config
	//getnerate key

	return true, nil
}
