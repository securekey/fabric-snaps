/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazyref"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	selection "github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/dynamicselection"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"

	"crypto/sha256"
	"encoding/base64"
	"sync/atomic"

	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config/endpoint"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	"github.com/hyperledger/fabric-sdk-go/pkg/util/concurrent/lazycache"
	pb "github.com/hyperledger/fabric/protos/peer"
	dynamicDiscovery "github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/provider"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories"
	factoriesMsp "github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories/msp"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/handler"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/localprovider"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/utils"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

const (
	txnSnapUser = "Txn-Snap-User"
)

type clientImpl struct {
	sync.RWMutex
	txnSnapConfig api.Config
	clientConfig  fabApi.EndpointConfig
	channelClient *channel.Client
	channelID     string
	context       contextApi.Channel
	mutex         sync.RWMutex
	configHash    atomic.Value
	sdk           *fabsdk.FabricSDK
}

// DynamicProviderFactory is configured with dynamic discovery provider and dynamic selection provider
type DynamicProviderFactory struct {
	defsvc.ProviderFactory
	ChannelUsers []selection.ChannelUser
}

// CreateDiscoveryProvider returns a new implementation of dynamic discovery provider
func (f *DynamicProviderFactory) CreateDiscoveryProvider(config fabApi.EndpointConfig) (fabApi.DiscoveryProvider, error) {
	return dynamicDiscovery.New(config), nil
}

// CreateSelectionProvider returns a new implementation of dynamic selection provider
func (f *DynamicProviderFactory) CreateSelectionProvider(config fabApi.EndpointConfig) (fabApi.SelectionProvider, error) {
	return selection.New(config, f.ChannelUsers)
}

// CustomConfig override client config
type CustomConfig struct {
	fabApi.EndpointConfig
	localPeer           *api.PeerConfig
	localPeerTLSCertPem []byte
}

// ChannelPeers returns the channel peers configuration
// TODO this is a workaround.
// Currently there is no way to pass in a set of target peers to the selection provider.
func (c *CustomConfig) ChannelPeers(name string) ([]fabApi.ChannelPeer, error) {
	peerConfig, err := c.PeerConfig(fmt.Sprintf("%s:%d", c.localPeer.Host,
		c.localPeer.Port))
	if err != nil {
		return nil, fmt.Errorf("error get peer config by url: %v", err)
	}
	networkPeer := fabApi.NetworkPeer{PeerConfig: *peerConfig, MSPID: string(c.localPeer.MSPid)}
	networkPeer.TLSCACerts = endpoint.TLSConfig{Pem: string(c.localPeerTLSCertPem)}
	peer := fabApi.ChannelPeer{PeerChannelConfig: fabApi.PeerChannelConfig{EndorsingPeer: true,
		ChaincodeQuery: true, LedgerQuery: true, EventSource: true}, NetworkPeer: networkPeer}
	logger.Debugf("ChannelPeers return %v", peer)
	return []fabApi.ChannelPeer{peer}, nil
}

var once sync.Once

//ServiceProviderFactory use to pass service provider factory(mock for unit test)
var ServiceProviderFactory apisdk.ServiceProviderFactory
var cache *lazycache.Cache

// GetInstance returns a singleton instance of the fabric client
func GetInstance(channelID string, txnSnapConfig api.Config) (api.Client, error) {
	return getInstance(newCacheKey(channelID, txnSnapConfig, ServiceProviderFactory))
}

// GetInstanceWithLocalDiscovery returns a singleton instance of the fabric client with local discovery
func GetInstanceWithLocalDiscovery(channelID string, txnSnapConfig api.Config) (api.Client, error) {
	localPeer, err := txnSnapConfig.GetLocalPeer()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "GetLocalPeer return error")
	}
	serviceProviderFactory := &localprovider.Factory{LocalPeer: localPeer, LocalPeerTLSCertPem: txnSnapConfig.GetTLSCertPem()}
	return getInstance(newLocalCacheKey(channelID, txnSnapConfig, serviceProviderFactory))
}

func getInstance(key CacheKey) (api.Client, error) {

	once.Do(func() {
		logger.Debugf("Setting client cache refresh interval %d\n", key.TxnSnapConfig().GetClientCacheRefreshInterval())
		cache = newRefCache(key.TxnSnapConfig().GetClientCacheRefreshInterval())
		logger.Debugf("Cache was intialized")
	})

	ref, err := cache.Get(key)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Got error while getting item from cache")
	}

	clientRef := ref.(*lazyref.Reference)
	client, err := clientRef.Get()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "error getting client")
	}
	return client.(api.Client), nil
}

