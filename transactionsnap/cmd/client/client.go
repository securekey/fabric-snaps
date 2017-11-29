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
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"

	protosMSP "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/msp"
	sdkpb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/bccsp"
	pb "github.com/hyperledger/fabric/protos/peer"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/securekey/fabric-snaps/transactionsnap/api"
	utils "github.com/securekey/fabric-snaps/transactionsnap/cmd/utils"
)

var module = "transaction-fabric-client"
var logger = logging.NewLogger(module)

const (
	txnSnapUser = "Txn-Snap-User"
)

type clientImpl struct {
	sync.RWMutex
	client           sdkApi.FabricClient
	selectionService api.SelectionService
	config           api.Config
}

var client *clientImpl
var once sync.Once

// GetInstance returns a singleton instance of the fabric client
func GetInstance(config api.Config) (api.Client, error) {
	var err error
	once.Do(func() {
		client = &clientImpl{selectionService: NewSelectionService(config), config: config}
		initError := client.initialize()
		if initError != nil {
			err = fmt.Errorf("Error initializing fabric client: %s", initError)
		}
	})

	if err != nil {
		return nil, err
	}

	return client, nil
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
		return nil, fmt.Errorf("Error creating new channel: %s", err)
	}
	ordererConfig, err := c.client.Config().RandomOrdererConfig()
	if err != nil {
		return nil, fmt.Errorf("GetRandomOrdererConfig return error: %s", err)
	}

	orderer, err := sdkFabApi.NewOrderer(ordererConfig.URL, ordererConfig.TLSCACerts.Path, "", c.client.Config())
	if err != nil {
		return nil, fmt.Errorf("Error adding orderer: %s", err)
	}
	channel.AddOrderer(orderer)

	return channel, nil
}

func (c *clientImpl) GetChannel(name string) (sdkApi.Channel, error) {
	c.RLock()
	defer c.RUnlock()

	channel := c.client.Channel(name)
	if channel == nil {
		return nil, fmt.Errorf("Channel %s has not been created", name)
	}

	return channel, nil
}

