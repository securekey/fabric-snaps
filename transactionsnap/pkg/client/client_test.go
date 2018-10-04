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

	"github.com/pkg/errors"

	"github.com/securekey/fabric-snaps/mocks/mockprovider"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"io/ioutil"

	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/sampleconfig"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	utilErr "github.com/securekey/fabric-snaps/util/errors"
)

const (
	cfgDir        = "../../cmd/sampleconfig/"
	txnCfgPath    = cfgDir + "config.yaml"
	peerCfgPath   = cfgDir + "core.yaml"
	org1MSP       = "Org1MSP"
	testChannelID = "mychannel"
	orgName       = "peerorg1"
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
	txnSnapConfig.SetConfigFile(txnCfgPath)
	txnSnapConfig.ReadInConfig()
	txnSnapConfig.SetEnvPrefix("core")
	txnSnapConfig.AutomaticEnv()
	txnSnapConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return config.NewMockConfig(txnSnapConfig, nil, nil)
}

func createConfig(t *testing.T, txnConfigPath, peerCfgPath string) api.Config {
	var txnSnapConfig, peerConfig *viper.Viper
	var configBytes []byte
	var err error

	if txnConfigPath != "" {
		txnSnapConfig = viper.New()
		txnSnapConfig.SetConfigType("yaml")
		txnSnapConfig.SetConfigFile(txnConfigPath)
		err = txnSnapConfig.ReadInConfig()
		assert.Nil(t, err)
		txnSnapConfig.SetEnvPrefix("core")
		txnSnapConfig.AutomaticEnv()
		txnSnapConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		configBytes, err = ioutil.ReadFile(txnConfigPath)
		assert.Nil(t, err)
	}

	if peerCfgPath != "" {
		peerConfig = viper.New()
		peerConfig.SetConfigType("yaml")
		peerConfig.SetConfigFile(peerCfgPath)
		err = peerConfig.ReadInConfig()
		assert.Nil(t, err)
		peerConfig.SetEnvPrefix("core")
		peerConfig.AutomaticEnv()
		peerConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	}

	return config.NewMockConfig(txnSnapConfig, peerConfig, configBytes)
}

//TestInitializer tests behavior on client initialize function
//tests if sdk and other related objects are refreshed only during config update
//tests if old sdk instances are closed before updating with client with new sdk instances
func TestInitializer(t *testing.T) {

	//Setup bccsp factory
	opts := sampleconfig.GetSampleBCCSPFactoryOpts("../../cmd/sampleconfig")

	//Now call init factories using opts you got
	bccspFactory.InitFactories(opts)

	configProvider := func(channelID string) (api.Config, error) {
		return createConfig(t, txnCfgPath, peerCfgPath), nil
	}

	client, err := checkClient(testChannelID, nil, configProvider, &mockprovider.Factory{})
	require.NoError(t, err)

	//sdk, channel client, context are loaded and config hash updated
	assert.NotNil(t, client.channelClient)
	assert.NotEmpty(t, client.channelID)
	assert.NotEmpty(t, client.configHash)
	assert.NotNil(t, client.context)
	assert.NotNil(t, client.sdk)

	//initialize again,it shouldn't take any effect
	oldSdk := client.sdk
	oldCtx := client.context
	oldChannelClient := client.channelClient
	oldHash := client.configHash

	client, err = checkClient(testChannelID, client, configProvider, &mockprovider.Factory{})
	assert.NoError(t, err)

	//pointer compare should pass on sdk, context , channel client and config hash
	assert.True(t, oldSdk == client.sdk)
	assert.True(t, oldCtx == client.context)
	assert.True(t, oldChannelClient == client.channelClient)
	assert.True(t, oldHash == client.configHash)

	//Do it again, initialize,it shouldnt take any effect
	oldSdk = client.sdk
	oldCtx = client.context
	oldChannelClient = client.channelClient
	oldHash = client.configHash

	client, err = checkClient(testChannelID, client, configProvider, &mockprovider.Factory{})
	assert.Nil(t, err)

	//pointer compare should pass on sdk, context , channel client and config hash
	assert.True(t, oldSdk == client.sdk)
	assert.True(t, oldCtx == client.context)
	assert.True(t, oldChannelClient == client.channelClient)
	assert.True(t, oldHash == client.configHash)

	//now tamper config hash to imitate config update behavior
	client.configHash = "XYZ"

	//initialze, it should update all the values
	oldSdk = client.sdk
	oldCtx = client.context
	oldChannelClient = client.channelClient
	oldHash = client.configHash

	client, err = checkClient(testChannelID, client, configProvider, &mockprovider.Factory{})
	assert.NoError(t, err)
	//pointer negative compare should pass on sdk, context , channel client and config hash
	assert.False(t, oldSdk == client.sdk)
	assert.False(t, oldCtx == client.context)
	assert.False(t, oldChannelClient == client.channelClient)
	assert.False(t, oldHash == client.configHash)
}

func TestRetryableErrors(t *testing.T) {

	assert.True(t, isRetryable(errors.New("InvokeHandler Query failed: sign proposal failed: sign failed: Private key not found [Key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(errors.New("XYZ: sign proposal failed:  [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(errors.New("XYZ: sign failed:  [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(errors.New("XYZ: Private key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(errors.New("XYZ: [Key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.False(t, isRetryable(errors.New("XYZ: proposal failed: signing failed: Public key incorrect format")))

	assert.True(t, isRetryable(utilErr.New(utilErr.GeneralError, "InvokeHandler Query failed: sign proposal failed: sign failed: Private key not found [Key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(utilErr.New(utilErr.GeneralError, "XYZ: sign proposal failed:  [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(utilErr.New(utilErr.GeneralError, "XYZ: sign failed:  [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(utilErr.New(utilErr.GeneralError, "XYZ: Private key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.True(t, isRetryable(utilErr.New(utilErr.GeneralError, "XYZ: [Key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")))
	assert.False(t, isRetryable(utilErr.New(utilErr.GeneralError, "XYZ: proposal failed: signing failed: Public key incorrect format")))

	err := errors.New("InvokeHandler Query failed: sign proposal failed: sign failed: Private key not found [Key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |.")
	assert.True(t, isRetryable(utilErr.WithMessage(utilErr.GeneralError, err, "InvokeHandler Query failed")))
	assert.True(t, isRetryable(utilErr.WithMessage(utilErr.GeneralError, err, "")))
	err = errors.New("XYZ: proposal failed: signing failed: Public key incorrect format")
	assert.False(t, isRetryable(utilErr.WithMessage(utilErr.GeneralError, err, "XYZ: proposal failed: signing failed: Public key incorrect format")))

	assert.False(t, isRetryable(nil))
	assert.False(t, isRetryable(""))
	assert.False(t, isRetryable("InvokeHandler Query failed: sign proposal failed: sign failed: Private key not found [Key not found [00000000  ab be 8e e0 f8 6c 22 7b  19 17 d2 08 92 14 97 60  |."))

}
