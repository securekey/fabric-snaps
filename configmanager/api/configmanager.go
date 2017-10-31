/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

// ConfigClient is used to publish messages
type ConfigClient interface {
	Get(org string, peer string, appname string) (viperConfig string, err error)
}

//ConfigManager is used to manage configuration in ledger(save,get,delete)
type ConfigManager interface {
	//Save configuration
	Save(jsonConfig string) (saved bool, err error)
	//Get configuration
	Get(org string, peer string, appname string) (appconfig string, err error)
	//Delete configuration
	Delete(org string, peer string, appname string) (deleted bool, err error)
}
