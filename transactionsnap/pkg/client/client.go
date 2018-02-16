/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	fab "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/peer"
	selection "github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn/selection/dynamicselection"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-txn/txnhandler"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	apisdk "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/api"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk/factory/defsvc"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	protosMSP "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	dynamicDiscovery "github.com/securekey/fabric-snaps/membershipsnap/pkg/discovery/local/provider"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/client/factories"
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
	clientConfig   apiconfig.Config
	channelClient  chclient.ChannelClient
	resourceClient fab.Resource
	channel        fab.Channel
	session        apisdk.SessionContext
}

// DynamicProviderFactory is configured with dynamic discovery provider and dynamic selection provider
type DynamicProviderFactory struct {
	defsvc.ProviderFactory
	ChannelUsers []selection.ChannelUser
}

// NewDiscoveryProvider returns a new implementation of dynamic discovery provider
func (f *DynamicProviderFactory) NewDiscoveryProvider(config apiconfig.Config) (fab.DiscoveryProvider, error) {
	return dynamicDiscovery.New(config), nil
}

// NewSelectionProvider returns a new implementation of dynamic selection provider
func (f *DynamicProviderFactory) NewSelectionProvider(config apiconfig.Config) (fab.SelectionProvider, error) {
	return selection.NewSelectionProvider(config, f.ChannelUsers, nil)
}

// CustomConfig override client config
type CustomConfig struct {
	apiconfig.Config
	localPeer           *api.PeerConfig
	localPeerTLSCertPem []byte
}

// ChannelPeers returns the channel peers configuration
func (c *CustomConfig) ChannelPeers(name string) ([]apiconfig.ChannelPeer, error) {
	networkPeer := apiconfig.NetworkPeer{PeerConfig: apiconfig.PeerConfig{URL: fmt.Sprintf("%s:%d", c.localPeer.Host,
		c.localPeer.Port), TLSCACerts: apiconfig.TLSConfig{Pem: string(c.localPeerTLSCertPem)}}, MspID: string(c.localPeer.MSPid)}
	peer := apiconfig.ChannelPeer{PeerChannelConfig: apiconfig.PeerChannelConfig{EndorsingPeer: true,
		ChaincodeQuery: true, LedgerQuery: true, EventSource: true}, NetworkPeer: networkPeer}
	return []apiconfig.ChannelPeer{peer}, nil
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
		if org.MspID == string(localPeer.MSPid) {
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
		channelUser := selection.ChannelUser{ChannelID: channelID, UserName: txnSnapUser, OrgName: orgname}
		serviceProviderFactory = &DynamicProviderFactory{ChannelUsers: []selection.ChannelUser{channelUser}}
	}

	sdk, err := fabsdk.New(NewCustomConfigProvider(clientConfig, localPeer, c.txnSnapConfig.GetTLSCertPem()),
		fabsdk.WithContextPkg(&factories.CredentialManagerProviderFactory{CryptoPath: c.txnSnapConfig.GetMspConfigPath()}),
		fabsdk.WithCorePkg(&factories.CustomCorePkg{ProviderName: cryptoProvider}),
		fabsdk.WithServicePkg(serviceProviderFactory))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	// new client
	sdkClient := sdk.NewClient(fabsdk.WithUser(txnSnapUser), fabsdk.WithOrg(orgname))

	// get user session
	session, err := sdkClient.Session()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "failed getting admin user session for org")
	}

	// get resource client
	resourceClient, err := sdk.FabricProvider().CreateResourceClient(session)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "NewResourceClient failed")
	}
	if resourceClient == nil {
		return errors.New(errors.GeneralError, "resource client is nil")
	}

	// Channel client is used to query and execute transactions
	chClient, err := sdkClient.Channel(channelID)
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Failed to create new channel(%v) client %v", channelID, err)
	}
	if chClient == nil {
		return errors.New(errors.GeneralError, "channel client is nil")
	}

	// Get channel
	chService, err := sdkClient.ChannelService(channelID)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Failed to get channel service")
	}

	channel, err := chService.Channel()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Failed to get channel from channel service")
	}
	if channel == nil {
		return errors.New(errors.GeneralError, "channel is nil")
	}

	c.resourceClient = resourceClient
	c.channelClient = chClient
	c.channel = channel
	c.clientConfig = clientConfig
	c.session = session
	return nil
}

//NewCustomConfigProvider return custom config provider
func NewCustomConfigProvider(config apiconfig.Config, localPeer *api.PeerConfig, localPeerTLSCertPem []byte) apiconfig.ConfigProvider {
	return func() (apiconfig.Config, error) {
		return &CustomConfig{Config: config, localPeer: localPeer, localPeerTLSCertPem: localPeerTLSCertPem}, nil
	}
}

