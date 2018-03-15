/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	selection "github.com/hyperledger/fabric-sdk-go/pkg/client/common/selection/dynamicselection"
	coreApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/core"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/context/api/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config/endpoint"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	pb "github.com/hyperledger/fabric/protos/peer"
	dynamicDiscovery "github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/provider"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories"
	factoriesMsp "github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories/msp"

	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/handler"
	utils "github.com/securekey/fabric-snaps/transactionsnap/pkg/utils"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("txnsnap")

const (
	txnSnapUser = "Txn-Snap-User"
)

type clientImpl struct {
	sync.RWMutex
	txnSnapConfig  api.Config
	clientConfig   coreApi.Config
	channelClient  *channel.Client
	channelService fabApi.ChannelService
	channelID      string
}

// DynamicProviderFactory is configured with dynamic discovery provider and dynamic selection provider
type DynamicProviderFactory struct {
	defsvc.ProviderFactory
	ChannelUsers []selection.ChannelUser
}

// CreateDiscoveryProvider returns a new implementation of dynamic discovery provider
func (f *DynamicProviderFactory) CreateDiscoveryProvider(config coreApi.Config, fabPvdr fabApi.InfraProvider) (fabApi.DiscoveryProvider, error) {
	return dynamicDiscovery.New(config), nil
}

// CreateSelectionProvider returns a new implementation of dynamic selection provider
func (f *DynamicProviderFactory) CreateSelectionProvider(config coreApi.Config) (fabApi.SelectionProvider, error) {
	return selection.New(config, f.ChannelUsers)
}

// CustomConfig override client config
type CustomConfig struct {
	coreApi.Config
	localPeer           *api.PeerConfig
	localPeerTLSCertPem []byte
}

// ChannelPeers returns the channel peers configuration
// TODO this is a workaround.
// Currently there is no way to pass in a set of target peers to the selection provider.
func (c *CustomConfig) ChannelPeers(name string) ([]coreApi.ChannelPeer, error) {

	networkPeer := coreApi.NetworkPeer{PeerConfig: coreApi.PeerConfig{URL: fmt.Sprintf("%s:%d", c.localPeer.Host,
		c.localPeer.Port), TLSCACerts: endpoint.TLSConfig{Pem: string(c.localPeerTLSCertPem)}}, MSPID: string(c.localPeer.MSPid)}
	peer := coreApi.ChannelPeer{PeerChannelConfig: coreApi.PeerChannelConfig{EndorsingPeer: true,
		ChaincodeQuery: true, LedgerQuery: true, EventSource: true}, NetworkPeer: networkPeer}
	logger.Debugf("ChannelPeers return %v", peer)
	return []coreApi.ChannelPeer{peer}, nil
}

var cachedClient map[string]*clientImpl

//var client *clientImpl
var clientMutex sync.RWMutex
var once sync.Once

// GetInstance returns a singleton instance of the fabric client
func GetInstance(channelID string, txnSnapConfig api.Config, serviceProviderFactory apisdk.ServiceProviderFactory) (api.Client, error) {
	once.Do(func() {
		logger.Debugf("Client cache was created")
		cachedClient = make(map[string]*clientImpl)
	})
	if channelID == "" {
		return nil, errors.New(errors.GeneralError, "Channel is required")
	}

	clientMutex.RLock()
	c := cachedClient[channelID] //client from cache
	clientMutex.RUnlock()
	if c != nil {
		return c, nil
	}
	clientMutex.Lock()
	defer clientMutex.Unlock()

	c = &clientImpl{txnSnapConfig: txnSnapConfig}
	err := c.initialize(channelID, serviceProviderFactory)
	if err != nil {
		return nil, err
	}

	//put client into cache
	cachedClient[channelID] = c
	return c, nil
}

func (c *clientImpl) initialize(channelID string, serviceProviderFactory apisdk.ServiceProviderFactory) error {
	// Get local peer
	localPeer, err := c.txnSnapConfig.GetLocalPeer()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "GetLocalPeer return error")
	}

	// Get client config
	clientConfig, err := config.FromRaw(c.txnSnapConfig.GetConfigBytes(), "yaml")()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "get client config return error")
	}

	// Get org name
	nconfig, err := clientConfig.NetworkConfig()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Failed to get network config")
	}
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

	sdk, err := fabsdk.New(NewCustomConfigProvider(clientConfig, localPeer, c.txnSnapConfig.GetTLSCertPem()),
		fabsdk.WithCorePkg(&factories.CustomCorePkg{ProviderName: cryptoProvider}),
		fabsdk.WithServicePkg(serviceProviderFactory),
		fabsdk.WithMSPPkg(&factoriesMsp.CustomMspPkg{CryptoPath: c.txnSnapConfig.GetMspConfigPath()}))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	// new client
	chContextProv := sdk.ChannelContext(channelID, fabsdk.WithUser(txnSnapUser), fabsdk.WithOrg(orgname))

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
	// Get channel service
	chService := chContext.ChannelService()
	if chService == nil {
		return errors.New(errors.GeneralError, "channel service is nil")
	}

	c.channelClient = chClient
	c.channelService = chService
	c.channelID = channelID
	c.clientConfig = clientConfig
	return nil
}

//NewCustomConfigProvider return custom config provider
func NewCustomConfigProvider(config coreApi.Config, localPeer *api.PeerConfig, localPeerTLSCertPem []byte) coreApi.ConfigProvider {
	return func() (coreApi.Config, error) {
		return &CustomConfig{Config: config, localPeer: localPeer, localPeerTLSCertPem: localPeerTLSCertPem}, nil
	}
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

	response, err := c.channelClient.InvokeHandler(customQueryHandler, channel.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args, TransientMap: endorseRequest.TransientData}, channel.WithTargets(targets...),
		channel.WithTimeout(coreApi.Execute, c.txnSnapConfig.GetHandlerTimeout()), channel.WithTargetFilter(endorseRequest.PeerFilter))

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

	resp, err := c.channelClient.InvokeHandler(customExecuteHandler, channel.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args, TransientMap: endorseRequest.TransientData}, channel.WithTargets(targets...),
		channel.WithTimeout(coreApi.Execute, c.txnSnapConfig.GetHandlerTimeout()), channel.WithTargetFilter(endorseRequest.PeerFilter))

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
	membership, err := c.channelService.Membership()
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

func (c *clientImpl) GetConfig() coreApi.Config {
	return c.clientConfig
}
