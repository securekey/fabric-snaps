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

	sdkApi "github.com/hyperledger/fabric-sdk-go/api"
	sdkFabApi "github.com/hyperledger/fabric-sdk-go/def/fabapi"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	pb "github.com/hyperledger/fabric/protos/peer"
	logging "github.com/op/go-logging"
	sconfig "github.com/securekey/fabric-snaps/api/config"
	config "github.com/securekey/fabric-snaps/pkg/snaps/transactionsnap/config"
)

var logger = logging.MustGetLogger("transaction-snap")

const (
	snapUser = "Snap-User"
)

// Client is a wrapper interface around the fabric client
// It enables multithreaded access to the client
type Client interface {
	// NewChannel registers a channel object with the fabric client
	// this object represents a channel on the fabric network
	// @param {string} name of the channel
	// @returns {Channel} channel object
	// @returns {error} error, if any
	NewChannel(string) (sdkApi.Channel, error)

	// GetChannel returns a channel object that has been added to the fabric client
	// @param {string} name of the channel
	// @returns {Channel} channel that was requested
	// @returns {error} error, if any
	GetChannel(string) (sdkApi.Channel, error)

	// EndorseTransaction request endorsement from the peers on this channel
	// for a transaction with the given parameters
	// @param {Channel} channel on which we want to transact
	// @param {string} chaincodeID identifies the chaincode to invoke
	// @param {[]string} args to pass to the chaincode
	// @param {[]Peer} (optional) targets for transaction
	// @param {map[string][]byte} transientData map
	// @returns {[]TransactionProposalResponse} responses from endorsers
	// @returns {error} error, if any
	EndorseTransaction(sdkApi.Channel, string, []string, map[string][]byte,
		[]sdkApi.Peer) ([]*sdkApi.TransactionProposalResponse, error)

	// CommitTransaction submits the given endorsements on the specified channel for
	// commit
	// @param {Channel} channel on which the transaction is taking place
	// @param {[]TransactionProposalResponse} responses from endorsers
	// @param {bool} register for Tx event
	// @returns {error} error, if any
	CommitTransaction(sdkApi.Channel, []*sdkApi.TransactionProposalResponse, bool) error

	// QueryChannels joined by the given peer
	// @param {Peer} The peer to query
	// @returns {[]string} list of channels
	// @returns {error} error, if any
	QueryChannels(config.PeerConfig) ([]string, error)

	// SetSelectionService is used to inject a selection service for testing
	// @param {SelectionService} SelectionService
	SetSelectionService(SelectionService)

	// GetSelectionService returns the SelectionService
	GetSelectionService() SelectionService

	//GetEventHub returns the EventHub
	// @returns {EventHub} EventHub
	// @returns {error} error, if any
	GetEventHub() (sdkApi.EventHub, error)

	// Hash message
	// @param {[]byte} message to hash
	// @returns {[[]byte} hash
	// @returns {error} error, if any
	Hash([]byte) ([]byte, error)

	// InitializeChannel initializes the given channel
	// @param {Channel} Channel that needs to be initialized
	// @returns {error} error, if any
	InitializeChannel(channel sdkApi.Channel) error

	// GetConfig get client config
	// @returns {Config} config
	GetConfig() sdkApi.Config
}

type clientImpl struct {
	sync.RWMutex
	client           sdkApi.FabricClient
	selectionService SelectionService
}

var client *clientImpl
var once sync.Once

// GetInstance returns a singleton instance of the fabric client
func GetInstance() (Client, error) {
	var err error
	once.Do(func() {
		client = &clientImpl{selectionService: NewSelectionService()}
		initError := client.initialize()
		if initError != nil {
			err = fmt.Errorf("Error initializing fabric client: %s", initError)
		}
	})

	if err != nil {
		fmt.Printf("Error in get instance on fabric client:%v", err)
		return nil, err
	}

	return client, nil
}

