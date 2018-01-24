/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	sdkConfigApi "github.com/hyperledger/fabric-sdk-go/api/apiconfig"
	sdkApi "github.com/hyperledger/fabric-sdk-go/api/apifabclient"
	apitxn "github.com/hyperledger/fabric-sdk-go/api/apitxn"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/status"
	"github.com/securekey/fabric-snaps/util/errors"

	sdkorderer "github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/orderer"

	protosMSP "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/bccsp"
	pb "github.com/hyperledger/fabric/protos/peer"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
	eventservice "github.com/securekey/fabric-snaps/eventservice/pkg/localservice"

	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/events"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	"github.com/securekey/fabric-snaps/transactionsnap/cmd/client/factories"
	utils "github.com/securekey/fabric-snaps/transactionsnap/cmd/utils"
)

var module = "txnsnap"
var logger = logging.NewLogger(module)

const (
	txnSnapUser = "Txn-Snap-User"
)

type clientImpl struct {
	sync.RWMutex
	client           sdkApi.Resource
	selectionService api.SelectionService
	config           api.Config
}

var cachedClient map[string]*clientImpl

//var client *clientImpl
var clientMutex sync.RWMutex
var once sync.Once

// GetInstance returns a singleton instance of the fabric client
func GetInstance(channelID string, config api.Config) (api.Client, error) {
	if channelID == "" {
		return nil, errors.New(errors.GeneralError, "Channel is required")
	}
	var c *clientImpl
	c.initializeCache()
	clientMutex.RLock()
	c = cachedClient[channelID] //client from cache
	clientMutex.RUnlock()

	if c != nil {
		return c, nil
	}

	clientMutex.Lock()
	defer clientMutex.Unlock()

	c = &clientImpl{selectionService: NewSelectionService(config), config: config}
	err := c.initialize(config.GetConfigBytes())
	if err != nil {
		logger.Errorf("Error initializing client: %s\n", err)
		return nil, errors.Wrap(errors.GeneralError, err, "error initializing fabric client")
	}

	if c.client == nil {
		logger.Errorf("Error: SDK client is nil!!!\n")
		return nil, errors.New(errors.GeneralError, "SDK client is nil")
	}
	//put client into cache
	cachedClient[channelID] = c
	return c, nil
}

//initializeCache used to initialize client cache
func (c *clientImpl) initializeCache() {
	once.Do(func() {
		logger.Debugf("Client cache was created")
		cachedClient = make(map[string]*clientImpl)
	})
}

func (c *clientImpl) NewChannel(name string) (sdkApi.Channel, error) {
	c.RLock()
	chain := c.client.Channel(name)
	c.RUnlock()

	if chain != nil {
		return chain, nil
	}

	c.Lock()
	defer c.Unlock()
	channel, err := c.client.NewChannel(name)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error creating new channel")
	}
	ordererConfig, err := c.client.Config().RandomOrdererConfig()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "GetRandomOrdererConfig return error")
	}

	opts, err := withOrdererOptions(ordererConfig)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "withOrdererOptions return error")
	}
	orderer, err := sdkorderer.New(c.client.Config(), opts...)

	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error adding orderer")
	}
	channel.AddOrderer(orderer)

	return channel, nil
}

func withOrdererOptions(ordererConfig *sdkConfigApi.OrdererConfig) ([]sdkorderer.Option, error) {
	opts := []sdkorderer.Option{}
	opts = append(opts, sdkorderer.WithURL(ordererConfig.URL))
	opts = append(opts, sdkorderer.WithServerName(""))

	ocert, err := ordererConfig.TLSCACerts.TLSCert()

	if err != nil {
		s, ok := status.FromError(err)
		// if error is other than EmptyCert, then it should not be ignored, else simply set TLS with no cert
		if !ok || s.Code != status.EmptyCert.ToInt32() {
			return nil, errors.Wrap(errors.GeneralError, err, "error getting orderer cert from the configs")
		}
	}
	if ocert != nil {
		opts = append(opts, sdkorderer.WithTLSCert(ocert))
	}

	return opts, nil
}

func (c *clientImpl) GetChannel(name string) (sdkApi.Channel, error) {
	c.RLock()
	defer c.RUnlock()

	channel := c.client.Channel(name)
	if channel == nil {
		return nil, errors.Errorf(errors.GeneralError, "Channel %s has not been created", name)
	}

	return channel, nil
}

