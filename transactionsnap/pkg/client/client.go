/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/status"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"

	contextApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config/lookup"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	peerpb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/metrics/cmd/filter/metrics"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/chprovider"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories"
	factoriesMsp "github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories/msp"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/handler"
	txsnapconfig "github.com/securekey/fabric-snaps/transactionsnap/pkg/config"
	"github.com/securekey/fabric-snaps/util"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/securekey/fabric-snaps/util/refcount"
)

var logger = logging.NewLogger("txnsnap")

const (
	txnSnapUser     = "Txn-Snap-User"
	defaultLogLevel = "info"
)

// ConfigProvider returns the config for the given channel
type ConfigProvider func(channelID string) (api.Config, error)

var retryCounter = metrics.RootScope.Counter("transaction_retry")

//PeerConfigPath use for testing
var PeerConfigPath = ""

//ServiceProviderFactory use to pass service provider factory(mock for unit test)
var ServiceProviderFactory apisdk.ServiceProviderFactory

//CfgProvider contains the config provider (may be mocked for unit testing)
var CfgProvider = func(channelID string) (api.Config, error) {
	return txsnapconfig.NewConfig(PeerConfigPath, channelID)
}

var cache = newRefCache(5 * time.Second) // FIXME: Make configurable

type clientImpl struct {
	*refcount.ReferenceCounter
	channelID              string
	txnSnapConfig          api.Config
	clientConfig           fabApi.EndpointConfig
	channelClient          *channel.Client
	context                contextApi.Channel
	configHash             string
	sdk                    *fabsdk.FabricSDK
	serviceProviderFactory *DynamicProviderFactory
}

// DynamicProviderFactory returns a Channel Provider that uses a dynamic discovery provider
// based on the local Membership Snap, dynamic selection provider, and the local Event Snap
type DynamicProviderFactory struct {
	defsvc.ProviderFactory
	chProvider *chprovider.Provider
}

func newServiceProvider() *DynamicProviderFactory {
	return &DynamicProviderFactory{}
}

// CreateChannelProvider returns a new default implementation of channel provider
func (f *DynamicProviderFactory) CreateChannelProvider(config fabApi.EndpointConfig) (fabApi.ChannelProvider, error) {
	if f.chProvider != nil {
		return f.chProvider, nil
	}

	chProvider, err := chprovider.New(config)
	if err != nil {
		return nil, err
	}
	f.chProvider = chProvider
	return chProvider, nil
}

func (f *DynamicProviderFactory) channelProvider() *chprovider.Provider {
	return f.chProvider
}

// CustomConfig override client config
type CustomConfig struct {
	fabApi.EndpointConfig
	localPeer           *api.PeerConfig
	localPeerTLSCertPem []byte
}

// ChannelPeers returns the channel peers configuration
func (c *CustomConfig) ChannelPeers(name string) ([]fabApi.ChannelPeer, bool) {
	url := fmt.Sprintf("%s:%d", c.localPeer.Host, c.localPeer.Port)
	peerConfig, ok := c.PeerConfig(url)
	if !ok {
		logger.Warnf("Could not find channel peer for [%s]", url)
		return nil, false
	}
	networkPeer, err := txsnapconfig.NewNetworkPeer(peerConfig, string(c.localPeer.MSPid), c.localPeerTLSCertPem)
	if err != nil {
		logger.Errorf(errors.WithMessage(errors.SystemError, err, fmt.Sprintf("Error creating network peer for [%s]", url)).GenerateLogMsg())
		return nil, false
	}

	peer := fabApi.ChannelPeer{PeerChannelConfig: fabApi.PeerChannelConfig{EndorsingPeer: true,
		ChaincodeQuery: true, LedgerQuery: true, EventSource: true}, NetworkPeer: *networkPeer}
	logger.Debugf("ChannelPeers return %v", peer)
	return []fabApi.ChannelPeer{peer}, true
}

// GetInstance returns an instance of the fabric client for the given channel.
func GetInstance(channelID string) (api.Client, error) {
	c := &clientWrapper{channelID: channelID}

	// Make sure that the client can be retrieved
	if _, err := c.getClient(); err != nil {
		return nil, err
	}

	return c, nil
}

// generateHash generates hash for give bytes
func generateHash(bytes []byte) string {
	digest := sha256.Sum256(bytes)
	return base64.StdEncoding.EncodeToString(digest[:])
}

func (c *clientImpl) close() {
	if c.sdk != nil {
		logger.Debugf("Closing SDK for client [%s]...", c.configHash)
		c.sdk.Close()
		logger.Debugf("... successfully closed SDK for client [%s]", c.configHash)
	}
}