//NewChannel creates new channel
func (c *clientImpl) NewChannel(name string) (sdkApi.Channel, error) {
	c.RLock()
	chain := c.client.GetChannel(name)
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
	ordererConfig, err := c.client.GetConfig().RandomOrdererConfig()
	if err != nil {
		return nil, fmt.Errorf("GetRandomOrdererConfig return error: %s", err)
	}
	orderer, err := sdkFabApi.NewOrderer(fmt.Sprintf("%s:%d",
		ordererConfig.Host, ordererConfig.Port), config.GetConfigPath(ordererConfig.TLS.Certificate), "", c.client.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("Error adding orderer: %s", err)
	}
	channel.AddOrderer(orderer)

	return channel, nil
}

//GetChannel by name
func (c *clientImpl) GetChannel(name string) (sdkApi.Channel, error) {
	c.RLock()
	defer c.RUnlock()

	channel := c.client.GetChannel(name)
	if channel == nil {
		return nil, fmt.Errorf("Channel %s has not been created", name)
	}

	return channel, nil
}

//EndorseTransaction sends proposal to endorser
func (c *clientImpl) EndorseTransaction(channel sdkApi.Channel, chaincodeID string,
	args []string, transientData map[string][]byte, targets []sdkApi.Peer) (
	[]*sdkApi.TransactionProposalResponse, error) {
	var peers []sdkApi.Peer
	var err error

	if targets == nil {
		// Select endorsers
		peers, err = c.selectionService.GetEndorsersForChaincode(channel.Name(),
			chaincodeID)
		if err != nil {
			return nil, fmt.Errorf("Error selecting endorsers: %s", err)
		}
	} else {
		peers = targets
	}

	c.RLock()
	defer c.RUnlock()

	logger.Debugf("Requesting endorsements from %s, on channel %s",
		chaincodeID, channel.Name())

	proposal, err := channel.CreateTransactionProposal(chaincodeID,
		channel.Name(), args, true, transientData)
	if err != nil {
		return nil, fmt.Errorf("Error creating transaction proposal: %s", err)
	}
	// TODO: Retry? Parameter is currently ignored by the client
	responses, err := channel.SendTransactionProposal(proposal, 0, peers)
	if err != nil {
		return nil, fmt.Errorf("Error sending transaction proposal: %s", err)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("Did not receive any endorsements")
	}
	var validResponses []*sdkApi.TransactionProposalResponse
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

//CommitTransaction  is called when transaction is commited
func (c *clientImpl) CommitTransaction(channel sdkApi.Channel,
	responses []*sdkApi.TransactionProposalResponse, registerTxEvent bool) error {
	c.RLock()
	defer c.RUnlock()

	logger.Debugf("Sending transaction for commit")

	transaction, err := channel.CreateTransaction(responses)
	if err != nil {
		return fmt.Errorf("Error creating transaction: %s", err)
	}
	done := make(chan bool)
	fail := make(chan error)
	txID := responses[0].Proposal.TransactionID
	if registerTxEvent {
		peer, err := c.selectionService.GetPeerForEvents(channel.Name())
		if err != nil {
			return fmt.Errorf("Error selecting peer: %s", err)
		}
		eventHub, err := sdkFabApi.NewEventHub(c.client)
		if err != nil {
			return fmt.Errorf("Failed sdkFabricTxn.GetDefaultImplEventHub() [%v]", err)
		}
		eventHub.SetPeerAddr(fmt.Sprintf("%s:%d", peer.EventHost,
			peer.EventPort), config.GetTLSRootCertPath(), "")
		if err := eventHub.Connect(); err != nil {
			return fmt.Errorf("Failed eventHub.Connect() [%v]", err)
		}
		defer eventHub.Disconnect()
		done, fail = c.registerTxEvent(txID, eventHub)
	}
	transactionResponses, err := channel.SendTransaction(transaction)
	if err != nil {
		return fmt.Errorf("Error sending transaction: %s", err)
	}

	var errorResponses []string
	for _, transactionResponse := range transactionResponses {
		if transactionResponse.Err != nil {
			errorResponses = append(errorResponses, transactionResponse.Err.Error())
		}
	}

	if len(errorResponses) > 0 {
		return fmt.Errorf(strings.Join(errorResponses, "\n"))
	}

	if registerTxEvent {
		select {
		case <-done:
		case <-fail:
			return fmt.Errorf("SendTransaction Error received from eventhub for txid(%s) error(%v)", txID, fail)
		case <-time.After(time.Second * 30):
			return fmt.Errorf("SendTransaction Didn't receive tx event for txid(%s)", txID)
		}
	}

	return nil
}

//QueryChannels joined by the given peer
func (c *clientImpl) QueryChannels(peer config.PeerConfig) ([]string, error) {
	p, err := sdkFabApi.NewPeer(fmt.Sprintf("%s:%d", peer.Host, peer.Port),
		config.GetTLSRootCertPath(), "", c.client.GetConfig())
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

//SetSelectionService is used to inject a selection service for testing
func (c *clientImpl) SetSelectionService(service SelectionService) {
	c.Lock()
	defer c.Unlock()
	c.selectionService = service
}

//GetSelectionService is used to get a selection service for testing
func (c *clientImpl) GetSelectionService() SelectionService {
	return c.selectionService
}

//GetEventHub returns the EventHub
func (c *clientImpl) GetEventHub() (sdkApi.EventHub, error) {
	return sdkFabApi.NewEventHub(c.client)
}

//InitializeChannel initializes the given channel
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
		return fmt.Errorf("Error initializing new channel: %s", err)
	}

	return nil
}