func (c *clientImpl) EndorseTransaction(channel sdkApi.Channel, endorseRequest *api.EndorseTxRequest) (
	[]*apitxn.TransactionProposalResponse, error) {

	if len(endorseRequest.Args) == 0 {
		return nil, errors.Errorf(errors.GeneralError,
			"Args cannot be empty. Args[0] is expected to be the function name")
	}

	var peers []sdkApi.Peer
	var processors []apitxn.ProposalProcessor
	var err error

	var ccIDsForEndorsement []string
	if endorseRequest.Targets == nil {
		if len(endorseRequest.ChaincodeIDs) == 0 {
			ccIDsForEndorsement = append(ccIDsForEndorsement, endorseRequest.ChaincodeID)
		} else {
			ccIDsForEndorsement = endorseRequest.ChaincodeIDs
		}

		// Select endorsers
		remainingAttempts := c.config.GetEndorserSelectionMaxAttempts()
		logger.Infof("Attempting to get endorsers - [%d] attempts...", remainingAttempts)
		for len(peers) == 0 && remainingAttempts > 0 {
			peers, err = c.selectionService.GetEndorsersForChaincode(channel.Name(),
				endorseRequest.PeerFilter, ccIDsForEndorsement...)
			if err != nil {
				return nil, errors.WithMessage(errors.GeneralError, err, "error selecting endorsers")
			}
			if len(peers) == 0 {
				remainingAttempts--
				logger.Warnf("No endorsers. [%d] remaining attempts...", remainingAttempts)
				time.Sleep(c.config.GetEndorserSelectionInterval())
			}
		}

		if len(peers) == 0 {
			logger.Errorf("No suitable endorsers found for transaction.")
			return nil, errors.New(errors.GeneralError, "no suitable endorsers found for transaction")
		}
	} else {
		peers = endorseRequest.Targets
	}

	for _, peer := range peers {
		logger.Debugf("Target peer %v", peer.URL())
		processors = append(processors, apitxn.ProposalProcessor(peer))
	}

	c.RLock()
	defer c.RUnlock()

	logger.Debugf("Requesting endorsements from %s, on channel %s",
		endorseRequest.ChaincodeID, channel.Name())

	request := apitxn.ChaincodeInvokeRequest{
		Targets:      processors,
		Fcn:          endorseRequest.Args[0],
		Args:         utils.GetByteArgs(endorseRequest.Args[1:]),
		TransientMap: endorseRequest.TransientData,
		ChaincodeID:  endorseRequest.ChaincodeID,
	}

	// TODO: Replace this code with the GO SDK's ChannelClient
	responses, _, err := channel.SendTransactionProposal(request)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Error sending transaction proposal")
	}

	// TODO: Replace the following code with the GO SDK's endorsement validation logic
	if len(responses) == 0 {
		return nil, errors.New(errors.GeneralError, "Did not receive any endorsements")
	}
	var errorResponses []string
	for _, response := range responses {
		if response.Err != nil {
			errorResponses = append(errorResponses, response.Err.Error())
		}
	}
	if len(errorResponses) > 0 {
		return responses, errors.Errorf(errors.GeneralError, strings.Join(errorResponses, "\n"))
	}
	if len(responses) != len(processors) {
		return responses, errors.Errorf(errors.GeneralError, "only %d out of %d responses were received", len(responses), len(processors))
	}

	return responses, nil
}

func (c *clientImpl) CommitTransaction(channel sdkApi.Channel,
	responses []*apitxn.TransactionProposalResponse, registerTxEvent bool) error {
	c.RLock()
	defer c.RUnlock()

	transaction, err := channel.CreateTransaction(responses)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Error creating transaction")
	}
	logger.Debugf("Sending transaction [%s] for commit", transaction.Proposal.TxnID.ID)

	var txStatusEventCh <-chan *eventapi.TxStatusEvent
	txID := transaction.Proposal.TxnID
	if registerTxEvent {
		events := eventservice.Get(channel.Name())
		reg, eventch, err := events.RegisterTxStatusEvent(txID.ID)
		if err != nil {
			return errors.Wrapf(errors.GeneralError, err, "unable to register for TxStatus event for TxID [%s] on channel [%s]", txID, channel.Name())
		}
		defer events.Unregister(reg)
		txStatusEventCh = eventch
	}
	resp, err := channel.SendTransaction(transaction)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Error sending transaction")
	}

	if resp.Err != nil {
		return errors.WithMessage(errors.GeneralError, resp.Err, "Error sending transaction")
	}

	if registerTxEvent {
		select {
		case txStatusEvent := <-txStatusEventCh:
			if txStatusEvent.TxValidationCode != pb.TxValidationCode_VALID {
				return errors.Errorf(errors.GeneralError, "transaction [%s] did not commit successfully. Code: [%s]", txID.ID, txStatusEvent.TxValidationCode)
			}
			logger.Debugf("Transaction [%s] successfully committed", txID.ID)
		case <-time.After(c.config.GetCommitTimeout()):
			return errors.Errorf(errors.GeneralError, "SendTransaction Didn't receive tx event for txid(%s)", txID.ID)
		}
	}

	return nil
}