func (c *clientImpl) membership() (fabApi.ChannelMembership, error) {
	return c.context.ChannelService().Membership()
}

func (c *clientImpl) discoveryService() (fabApi.DiscoveryService, error) {
	return c.context.ChannelService().Discovery()
}

func (c *clientImpl) eventService() (fabApi.EventService, error) {
	return c.context.ChannelService().EventService()
}

func (c *clientImpl) channelConfig() (fabApi.ChannelCfg, error) {
	return c.context.ChannelService().ChannelConfig()
}

func newClient(channelID string, cfg api.Config, serviceProviderFactory apisdk.ServiceProviderFactory) (*clientImpl, errors.Error) {
	// Get client config
	configProvider := func() ([]core.ConfigBackend, error) {
		// Make sure the buffer is created each time it is called, otherwise
		// there will be no data left in the buffer the second time it's called
		return config.FromRaw(cfg.GetConfigBytes(), "yaml")()
	}

	configBackends, err := configProvider()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "get client config return error")
	}

	endpointConfig, err := fab.ConfigFromBackend(configBackends...)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "from backend returned error")
	}

	// Get org name
	nconfig := endpointConfig.NetworkConfig()

	// Get local peer
	localPeer, err := cfg.GetLocalPeer()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "GetLocalPeer return error")
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
		return nil, errors.Errorf(errors.GeneralError, "Failed to get %s from client config", localPeer.MSPid)
	}

	//Get cryptosuite provider name from name from peerconfig
	cryptoProvider, err := cfg.GetCryptoProvider()
	if err != nil {
		return nil, errors.Errorf(errors.GeneralError, "error getting crypto provider on channel [%s]: %s", channelID)
	}

	var spFactory *DynamicProviderFactory
	if serviceProviderFactory == nil {
		spFactory = newServiceProvider()
		serviceProviderFactory = spFactory
	}

	customEndpointConfig := NewCustomConfig(endpointConfig, localPeer, cfg.GetTLSCertPem())

	//create sdk
	sdk, err := fabsdk.New(configProvider,
		fabsdk.WithEndpointConfig(customEndpointConfig),
		fabsdk.WithCorePkg(&factories.CustomCorePkg{ProviderName: cryptoProvider}),
		fabsdk.WithServicePkg(serviceProviderFactory),
		fabsdk.WithMSPPkg(&factoriesMsp.CustomMspPkg{CryptoPath: cfg.GetMspConfigPath()}))
	if err != nil {
		return nil, errors.Wrapf(errors.GeneralError, err, "Error creating SDK on channel [%s]", channelID)
	}

	// new channel context prov
	chContextProv := sdk.ChannelContext(channelID, fabsdk.WithUser(txnSnapUser), fabsdk.WithOrg(orgname))

	chContext, err := chContextProv()
	if err != nil {
		return nil, errors.Wrapf(errors.GeneralError, err, "Failed to call func channel(%v) context", channelID)
	}

	// Channel client is used to query and execute transactions
	chClient, err := channel.New(func() (contextApi.Channel, error) {
		return chContext, nil
	})
	if err != nil {
		return nil, errors.Errorf(errors.GeneralError, "Failed to create new channel(%v) client: %v", channelID, err)
	}

	//update log level
	cfgBackend, err := sdk.Config()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "failed to get config backend from sdk")
	}

	client := &clientImpl{
		channelID:              channelID,
		sdk:                    sdk,
		channelClient:          chClient,
		txnSnapConfig:          cfg,
		clientConfig:           customEndpointConfig,
		context:                chContext,
		configHash:             generateHash(cfg.GetConfigBytes()),
		serviceProviderFactory: spFactory,
	}
	// close will be called when the client is closed and the last reference is released.
	client.ReferenceCounter = refcount.New(client.close)

	client.updateLogLevel(cfgBackend)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "error initializing logging")
	}

	logger.Infof("Successfully initialized client on channel [%s] with config hash [%s]", channelID, client.configHash)

	return client, nil
}

//NewCustomConfig return custom endpoint config
func NewCustomConfig(config fabApi.EndpointConfig, localPeer *api.PeerConfig, localPeerTLSCertPem []byte) fabApi.EndpointConfig {
	return &CustomConfig{EndpointConfig: config, localPeer: localPeer, localPeerTLSCertPem: localPeerTLSCertPem}
}

