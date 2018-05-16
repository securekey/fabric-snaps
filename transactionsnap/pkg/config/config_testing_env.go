// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/spf13/viper"
)

// NewMockConfig setup mock configuration for testing
func NewMockConfig(txnSnapConfig, peerConfig *viper.Viper, configBytes []byte) api.Config {
	return &Config{
		peerConfig:         peerConfig,
		txnSnapConfig:      txnSnapConfig,
		txnSnapConfigBytes: configBytes,
	}
}
