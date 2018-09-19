/*
   Copyright SecureKey Technologies Inc.
   This file contains software code that is the intellectual property of SecureKey.
   SecureKey reserves all rights in the code and you may not use it without
	 written permission from SecureKey.
*/

package listener

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/securekey/fabric-snaps/transactionsnap/pkg/txsnapservice"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("configsnap")

// ChaincodeListener listen for events emitted by fabric
type ChaincodeListener interface {
	// Listen for events
	Listen() (<-chan *fab.CCEvent, error)
	Stop()
}

// EventSource defines the source of an event
type EventSource struct {
	// ChannelID name of the channel to listen on
	ChannelID string
	// ChaincodeID chaincode that will emit the chaincode event
	ChaincodeID string
	// EventName name of the event
	EventName string
}

type listenerImpl struct {
	source *EventSource
	reg    fab.Registration
}

// NewChaincodeListener Create a new chaincode listener on the specified source
func NewChaincodeListener(source *EventSource) (ChaincodeListener, error) {
	// Validate arguments
	if source == nil || source.ChaincodeID == "" || source.ChannelID == "" || source.EventName == "" {
		return nil, errors.New(errors.SystemError, "A valid event source is required to create listener")
	}

	return &listenerImpl{
		source: source,
	}, nil
}

func (l *listenerImpl) Listen() (<-chan *fab.CCEvent, error) {
	channelID := l.source.ChannelID
	ccID := l.source.ChaincodeID
	eventFilter := l.source.EventName

	logger.Infof("Starting chaincode listener on channel [%s] for CC [%s] and Event [%s]", channelID, ccID, eventFilter)

	eventService, err := getEventService(channelID)
	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "GetEventService failed")
	}

	logger.Debugf("Registering for CC events on channel %s, CC %s, and event filter %s", channelID, ccID, eventFilter)

	reg, eventch, err := eventService.RegisterChaincodeEvent(ccID, eventFilter)
	if err != nil {
		return nil, errors.Wrapf(errors.SystemError, err, "Error registering for CC events on channel %s, CC %s, and event filter %s", channelID, ccID, eventFilter)
	}

	l.reg = reg

	if err != nil {
		return nil, errors.WithMessage(errors.SystemError, err, "Error initializing listener")
	}

	return eventch, nil
}

func (l *listenerImpl) Stop() {
	if l.reg == nil {
		return
	}

	eventService, err := getEventService(l.source.ChannelID)
	if err != nil {
		logger.Warnf("GetEventService failed %s", err)
		return
	}

	eventService.Unregister(l.reg)
}

func getEventService(channelID string) (fab.EventService, error) {
	//get event service via the tx snap service
	txService, err := txsnapservice.Get(channelID)
	if err != nil {
		logger.Debugf("Cannot get txService: %v", err)
		return nil, err
	}
	return txService.FcClient.EventService()
}