func (c *clientImpl) endorseTransaction(endorseRequest *api.EndorseTxRequest) (*channel.Response, errors.Error) {
	logger.Debugf("EndorseTransaction with endorseRequest %+v", getDisplayableEndorseRequest(endorseRequest))

	targets := endorseRequest.Targets
	if len(endorseRequest.Args) < 1 {
		return nil, errors.New(errors.MissingRequiredParameterError, "function arg is required")
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

	response, err := c.channelClient.InvokeHandler(customQueryHandler, channel.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args, TransientMap: endorseRequest.TransientData}, channel.WithTargets(targets...), channel.WithTargetFilter(endorseRequest.PeerFilter),
		channel.WithRetry(c.retryOpts()), channel.WithBeforeRetry(func(err error) {
			logger.Infof("Retrying on error: %s", err.Error())
			retryCounter.Inc(1)
		}))

	if err != nil {
		return nil, errors.WithMessage(errors.EndorseTxError, err, "InvokeHandler Query failed")
	}
	return &response, nil
}

// getDisplayableEndorseRequest strips out TransientData and Args[1:] from endorseRequest for logging purposes
func getDisplayableEndorseRequest(endorseRequest *api.EndorseTxRequest) api.EndorseTxRequest {
	arg0 := ""
	if len(endorseRequest.Args) > 0 {
		arg0 = endorseRequest.Args[0]
	}
	newMessage := api.EndorseTxRequest{
		ChaincodeID:          endorseRequest.ChaincodeID,
		PeerFilter:           endorseRequest.PeerFilter,
		RWSetIgnoreNameSpace: endorseRequest.RWSetIgnoreNameSpace,
		ChaincodeIDs:         endorseRequest.ChaincodeIDs,
		Targets:              endorseRequest.Targets,
		Args:                 []string{arg0},
	}

	return newMessage
}

func (c *clientImpl) computeTxnID(nonce, creator []byte, h hash.Hash) (string, error) {
	logger.Debugf("computeTxnID nonce %s creator %s", nonce, creator)
	b := append(nonce, creator...)

	_, err := h.Write(b)
	if err != nil {
		return "", err
	}
	digest := h.Sum(nil)
	id := hex.EncodeToString(digest)

	return id, nil
}

func (c *clientImpl) commitTransaction(endorseRequest *api.EndorseTxRequest, registerTxEvent bool, callback api.EndorsedCallback) (*channel.Response, errors.Error) {
	logger.Debugf("CommitTransaction with endorseRequest %+v", getDisplayableEndorseRequest(endorseRequest))
	if len(endorseRequest.Nonce) != 0 || endorseRequest.TransactionID != "" {
		validTxnID := false
		logger.Debugf("CommitTransaction endorseRequest.Nonce is not empty")
		creator, err := c.context.Serialize()
		if err != nil {
			return nil, errors.New(errors.SystemError, "get creator failed")
		}
		logger.Debugf("Get peer creator %s", creator)
		if len(endorseRequest.Nonce) != 0 && endorseRequest.TransactionID != "" {
			logger.Debugf("CommitTransaction endorseRequest.TransactionID is not empty")
			ho := cryptosuite.GetSHA256Opts()
			h, err := c.context.CryptoSuite().GetHash(ho)
			if err != nil {
				return nil, errors.New(errors.SystemError, "hash function creation failed")
			}
			txnID, err := c.computeTxnID(endorseRequest.Nonce, creator, h)
			if err != nil {
				return nil, errors.New(errors.SystemError, "computeTxnID failed")
			}
			logger.Debugf("compare computeTxnID txID %s with endorseRequest.TransactionID %s", txnID, endorseRequest.TransactionID)
			if txnID == endorseRequest.TransactionID {
				validTxnID = true
			}
		}
		if !validTxnID {
			return &channel.Response{TxValidationCode: pb.TxValidationCode_BAD_PROPOSAL_TXID, Payload: creator}, nil
		}
	}

	targets := endorseRequest.Targets
	if len(endorseRequest.Args) < 1 {
		return nil, errors.New(errors.MissingRequiredParameterError, "function arg is required")
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
					handler.NewCheckForCommitHandler(endorseRequest.RWSetIgnoreNameSpace, callback, endorseRequest.CommitType,
						handler.NewCommitTxHandler(registerTxEvent, c.channelID),
					),
				),
			),
		),
	)

	resp, err := c.channelClient.InvokeHandler(customExecuteHandler, channel.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args, TransientMap: endorseRequest.TransientData}, channel.WithTargets(targets...), channel.WithTargetFilter(endorseRequest.PeerFilter),
		channel.WithRetry(c.retryOpts()), channel.WithBeforeRetry(func(err error) {
			logger.Infof("Retrying on error: %s", err.Error())
			retryCounter.Inc(1)
		}))

	if err != nil {
		return nil, errors.WithMessage(errors.CommitTxError, err, "InvokeHandler execute failed")
	}
	return &resp, nil
}