// /QueryChannels to query channels based on peer
func (c *clientImpl) QueryChannels(peer sdkApi.Peer) ([]string, error) {
	responses, err := c.client.QueryChannels(peer)

	if err != nil {
		return nil, errors.Errorf(errors.GeneralError, "Error querying channels on peer %+v : %s", peer, err)
	}
	channels := []string{}

	for _, response := range responses.GetChannels() {
		channels = append(channels, response.ChannelId)
	}

	return channels, nil
}

// Verify Transaction Proposal signature
func (c *clientImpl) VerifyTxnProposalSignature(channel sdkApi.Channel, proposalBytes []byte) error {
	if channel.MSPManager() == nil {
		return errors.Errorf(errors.GeneralError, "Channel %s GetMSPManager is nil", channel.Name())
	}
	msps, err := channel.MSPManager().GetMSPs()
	if err != nil {
		return errors.Errorf(errors.GeneralError, "GetMSPs return error:%v", err)
	}
	if len(msps) == 0 {
		return errors.Errorf(errors.GeneralError, "Channel %s MSPManager.GetMSPs is empty", channel.Name())
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

func (c *clientImpl) SetSelectionService(service api.SelectionService) {
	c.Lock()
	defer c.Unlock()
	c.selectionService = service
}

func (c *clientImpl) GetSelectionService() api.SelectionService {
	return c.selectionService
}

func (c *clientImpl) GetEventHub() (sdkApi.EventHub, error) {
	eventHub, err := events.NewEventHub(c.client)
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed to get NewEventHub")
	}
	return eventHub, err
}

func (c *clientImpl) InitializeChannel(channel sdkApi.Channel) error {
	c.RLock()
	isInitialized := channel.IsInitialized()
	c.RUnlock()
	if isInitialized {
		logger.Debug("Chain is initialized. Returning.")
		return nil
	}
	c.Lock()
	defer c.Unlock()

	err := channel.Initialize(nil)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Error initializing new channel")
	}
	// Channel initialized. Add MSP roots to TLS cert pool.
	err = c.initializeTLSPool(channel)
	if err != nil {
		return err
	}

	return nil
}

func (c *clientImpl) initializeTLSPool(channel sdkApi.Channel) error {
	globalCertPool, err := c.client.Config().TLSCACertPool()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Failed TLSCACertPool")
	}

	mspMap, err := channel.MSPManager().GetMSPs()
	if err != nil {
		return errors.Errorf(errors.GeneralError, "Error getting MSPs for channel %s: %v",
			channel.Name(), err)
	}

	for _, msp := range mspMap {
		for _, cert := range msp.GetTLSRootCerts() {
			globalCertPool.AppendCertsFromPEM(cert)
		}

		for _, cert := range msp.GetTLSIntermediateCerts() {
			globalCertPool.AppendCertsFromPEM(cert)
		}
	}

	c.client.Config().SetTLSCACertPool(globalCertPool)
	return nil
}

func (c *clientImpl) initialize(sdkConfig []byte) error {

	sdk, err := fabsdk.New(config.FromRaw(sdkConfig, "yaml"),
		fabsdk.WithContextPkg(&factories.CredentialManagerProviderFactory{CryptoPath: c.config.GetMspConfigPath()}),
		fabsdk.WithCorePkg(&factories.DefaultCryptoSuiteProviderFactory{}))
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	configProvider := sdk.ConfigProvider()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "Error getting config")
	}

	localPeer, err := c.config.GetLocalPeer()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "GetLocalPeer return error")
	}

	//Find orgname matching localpeer mspID
	nconfig, err := configProvider.NetworkConfig()
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

	userSession, err := sdk.NewClient(fabsdk.WithUser(txnSnapUser), fabsdk.WithOrg(orgname)).Session()
	if err != nil {
		return errors.Wrapf(errors.GeneralError, err, "failed getting user session for org %s", orgname)
	}
	client, err := sdk.FabricProvider().NewResourceClient(userSession.Identity())
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "NewResourceClient failed")
	}
	c.client = client

	logger.Debugf("Done initializing client. Default log level: %s, fabric_sdk_go log level: %s, txn-snap-config log lelvel: %s", logging.GetLevel(""), logging.GetLevel("fabric_sdk_go"), logging.GetLevel("txn-snap-config"))

	return nil
}

func (c *clientImpl) Hash(message []byte) (hash []byte, err error) {
	hash, err = c.client.CryptoSuite().Hash(message, &bccsp.SHAOpts{})
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "Failed Hash")
	}
	return hash, err
}

func (c *clientImpl) GetConfig() sdkConfigApi.Config {
	return c.client.Config()
}

func (c *clientImpl) GetSigningIdentity() sdkApi.IdentityContext {
	return c.client.IdentityContext()
}
