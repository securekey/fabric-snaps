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
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"io/ioutil"

	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/service"
	"github.com/securekey/fabric-snaps/membershipsnap/pkg/membership"
	"github.com/securekey/fabric-snaps/mocks/mockbcinfo"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/sampleconfig"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
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

	//create client
	client := &clientImpl{
		txnSnapConfig: createConfig(t, txnCfgPath, peerCfgPath),
	}

	//make sure everything is nil/empty before initialize
	assert.Nil(t, client.channelClient)
	assert.Empty(t, client.channelID)
	assert.Empty(t, client.configHash.Load())
	assert.Nil(t, client.context)
	assert.Nil(t, client.sdk)

	//initialize on test channel ID
	err := client.initialize(testChannelID, &MockProviderFactory{})
	assert.Nil(t, err)

	//sdk, channel client, context are loaded and config hash updated
	assert.NotNil(t, client.channelClient)
	assert.NotEmpty(t, client.channelID)
	assert.NotEmpty(t, client.configHash.Load())
	assert.NotNil(t, client.context)
	assert.NotNil(t, client.sdk)

	//initialize again,it shouldn't take any effect
	oldSdk := client.sdk
	oldCtx := client.context
	oldChannelClient := client.channelClient
	oldHash := client.configHash.Load()

	err = client.initialize(testChannelID, &MockProviderFactory{})
	assert.Nil(t, err)

	//pointer compare should pass on sdk, context , channel client and config hash
	assert.True(t, oldSdk == client.sdk)
	assert.True(t, oldCtx == client.context)
	assert.True(t, oldChannelClient == client.channelClient)
	assert.True(t, oldHash == client.configHash.Load())

	//Do it again, initialize,it shouldnt take any effect
	oldSdk = client.sdk
	oldCtx = client.context
	oldChannelClient = client.channelClient
	oldHash = client.configHash.Load()

	err = client.initialize(testChannelID, &MockProviderFactory{})
	assert.Nil(t, err)

	//pointer compare should pass on sdk, context , channel client and config hash
	assert.True(t, oldSdk == client.sdk)
	assert.True(t, oldCtx == client.context)
	assert.True(t, oldChannelClient == client.channelClient)
	assert.True(t, oldHash == client.configHash.Load())

	//now tamper config hash to imitate config update behavior
	client.configHash.Store("XYZ")

	//initialze, it should update all the values
	oldSdk = client.sdk
	oldCtx = client.context
	oldChannelClient = client.channelClient
	oldHash = client.configHash.Load()

	err = client.initialize(testChannelID, &MockProviderFactory{})
	assert.Nil(t, err)
	//pointer negative compare should pass on sdk, context , channel client and config hash
	assert.False(t, oldSdk == client.sdk)
	assert.False(t, oldCtx == client.context)
	assert.False(t, oldChannelClient == client.channelClient)
	assert.False(t, oldHash == client.configHash.Load())

	//Test if previous sdk is closed,
	chCtxPvdr := oldSdk.ChannelContext(testChannelID, fabsdk.WithUser(txnSnapUser), fabsdk.WithOrg(orgName))
	chCtx, err := chCtxPvdr()
	assert.NotNil(t, chCtx)
	assert.Nil(t, err)

	//channel membership should fail with cache closed error since oldSDK is closed
	mmbr, err := chCtx.ChannelService().Membership()
	assert.Nil(t, mmbr)
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "Channel_Cfg_Cache - cache is closed"))
}

type MockProviderFactory struct {
	defsvc.ProviderFactory
}

func (m *MockProviderFactory) CreateDiscoveryProvider(config fabApi.EndpointConfig) (fabApi.DiscoveryProvider, error) {
	return &impl{clientConfig: config}, nil
}

type impl struct {
	clientConfig fabApi.EndpointConfig
}

// CreateDiscoveryService return impl of DiscoveryService
func (p *impl) CreateDiscoveryService(channelID string) (fabApi.DiscoveryService, error) {
	memService := membership.NewServiceWithMocks([]byte(org1MSP), "internalhost1:1000", mockbcinfo.ChannelBCInfos(mockbcinfo.NewChannelBCInfo(channelID, mockbcinfo.BCInfo(uint64(1000)))))
	return service.New(channelID, p.clientConfig, memService), nil
}