func (c *clientImpl) verifyTxnProposalSignature(proposalBytes []byte) errors.Error {

	signedProposal := &peerpb.SignedProposal{}
	if err := proto.Unmarshal(proposalBytes, signedProposal); err != nil {
		return errors.Wrap(errors.UnmarshalError, err, "Unmarshal clientProposalBytes error")
	}

	creatorBytes, err := util.GetCreatorFromSignedProposal(signedProposal)
	if err != nil {
		return errors.Wrap(errors.SystemError, err, "GetCreatorFromSignedProposal return error")
	}

	logger.Debugf("checkSignatureFromCreator info: creator is %s", creatorBytes)

	membership, err := c.membership()
	if err != nil {
		return errors.Wrap(errors.SystemError, err, "Failed to get Membership from channelService")
	}

	// ensure that creator is a valid certificate
	err = membership.Validate(creatorBytes)
	if err != nil {
		return errors.Wrap(errors.InvalidCreatorError, err, "The creator certificate is not valid")
	}

	logger.Debug("verifyTPSignature info: creator is valid")

	// validate the signature
	err = membership.Verify(creatorBytes, signedProposal.ProposalBytes, signedProposal.Signature)
	if err != nil {
		return errors.Wrap(errors.InvalidSignatureError, err, "The creator's signature over the proposal is not valid")
	}

	logger.Debug("VerifyTxnProposalSignature exits successfully")

	return nil
}

func (c *clientImpl) getLocalPeer() (fabApi.Peer, error) {
	peerCfg, codedErr := c.txnSnapConfig.GetLocalPeer()
	if codedErr != nil {
		return nil, codedErr
	}

	peerConfig, ok := c.clientConfig.PeerConfig(fmt.Sprintf("%s:%d", peerCfg.Host, peerCfg.Port))
	if !ok {
		return nil, errors.Errorf(errors.MissingConfigDataError, "Failed to get peer config by url")
	}

	targetPeer, err := peer.New(c.clientConfig,
		peer.FromPeerConfig(
			&fabApi.NetworkPeer{
				PeerConfig: *peerConfig,
				MSPID:      string(peerCfg.MSPid),
			},
		),
		peer.WithTLSCert(c.txnSnapConfig.GetTLSRootCert()),
	)
	if err != nil {
		return nil, errors.Wrap(errors.SystemError, err, "Failed create peer by peer config")
	}

	return targetPeer, nil
}

func (c *clientImpl) getDiscoveredPeer(url string) (fabApi.Peer, error) {
	discovery, err := c.discoveryService()
	if err != nil {
		return nil, errors.Wrapf(errors.SystemError, err, "Failed to get discovery service for channel [%s]", c.channelID)
	}

	peers, err := discovery.GetPeers()
	if err != nil {
		return nil, errors.Wrap(errors.SystemError, err, "Failed to get peers for discovery service")
	}
	for _, peer := range peers {
		if peer.URL() == url {
			return peer, nil
		}
	}
	return nil, errors.Errorf(errors.SystemError, "Peer [%s] not found", url)
}

func (c *clientImpl) retryOpts() retry.Opts {
	opts := c.txnSnapConfig.RetryOpts()
	opts.RetryableCodes = make(map[status.Group][]status.Code)
	for key, value := range retry.ChannelClientRetryableCodes {
		opts.RetryableCodes[key] = value
	}
	ccCodes, err := c.txnSnapConfig.CCErrorRetryableCodes()
	if err != nil {
		logger.Warnf("Could not parse CC error retry args: %s", err)
	}
	for _, code := range ccCodes {
		addRetryCode(opts.RetryableCodes, status.ChaincodeStatus, status.Code(code))
	}

	addRetryCode(opts.RetryableCodes, status.ClientStatus, status.NoPeersFound)

	return opts
}

func (c *clientImpl) updateLogLevel(configBacked core.ConfigBackend) errors.Error {
	logLevel := lookup.New(configBacked).GetString("txnsnap.loglevel")
	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	level, err := logging.LogLevel(logLevel)
	if err != nil {
		return errors.WithMessage(errors.InitializeLoggingError, err, "Error initializing log level")
	}

	logging.SetLevel("txnsnap", level)
	logger.Debugf("Txnsnap logging initialized. Log level: %s", logLevel)

	return nil
}

// addRetryCode adds the given group and code to the given map
func addRetryCode(codes map[status.Group][]status.Code, group status.Group, code status.Code) {
	g, exists := codes[group]
	if !exists {
		g = []status.Code{}
	}
	codes[group] = append(g, code)
}
