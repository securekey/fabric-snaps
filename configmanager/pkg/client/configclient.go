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
	stub  shim.ChaincodeStubInterface
	viper *viper.Viper
}

// NewConfigClient returns a new ConfigClient
func NewConfigClient(stub shim.ChaincodeStubInterface, viper *viper.Viper) api.ConfigClient {
	return &configClientImpl{stub: stub, viper: viper}
}

func (cc *configClientImpl) Get(org string, peer string, appname string) (viper *viper.Viper, err error) {
	return cc.viper, nil
}