func (c *clientImpl) EndorseTransaction(channel sdkApi.Channel, chaincodeID string,
	args []string, transientData map[string][]byte, targets []sdkApi.Peer, ccIDsForEndorsement []string) (
	[]*apitxn.TransactionProposalResponse, error) {
	var peers []sdkApi.Peer
	var processors []apitxn.ProposalProcessor
	var err error

	if targets == nil {
		if len(ccIDsForEndorsement) == 0 {
			ccIDsForEndorsement = append(ccIDsForEndorsement, chaincodeID)
		}
		// Select endorsers
		peers, err = c.selectionService.GetEndorsersForChaincode(channel.Name(),
			ccIDsForEndorsement...)
		if err != nil {
			return nil, fmt.Errorf("Error selecting endorsers: %s", err)
		}
	} else {
		peers = targets
	}

	for _, peer := range peers {
		logger.Debugf("Target peer %v", peer.URL())
		processors = append(processors, apitxn.ProposalProcessor(peer))
	}

	c.RLock()
	defer c.RUnlock()

	logger.Debugf("Requesting endorsements from %s, on channel %s",
		chaincodeID, channel.Name())

	if len(args) == 0 {
		return nil, fmt.Errorf(
			"Args cannot be empty. Args[0] is expected to be the function name")
	}

	request := apitxn.ChaincodeInvokeRequest{
		Targets:      processors,
		Fcn:          args[0],
		Args:         utils.GetByteArgs(args[1:]),
		TransientMap: transientData,
		ChaincodeID:  chaincodeID,
	}

	responses, _, err := channel.SendTransactionProposal(request)
	if err != nil {
		return nil, fmt.Errorf("Error sending transaction proposal: %s", err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("Did not receive any endorsements")
	}
	var validResponses []*apitxn.TransactionProposalResponse
	var errorCount int
	var errorResponses []string
	for _, response := range responses {
		if response.Err != nil {
			errorCount++
			errorResponses = append(errorResponses, response.Err.Error())
		} else {
			validResponses = append(validResponses, response)
		}
	}

	if errorCount == len(responses) {
		return nil, fmt.Errorf(strings.Join(errorResponses, "\n"))
	}

	return validResponses, nil
}

func (c *clientImpl) CommitTransaction(channel sdkApi.Channel,
	responses []*apitxn.TransactionProposalResponse, registerTxEvent bool, registerTxEventTimeout time.Duration) error {
	c.RLock()
	defer c.RUnlock()

	logger.Debugf("Sending transaction for commit")

	transaction, err := channel.CreateTransaction(responses)
	if err != nil {
		return fmt.Errorf("Error creating transaction: %s", err)
	}
	done := make(chan bool)
	fail := make(chan error)
	txID := transaction.Proposal.TxnID
	if registerTxEvent {
		localPeer, err := c.config.GetLocalPeer()
		if err != nil {
			return fmt.Errorf("GetLocalPeer return error [%v]", err)
		}
		eventHub, err := sdkFabApi.NewEventHub(c.client)
		if err != nil {
			return fmt.Errorf("Failed sdkFabricTxn.GetDefaultImplEventHub() [%v]", err)
		}
		eventHub.SetPeerAddr(fmt.Sprintf("%s:%d", localPeer.EventHost, localPeer.EventPort), "", "")
		if err := eventHub.Connect(); err != nil {
			return fmt.Errorf("Failed eventHub.Connect() [%v]", err)
		}
		defer eventHub.Disconnect()
		done, fail = c.registerTxEvent(txID, eventHub)
	}
	resp, err := channel.SendTransaction(transaction)
	if err != nil {
		return fmt.Errorf("Error sending transaction: %s", err)
	}

	if resp.Err != nil {
		return fmt.Errorf("Error sending transaction: %s", resp.Err.Error())
	}

	if registerTxEvent {
		select {
		case <-done:
		case <-fail:
			return fmt.Errorf("SendTransaction Error received from eventhub for txid(%s) error(%v)", txID.ID, fail)
		case <-time.After(time.Second * registerTxEventTimeout):
			return fmt.Errorf("SendTransaction Didn't receive tx event for txid(%s)", txID.ID)
		}
	}

	return nil
}

func (c *clientImpl) QueryChannels(peer api.PeerConfig) ([]string, error) {
	p, err := sdkFabApi.NewPeer(fmt.Sprintf("%s:%d", peer.Host, peer.Port),
		c.config.GetTLSRootCertPath(), "", c.client.Config())
	if err != nil {
		return nil, fmt.Errorf("Error creating peer: %s", err)
	}
	responses, err := c.client.QueryChannels(p)

	if err != nil {
		return nil, fmt.Errorf("Error querying channels on peer %+v : %s", peer, err)
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
		return fmt.Errorf("Channel %s GetMSPManager is nil", channel.Name())
	}
	msps, err := channel.MSPManager().GetMSPs()
	if err != nil {
		return fmt.Errorf("GetMSPs return error:%v", err)
	}
	if len(msps) == 0 {
		return fmt.Errorf("Channel %s MSPManager.GetMSPs is empty", channel.Name())
	}

	signedProposal := &pb.SignedProposal{}
	if err := proto.Unmarshal(proposalBytes, signedProposal); err != nil {
		return fmt.Errorf("Unmarshal clientProposalBytes error %v", err)
	}

	creatorBytes, err := utils.GetCreatorFromSignedProposal(signedProposal)
	if err != nil {
		return fmt.Errorf("GetCreatorFromSignedProposal return  error %v", err)
	}

	serializedIdentity := &protosMSP.SerializedIdentity{}
	if err := proto.Unmarshal(creatorBytes, serializedIdentity); err != nil {
		return fmt.Errorf("Unmarshal creatorBytes error %v", err)
	}

	msp := msps[serializedIdentity.Mspid]
	if msp == nil {
		return fmt.Errorf("MSP %s not found", serializedIdentity.Mspid)
	}

	creator, err := msp.DeserializeIdentity(creatorBytes)
	if err != nil {
		return fmt.Errorf("Failed to deserialize creator identity, err %s", err)
	}
	logger.Debugf("checkSignatureFromCreator info: creator is %s", creator.GetIdentifier())
	// ensure that creator is a valid certificate
	err = creator.Validate()
	if err != nil {
		return fmt.Errorf("The creator certificate is not valid, err %s", err)
	}

	logger.Debugf("verifyTPSignature info: creator is valid")

	// validate the signature
	err = creator.Verify(signedProposal.ProposalBytes, signedProposal.Signature)
	if err != nil {
		return fmt.Errorf("The creator's signature over the proposal is not valid, err %s", err)
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
	return sdkFabApi.NewEventHub(c.client)
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
		return fmt.Errorf("Error initializing new channel: %s", err.Error())
	}
	// Channel initialized. Add MSP roots to TLS cert pool.
	c.initializeTLSPool(channel)

	return nil
}

func (c *clientImpl) initializeTLSPool(channel sdkApi.Channel) error {
	globalCertPool, err := c.client.Config().TLSCACertPool("")
	if err != nil {
		return err
	}

	mspMap, err := channel.MSPManager().GetMSPs()
	if err != nil {
		return fmt.Errorf("Error getting MSPs for channel %s: %s",
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

func (c *clientImpl) initialize() error {

	sdkOptions := sdkFabApi.Options{
		ConfigFile:      c.config.GetConfigPath("") + "/config.yaml",
		ProviderFactory: &defaultCryptoSuiteProviderFactory{},
	}

	sdk, err := sdkFabApi.NewSDK(sdkOptions)
	if err != nil {
		panic(fmt.Sprintf("Failed to create new SDK: %s", err))
	}

	configProvider := sdk.ConfigProvider()
	if err != nil {
		return fmt.Errorf("Error getting config: %s", err)
	}

	localPeer, err := c.config.GetLocalPeer()
	if err != nil {
		return fmt.Errorf("GetLocalPeer return error [%v]", err)
	}

	//Find orgname matching localpeer mspID
	nconfig, err := configProvider.NetworkConfig()
	if err != nil {
		return fmt.Errorf("Failed to get network config %v", err)
	}
	var orgname string
	for name, org := range nconfig.Organizations {
		if org.MspID == string(localPeer.MSPid) {
			orgname = name
			break
		}
	}

	userSession, err := sdk.NewPreEnrolledUserSession(orgname, txnSnapUser)
	if err != nil {
		return fmt.Errorf("Failed to get NewPreEnrolledUserSession [%s]", err)
	}
	client, err := sdk.NewSystemClient(userSession)
	if err != nil {
		return fmt.Errorf("Failed to get new client [%s]", err)
	}
	c.client = client

	logger.Debugf("Done initializing client. Default log level: %s, fabric_sdk_go log level: %s, txn-snap-config log lelvel: %s", logging.GetLevel(""), logging.GetLevel("fabric_sdk_go"), logging.GetLevel("txn-snap-config"))

	return nil
}

func (c *clientImpl) Hash(message []byte) (hash []byte, err error) {
	return c.client.CryptoSuite().Hash(message, &bccsp.SHAOpts{})
}

func (c *clientImpl) GetConfig() sdkConfigApi.Config {
	return c.client.Config()
}

func (c *clientImpl) GetUser() sdkApi.User {
	return c.client.UserContext()
}

// RegisterTxEvent registers on the given eventhub for the give transaction
// returns a boolean channel which receives true when the event is complete
// and an error channel for errors
func (c *clientImpl) registerTxEvent(txID apitxn.TransactionID, eventHub sdkApi.EventHub) (chan bool, chan error) {
	done := make(chan bool)
	fail := make(chan error)

	eventHub.RegisterTxEvent(txID, func(txId string, errorCode sdkpb.TxValidationCode, err error) {
		if err != nil {
			logger.Debugf("Received error event for txid(%s)\n", txId)
			fail <- err
		} else {
			logger.Debugf("Received success event for txid(%s)\n", txId)
			done <- true
		}
	})

	return done, fail
}