func (c *clientImpl) initialize() error {
	sconfig.Init("")
	clientConfig, err := sdkFabApi.NewConfig(sconfig.GetConfigPath("") + "/config.yaml")
	logger.Debug("clientConfig path", clientConfig)

	if err != nil {
		return fmt.Errorf("Error initializaing config: %s", err)
	}
	clientConfig.CSPConfig()
	err = factory.InitFactories(&factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily: clientConfig.SecurityAlgorithm(),
			SecLevel:   clientConfig.SecurityLevel(),
			FileKeystore: &factory.FileKeystoreOpts{
				KeyStorePath: clientConfig.KeyStorePath(),
			},
			Ephemeral: true,
		},
	})
	if err != nil {
		return fmt.Errorf("Failed getting ephemeral software-based BCCSP [%s]", err)
	}
	config.Init("")
	localPeer, err := config.GetLocalPeer()
	if err != nil {
		return fmt.Errorf("GetLocalPeer return error [%v]", err)
	}
	cryptoSuite := bccspFactory.GetDefault()
	user, err := sdkFabApi.NewPreEnrolledUser(clientConfig,
		config.GetEnrolmentKeyPath(), config.GetEnrolmentCertPath(), snapUser, string(localPeer.MSPid), cryptoSuite)
	if err != nil {
		return fmt.Errorf("Failed NewClientWithPreEnrolledUser() [%s]", err)
	}
	client, err := sdkFabApi.NewClient(user, true, "", clientConfig)
	if err != nil {
		return fmt.Errorf("Failed NewClient() [%s]", err)
	}
	c.client = client

	return nil
}

//Hash calculates hash of message
func (c *clientImpl) Hash(message []byte) (hash []byte, err error) {
	return c.client.GetCryptoSuite().Hash(message, &bccsp.SHAOpts{})
}

//GetConfig get client config
func (c *clientImpl) GetConfig() sdkApi.Config {
	return c.client.GetConfig()
}

// RegisterTxEvent registers on the given eventhub for the give transaction
// returns a boolean channel which receives true when the event is complete
// and an error channel for errors
func (c *clientImpl) registerTxEvent(txID string, eventHub sdkApi.EventHub) (chan bool, chan error) {
	done := make(chan bool)
	fail := make(chan error)

	eventHub.RegisterTxEvent(txID, func(txId string, errorCode pb.TxValidationCode, err error) {
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
