/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/spf13/viper"
)

// ConfigKey contain org,peer,appname
type ConfigKey struct {
	Org     string
	Peer    string
	Appname string
}

// ConfigClient is used to publish messages
type ConfigClient interface {
	Get(stub shim.ChaincodeStubInterface, configKey *ConfigKey) (viper *viper.Viper, err error)
}

//ConfigManager is used to manage configuration in ledger(save,get,delete)
type ConfigManager interface {
	//Save configuration
	Save(jsonConfig string) (saved bool, err error)
	//Get configuration
	Get(configKey ConfigKey) (appconfig string, err error)
	//Delete configuration
	Delete(configKey ConfigKey) (deleted bool, err error)
}
