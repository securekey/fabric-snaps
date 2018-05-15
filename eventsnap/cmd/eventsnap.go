/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"sync"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/context"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/chconfig"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/comm"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/deliverclient"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/dispatcher"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/securekey/fabric-snaps/eventservice/pkg/localservice"
	"github.com/securekey/fabric-snaps/eventsnap/cmd/config"
	txnSnapClient "github.com/securekey/fabric-snaps/transactionsnap/pkg/client"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("eventsnap")
var delayStartChannelEventsDuration = 5 * time.Second

const (
	channelConfigCheckDuration = 1 * time.Second
)

var mutex sync.RWMutex

// eventSnap starts the Channel Event Server which allows clients to register
// for channel events. It also registers a local event service on the peer so that other
// snaps may register for channel events directly.
type eventSnap struct {
	// peerConfigPath is only set by unit tests
	peerConfigPath string
}

// New returns a new Event Snap
func New() shim.Chaincode {
	return &eventSnap{}
}

// Init initializes the Event Snap.
// The Event Server is registered when Init is called without a channel and
// a new, channel-specific event service is registered each time Init is called with a channel.
func (s *eventSnap) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Warnf("******** Init Event Snap on channel [%s]", stub.GetChannelID())

	channelID := stub.GetChannelID()
	if channelID == "" {
		return shim.Success(nil)
	}

	// Check the config periodically and start
	logger.Warnf("The event service will be started when configuration is available.", stub.GetChannelID())
	go s.delayStartChannelEvents(stub.GetChannelID())

	return shim.Success(nil)
}

// Invoke isn't implemented for this snap.
func (s *eventSnap) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("not implemented")
}

// startChannelEvents starts a new event relay and local event service for the given channel,
// and also starts a new Go routine that relays events to the channel server.
func (s *eventSnap) startChannelEvents(channelID string, esconfig *config.EventSnapConfig) error {
	existingLocalEventService := localservice.Get(channelID)
	if existingLocalEventService != nil {
		logger.Errorf("Event service already initialized for channel [%s]", channelID)
		return errors.Errorf(errors.GeneralError, "Event service already initialized for channel [%s]", channelID)
	}
	txnsnapser, err := txsnapservice.Get(channelID)
	if err != nil {
		logger.Errorf("Error getting txsnapservice: %s", err)
		return errors.WithMessage(errors.GeneralError, err, "Error getting txsnapservice")
	}

	client, err := txnSnapClient.GetInstanceWithLocalDiscovery(channelID, txnsnapser.Config)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "GetInstanceWithLocalDiscovery return error")
	}

	// Create a new channel event service which gets its events from the event relay
	eventClient, err := s.connectEventClient(client.GetContext(), channelID, esconfig)
	if err != nil {
		logger.Errorf("Error connecting event client: %s", err)
		return errors.WithMessage(errors.GeneralError, err, "Error connecting event client")
	}

	// Register the local event service for the channel
	if err := localservice.Register(channelID, eventClient); err != nil {
		logger.Errorf("Error registering local event service: %s", err)
		return errors.WithMessage(errors.GeneralError, err, "Error registering local event service")
	}

	return nil
}

func (s *eventSnap) delayStartChannelEvents(channelID string) {
	// wait for 5 seconds for delivery client to start
	time.Sleep(delayStartChannelEventsDuration)
	for {
		time.Sleep(channelConfigCheckDuration)
		logger.Debugf("Checking if EventSnap configuration is available for channel [%s]...", channelID)
		if config, err := config.New(channelID, s.peerConfigPath); err != nil {
			logger.Warnf("Error reading configuration: %s", err)
		} else if config != nil {
			if err := s.startChannelEvents(channelID, config); err != nil {
				logger.Errorf("Error starting channel events for channel [%s]: %s. Aborting!!!", channelID, err.Error())
			} else {
				logger.Infof("Channel events successfully started for channel [%s].", channelID)
			}
			return
		}
		logger.Debugf("... EventSnap configuration is not available yet for channel [%s]", channelID)
	}
}

// startEventService ...
func (s *eventSnap) connectEventClient(context context.Client, channelID string, config *config.EventSnapConfig) (fab.EventClient, error) {
	logger.Infof("Starting event service for channel [%s]...", channelID)

	// FIXME: This will go away with the latest SDK
	chConfig := chconfig.NewChannelCfg(channelID)

	eventClient, err := deliverclient.New(
		context, chConfig,
		comm.WithConnectTimeout(config.ResponseTimeout), // FIXME: Should be connect timeout
		dispatcher.WithEventConsumerBufferSize(config.EventConsumerBufferSize),
		dispatcher.WithEventConsumerTimeout(config.EventConsumerTimeout),
		client.WithMaxConnectAttempts(0),                      // Try connecting forever
		client.WithMaxReconnectAttempts(0),                    // Retry connecting forever
		client.WithTimeBetweenConnectAttempts(10*time.Second), // TODO: Make configurable
		client.WithResponseTimeout(config.ResponseTimeout),
		// deliverclient.WithBlockEvents(), // TODO: Use block events?
	)

	if err != nil {
		return nil, err
	}

	if err := eventClient.Connect(); err != nil {
		return nil, err
	}

	logger.Infof("... started event service for channel [%s]...", chConfig.ID())
	return eventClient, nil
}

func main() {
}