// GeneratePin returns pin of an x509 certificate
func (c *clientImpl) generateHash(bytes []byte) string {
	digest := sha256.Sum256(bytes)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func (c *clientImpl) initialize(channelID string, serviceProviderFactory apisdk.ServiceProviderFactory) error {

	currentCfgHash := c.generateHash(c.txnSnapConfig.GetConfigBytes())

	//compare config hash
	if c.configHash.Load() == currentCfgHash {
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	//close existing sdk instance if any
	if c.sdk != nil {
		c.sdk.Close()
	}

	// Get client config
	configProvider := func() ([]core.ConfigBackend, error) {
		// Make sure the buffer is created each time it is called, otherwise
		// there will be no data left in the buffer the second time it's called
		return config.FromRaw(c.txnSnapConfig.GetConfigBytes(), "yaml")()
	}

	configBackends, err := configProvider()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "get client config return error")
	}

	endpointConfig, err := fab.ConfigFromBackend(configBackends...)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "from backend returned error")
	}

	// Get org name
	nconfig, err := endpointConfig.NetworkConfig()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Failed to get network config")
	}

	// Get local peer
	localPeer, err := c.txnSnapConfig.GetLocalPeer()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "GetLocalPeer return error")
	}

	//lookup for orgname
	var orgname string
	for name, org := range nconfig.Organizations {
		if org.MSPID == string(localPeer.MSPid) {
			orgname = name
			break
		}
	}
	if orgname == "" {
		return errors.Errorf(errors.GeneralError, "Failed to get %s from client config", localPeer.MSPid)
	}

	//Get cryptosuite provider name from name from peerconfig
	cryptoProvider, err := c.txnSnapConfig.GetCryptoProvider()
	if err != nil {
		return err
	}

	if serviceProviderFactory == nil {
		channelUser := selection.ChannelUser{ChannelID: channelID, Username: txnSnapUser, OrgName: orgname}
		serviceProviderFactory = &DynamicProviderFactory{ChannelUsers: []selection.ChannelUser{channelUser}}
	}

	customEndpointConfig := NewCustomConfig(endpointConfig, localPeer, c.txnSnapConfig.GetTLSCertPem())

	//create sdk
	c.sdk, err = fabsdk.New(configProvider,
		fabsdk.WithConfigEndpoint(customEndpointConfig),
		fabsdk.WithCorePkg(&factories.CustomCorePkg{ProviderName: cryptoProvider}),
		fabsdk.WithServicePkg(serviceProviderFactory),
		fabsdk.WithMSPPkg(&factoriesMsp.CustomMspPkg{CryptoPath: c.txnSnapConfig.GetMspConfigPath()}))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	// new channel context prov
	chContextProv := c.sdk.ChannelContext(channelID, fabsdk.WithUser(txnSnapUser), fabsdk.WithOrg(orgname))

	// Channel client is used to query and execute transactions
	chClient, err := channel.New(chContextProv)
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Failed to create new channel(%v) client %v", channelID, err)
	}
	if chClient == nil {
		return errors.New(errors.GeneralError, "channel client is nil")
	}

	chContext, err := chContextProv()
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Failed to call func channel(%v) context %v", channelID, err)
	}

	c.channelClient = chClient
	c.channelID = channelID
	c.clientConfig = customEndpointConfig
	c.context = chContext
	c.configHash.Store(currentCfgHash)

	return nil
}

//NewCustomConfig return custom endpoint config
func NewCustomConfig(config fabApi.EndpointConfig, localPeer *api.PeerConfig, localPeerTLSCertPem []byte) fabApi.EndpointConfig {
	return &CustomConfig{EndpointConfig: config, localPeer: localPeer, localPeerTLSCertPem: localPeerTLSCertPem}
}

func (c *clientImpl) EndorseTransaction(endorseRequest *api.EndorseTxRequest) (*channel.Response, error) {
	logger.Debugf("EndorseTransaction with endorseRequest %v", endorseRequest)

	targets := endorseRequest.Targets
	if len(endorseRequest.Args) < 1 {
		return nil, errors.New(errors.GeneralError, "function arg is required")
	}
	args := make([][]byte, 0)
	if len(endorseRequest.Args) > 1 {
		for _, value := range endorseRequest.Args[1:] {
			args = append(args, []byte(value))
		}
	}

	customQueryHandler := handler.NewPeerFilterHandler(endorseRequest.ChaincodeIDs, c.txnSnapConfig,
		invoke.NewEndorsementHandler(
			invoke.NewEndorsementValidationHandler(
				invoke.NewSignatureValidationHandler(),
			),
		),
	)

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	response, err := c.channelClient.InvokeHandler(customQueryHandler, channel.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args, TransientMap: endorseRequest.TransientData}, channel.WithTargets(targets...), channel.WithTargetFilter(endorseRequest.PeerFilter),
		channel.WithRetry(c.retryOpts()))

	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "InvokeHandler Query failed")
	}
	return &response, nil
}