func (c *clientImpl) EndorseTransaction(endorseRequest *api.EndorseTxRequest) ([]*fab.TransactionProposalResponse, error) {
	logger.Debugf("EndorseTransaction with endorseRequest %v", endorseRequest)

	targets := peer.PeersToTxnProcessors(endorseRequest.Targets)
	args := make([][]byte, 0)
	for _, value := range endorseRequest.Args[1:] {
		args = append(args, []byte(value))
	}

	customQueryHandler := handler.NewPeerFilterHandler(endorseRequest.PeerFilter,
		txnhandler.NewEndorsementHandler(
			txnhandler.NewEndorsementValidationHandler(
				txnhandler.NewSignatureValidationHandler(),
			),
		),
	)

	response, err := c.channelClient.InvokeHandler(customQueryHandler, chclient.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args}, chclient.WithProposalProcessor(targets...))

	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "InvokeHandler Query failed")
	}
	return response.Responses, nil
}

func (c *clientImpl) CommitTransaction(endorseRequest *api.EndorseTxRequest, callback api.EndorsedCallback) ([]*fab.TransactionProposalResponse, error) {
	logger.Debugf("CommitTransaction with endorseRequest %v", endorseRequest)
	targets := peer.PeersToTxnProcessors(endorseRequest.Targets)
	args := make([][]byte, 0)
	for _, value := range endorseRequest.Args[1:] {
		args = append(args, []byte(value))
	}

	customExecuteHandler := handler.NewPeerFilterHandler(endorseRequest.PeerFilter,
		txnhandler.NewEndorsementHandler(
			txnhandler.NewEndorsementValidationHandler(
				txnhandler.NewSignatureValidationHandler(
					handler.NewCheckForCommitHandler(endorseRequest.RWSetIgnoreNameSpace, callback,
						handler.NewLocalEventCommitHandler(),
					),
				),
			),
		),
	)

	resp, err := c.channelClient.InvokeHandler(customExecuteHandler, chclient.Request{ChaincodeID: endorseRequest.ChaincodeID, Fcn: endorseRequest.Args[0],
		Args: args}, chclient.WithProposalProcessor(targets...))

	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "InvokeHandler execute failed")
	}
	return resp.Responses, nil
}

func (c *clientImpl) QueryChannels(peer fab.Peer) ([]string, error) {
	responses, err := c.resourceClient.QueryChannels(peer)

	if err != nil {
		return nil, errors.Errorf(errors.GeneralError, "Error querying channels on peer %+v : %s", peer, err)
	}
	channels := []string{}

	for _, response := range responses.GetChannels() {
		channels = append(channels, response.ChannelId)
	}

	return channels, nil
}

func (c *clientImpl) VerifyTxnProposalSignature(proposalBytes []byte) error {

	if c.channel.MSPManager() == nil {
		return errors.New(errors.GeneralError, "GetMSPManager is nil")
	}
	msps, err := c.channel.MSPManager().GetMSPs()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "GetMSPs return error")
	}
	if len(msps) == 0 {
		return errors.New(errors.GeneralError, "MSPManager.GetMSPs is empty")
	}

	signedProposal := &pb.SignedProposal{}
	if err := proto.Unmarshal(proposalBytes, signedProposal); err != nil {
		return errors.Wrap(errors.GeneralError, err, "Unmarshal clientProposalBytes error")
	}

	creatorBytes, err := utils.GetCreatorFromSignedProposal(signedProposal)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "GetCreatorFromSignedProposal return error")
	}

	serializedIdentity := &protosMSP.SerializedIdentity{}
	if err := proto.Unmarshal(creatorBytes, serializedIdentity); err != nil {
		return errors.Wrap(errors.GeneralError, err, "Unmarshal creatorBytes error")
	}

	msp := msps[serializedIdentity.Mspid]
	if msp == nil {
		return errors.Errorf(errors.GeneralError, "MSP %s not found", serializedIdentity.Mspid)
	}

	creator, err := msp.DeserializeIdentity(creatorBytes)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "Failed to deserialize creator identity")
	}
	logger.Debugf("checkSignatureFromCreator info: creator is %s", creator.GetIdentifier())
	// ensure that creator is a valid certificate
	err = creator.Validate()
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "The creator certificate is not valid")
	}

	logger.Debugf("verifyTPSignature info: creator is valid")

	// validate the signature
	err = creator.Verify(signedProposal.ProposalBytes, signedProposal.Signature)
	if err != nil {
		return errors.Wrap(errors.GeneralError, err, "The creator's signature over the proposal is not valid")
	}

	logger.Debugf("VerifyTxnProposalSignature exists successfully")

	return nil
}

func (c *clientImpl) GetConfig() apiconfig.Config {
	return c.clientConfig
}
