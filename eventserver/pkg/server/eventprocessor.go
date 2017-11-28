/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger/fabric/core/aclmgmt"
	pb "github.com/hyperledger/fabric/protos/peer"
	eventserverapi "github.com/securekey/fabric-snaps/eventserver/api"
	"github.com/securekey/fabric-snaps/eventserver/pkg/channelutil"
)

//eventProcessor has a map of channel ids to handlers interested in that
//channel. start() kicks of the event processor where it waits for events
type eventProcessor struct {
	// this is a global lock protecting resources of the event processor. The
	// assumption is client handler reg/dereg is very infrequent compared to
	// the sending of events. We use this option for simplicity as opposed to
	// doing multiple levels of locks.
	sync.RWMutex
	registeredListeners map[string]*channelHandlerList

	eventChannel chan *pb.Event

	//timeout duration for server to send an event.
	//if < 0, if buffer full, unblocks immediately and not send
	//if 0, if buffer full, will block and guarantee the event will be sent out
	//if > 0, if buffer full, blocks till timeout
	timeout time.Duration

	//time difference from peer time where registration events can be considered
	//valid
	timeWindow time.Duration
}

//global eventProcessor singleton created by initializeEventProcessor
//var gEventProcessor *eventProcessor
var once sync.Once

//initialize event processor and start
func initializeEventProcessor(config *ChannelServerConfig) *eventProcessor {
	var ep *eventProcessor
	once.Do(func() {
		ep = &eventProcessor{
			registeredListeners: make(map[string]*channelHandlerList),
			eventChannel:        make(chan *pb.Event, config.BufferSize),
			timeout:             config.Timeout,
			timeWindow:          config.TimeWindow,
		}
		go ep.start()
	})
	return ep
}

func (ep *eventProcessor) start() {
	for {
		//wait for event
		e := <-ep.eventChannel
		if e.Event == nil {
			logger.Errorf("Event is nil\n")
			continue
		}

		sendMessage := func(e *pb.Event) {
			ep.RLock()
			defer ep.RUnlock()

			channelID, err := channelutil.ChannelIDFromEvent(e)
			if err != nil {
				logger.Errorf("unable to extract channel ID from the event: %s\n", err)
				return
			}

			hl, exists := ep.registeredListeners[channelID]
			if !exists {
				// handler doesn't exist for channel, i.e. no clients have registered for
				// events on this channel
				logger.Infof("handler doesn't exist for channel, i.e. no clients have registered for events on channel %s\n", channelID)
				return
			}

			hl.foreach(e, func(h *channelHandler) {
				eptr := *e
				switch eptr.Event.(type) {
				case *pb.Event_Block:
					if !h.interestedEvents[aclmgmt.BLOCKEVENT] {
						logger.Errorf("block event not allowed for channel [%s]\n", channelID)
						return
					}
				case *pb.Event_FilteredBlock:
					if !h.interestedEvents[aclmgmt.FILTEREDBLOCKEVENT] {
						logger.Errorf("filtered block event not allowed for channel [%s]\n", channelID)
						return
					}
				}
				csresp := &eventserverapi.ChannelServiceResponse{Response: &eventserverapi.ChannelServiceResponse_Event{Event: e}}
				logger.Debugf("sending event %s\n", csresp)
				h.SendMessage(csresp)
			})
			return
		}
		sendMessage(e)
	}
}

func (ep *eventProcessor) registerHandler(channelID string, ch *channelHandler) {
	logger.Warningf("registering channel handler for channel: %s", channelID)

	ep.RLock()
	hl, ok := ep.registeredListeners[channelID]
	if !ok {
		ep.RUnlock()
		ep.Lock()
		// check again for existence of handler list as it may have been created
		// while waiting to obtain the write lock
		hl, ok := ep.registeredListeners[channelID]
		if !ok {
			hl = new(channelHandlerList)
			hl.handlers = make(map[*channelHandler]bool)
			hl.add(ch)
			ep.registeredListeners[channelID] = hl
		}
		ep.Unlock()
		return
	}
	ep.RUnlock()
	hl.Lock()
	defer hl.Unlock()
	hl.add(ch)
}

func (ep *eventProcessor) deregisterHandler(channelID string, ch *channelHandler) error {
	logger.Debugf("deregistering channel handler for channel: %s", channelID)

	ep.Lock()
	defer ep.Unlock()
	hl, ok := ep.registeredListeners[channelID]
	if !ok {
		return fmt.Errorf("channel handler list does not exist for channel [%s]", channelID)
	}
	hl.del(ch)
	return nil
}

// deregisterStream deregisters the stream from all channels that it was
// listening for events
func (ep *eventProcessor) deregisterStream(stream eventserverapi.Channel_ChatServer) {
	for channelID := range ep.registeredListeners {
		for handler := range ep.registeredListeners[channelID].handlers {
			if handler.ChatStream == stream {
				handler.deregister(channelID)
			}
		}
	}
}

//------------- Channel Service server APIs -------------------------------

// Send sends the event to interested clients
func (ep *eventProcessor) send(e *pb.Event) error {
	logger.Debugf("Entry")
	defer logger.Debugf("Exit")
	if e.Event == nil {
		logger.Error("event not set")
		return fmt.Errorf("event not set")
	}

	if ep == nil {
		logger.Errorf("****** Event processor is nil\n")
		return nil
	}

	if ep.timeout < 0 {
		logger.Debugf("Event processor timeout < 0")
		logger.Debugf("****** Sending...\n")
		select {
		case ep.eventChannel <- e:
			logger.Debugf("****** Sent event...\n")
		default:
			return fmt.Errorf("could not send the block event")
		}
	} else if ep.timeout == 0 {
		logger.Debugf("Event processor timeout = 0")
		logger.Debugf("****** Sending event...\n")
		ep.eventChannel <- e
		logger.Debugf("****** Sent event...\n")
	} else {
		logger.Debugf("Event processor timeout > 0")
		logger.Debugf("****** Sending event...\n")
		select {
		case ep.eventChannel <- e:
			logger.Debugf("****** Sent event...\n")
		case <-time.After(ep.timeout):
			return fmt.Errorf("could not send the block event")
		}
	}

	logger.Debugf("Event sent successfully")
	return nil
}
