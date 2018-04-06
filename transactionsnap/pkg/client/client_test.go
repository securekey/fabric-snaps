// +build testing

/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"os"
	"strings"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
)

func TestRetryOpts(t *testing.T) {
	impl := &clientImpl{
		txnSnapConfig: mockConfig(),
	}
	os.Setenv("CORE_TXNSNAP_RETRY_CCERRORCODES", "500 501")
	opts := impl.retryOpts()
	assert.NotNil(t, opts)
	o, ok := opts.RetryableCodes[status.ChaincodeStatus]
	assert.True(t, ok)
	assert.Len(t, o, 2)
	assert.Contains(t, o, status.Code(500))
	assert.Contains(t, o, status.Code(501))

	os.Setenv("CORE_TXNSNAP_RETRY_CCERRORCODES", "")
	opts = impl.retryOpts()
	assert.NotNil(t, opts)
	_, ok = opts.RetryableCodes[status.ChaincodeStatus]
	assert.False(t, ok)

	o, ok = opts.RetryableCodes[status.ClientStatus]
	assert.True(t, ok)
	assert.Contains(t, o, status.NoPeersFound)
}

func mockConfig() api.Config {
	txnSnapConfig := viper.New()
	txnSnapConfig.SetConfigType("YAML")
	txnSnapConfig.SetConfigFile("../../cmd/sampleconfig/config.yaml")
	txnSnapConfig.ReadInConfig()
	txnSnapConfig.SetEnvPrefix("core")
	txnSnapConfig.AutomaticEnv()
	txnSnapConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return config.NewMockConfig(txnSnapConfig, nil)
}
