/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/spf13/viper"
)

type configClientImpl struct {
	viper *viper.Viper //temp until the configmanager implementation complete
}

func (cc *configClientImpl) Get(stub shim.ChaincodeStubInterface, configKey *api.ConfigKey) (viper *viper.Viper, err error) {
	return cc.viper, nil
}