func (c *clientImpl) CommitTransaction(endorseRequest *api.EndorseTxRequest, registerTxEvent bool, callback api.EndorsedCallback) (*channel.Response, error) {
	logger.Debugf("CommitTransaction with endorseRequest %v", endorseRequest)
	targets := endorseRequest.Targets
	if len(endorseRequest.Args) < 1 {
		return nil, errors.New(errors.GeneralError, "function arg is required")
	}
	args := make([][]byte, 0)
	if len(endorseRequest.Args) > 1 {
		for _, value := range endorseRequest.Args[1:] {
			args = append(args, []byte(value))
		}
	}

	customExecuteHandler := handler.NewPeerFilterHandler(endorseRequest.ChaincodeIDs, c.txnSnapConfig,
		invoke.NewEndorsementHandler(
			invoke.NewEndorsementValidationHandler(
				invoke.NewSignatureValidationHandler(
					handler.NewCheckForCommitHandler(endorseRequest.RWSetIgnoreNameSpace, callback,
						handler.NewLocalEventCommitHandler(registerTxEvent, c.channelID),
					),
				),
			),
		),
	)

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	resp, err := c.channelClient.InvokeHandler(customExecuteHandler, channel.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args, TransientMap: endorseRequest.TransientData}, channel.WithTargets(targets...), channel.WithTargetFilter(endorseRequest.PeerFilter),
		channel.WithRetry(c.retryOpts()))

	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "InvokeHandler execute failed")
	}
	return &resp, nil
}

func (c *clientImpl) VerifyTxnProposalSignature(proposalBytes []byte) error {

	signedProposal := &pb.SignedProposal{}
	if err := proto.Unmarshal(proposalBytes, signedProposal); err != nil {
		return errors.Wrap(errors.GeneralError, err, "Unmarshal clientProposalBytes error")
	}

	creatorBytes, err := utils.GetCreatorFromSignedProposal(signedProposal)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "GetCreatorFromSignedProposal return error")
	}

	logger.Debugf("checkSignatureFromCreator info: creator is %s", creatorBytes)

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	membership, err := c.context.ChannelService().Membership()
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "Failed to get Membership from channelService")
	}

	// ensure that creator is a valid certificate
	err = membership.Validate(creatorBytes)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "The creator certificate is not valid")
	}

	logger.Debugf("verifyTPSignature info: creator is valid")

	// validate the signature
	err = membership.Verify(creatorBytes, signedProposal.ProposalBytes, signedProposal.Signature)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "The creator's signature over the proposal is not valid")
	}

	logger.Debugf("VerifyTxnProposalSignature exists successfully")

	return nil
}

func (c *clientImpl) GetTargetPeer(peerCfg *api.PeerConfig, opts ...peer.Option) (fabApi.Peer, error) {

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	//TODO argument 'peerCfg' should come as opts instead of nil check
	if peerCfg != nil {
		peerConfig, err := c.clientConfig.PeerConfig(fmt.Sprintf("%s:%d", peerCfg.Host,
			peerCfg.Port))
		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "Failed to get peer config by url")
		}
		opts = append(opts, peer.FromPeerConfig(&fabApi.NetworkPeer{PeerConfig: *peerConfig, MSPID: string(peerCfg.MSPid)}),
			peer.WithTLSCert(c.txnSnapConfig.GetTLSRootCert()))
	}

	targetPeer, err := peer.New(c.clientConfig, opts...)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "Failed create peer by peer config")
	}

	return targetPeer, nil
}

func (c *clientImpl) retryOpts() retry.Opts {
	opts := c.txnSnapConfig.RetryOpts()
	opts.RetryableCodes = make(map[status.Group][]status.Code)
	for key, value := range retry.ChannelClientRetryableCodes {
		opts.RetryableCodes[key] = value
	}
	ccCodes, err := c.txnSnapConfig.CCErrorRetryableCodes()
	if err != nil {
		logger.Warnf("Could not parse CC error retry args: %s", err.Error())
	}
	for _, code := range ccCodes {
		addRetryCode(opts.RetryableCodes, status.ChaincodeStatus, status.Code(code))
	}

	addRetryCode(opts.RetryableCodes, status.ClientStatus, status.NoPeersFound)

	return opts
}

//GetContext returns SDK context object of given client
//For thread safety, care should be taken while using returned value since it can be updated if there are any
// txnsnap config updates and lazyref cache refresh kicks in.
func (c *clientImpl) GetContext() contextApi.Client {

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.context
}

// addRetryCode adds the given group and code to the given map
func addRetryCode(codes map[status.Group][]status.Code, group status.Group, code status.Code) {
	g, exists := codes[group]
	if !exists {
		g = []status.Code{}
	}
	codes[group] = append(g, code)
}
