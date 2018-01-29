/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package server

import (
	"sync"

	pb "github.com/hyperledger/fabric/protos/peer"
	eventserverapi "github.com/securekey/fabric-snaps/eventserver/api"
	"github.com/securekey/fabric-snaps/util/errors"
)

type channelHandler struct {
	channelServer *ChannelServer
	ChatStream    eventserverapi.Channel_ChatServer
	// stores the events the client is interested in and allowed to receive
	// (based on the ACL provider)
	interestedEvents map[string]bool
}

func newChannelHandler(cs *ChannelServer, stream eventserverapi.Channel_ChatServer) *channelHandler {
	ch := &channelHandler{
		channelServer: cs,
		ChatStream:    stream,
	}

	ch.interestedEvents = make(map[string]bool)
	return ch
}

func (ch *channelHandler) addInterestedEvent(eventName string) {
	ch.interestedEvents[eventName] = true
}

// SendMessage sends a message to the remote PEER through the stream
func (ch *channelHandler) SendMessage(msg *eventserverapi.ChannelServiceResponse) error {
	err := ch.ChatStream.Send(msg)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error Sending message through ChatStream")
	}
	return nil
}

// register registers a client handler for a specific channel
func (ch *channelHandler) register(channelID string) {
	ch.channelServer.gEventProcessor.registerHandler(channelID, ch)
}

// deregister deregisters a client handler for a specific channel
func (ch *channelHandler) deregister(channelID string) error {
	err := ch.channelServer.gEventProcessor.deregisterHandler(channelID, ch)
	if err != nil {
		return err
	}
	return nil
}

type handlerList interface {
	add(ch *channelHandler) (bool, error)
	del(ch *channelHandler) (bool, error)
	foreach(evt *pb.Event, action func(ch *channelHandler))
}

type channelHandlerList struct {
	sync.RWMutex
	handlers map[*channelHandler]bool
}

func (hl *channelHandlerList) add(ch *channelHandler) bool {
	if ch == nil {
		logger.Warnf("cannot add nil channel handler")
		return false
	}
	if _, ok := hl.handlers[ch]; ok {
		logger.Warnf("handler exists for channel")
		return false
	}
	hl.handlers[ch] = true
	return true
}

func (hl *channelHandlerList) del(ch *channelHandler) bool {
	if _, ok := hl.handlers[ch]; !ok {
		logger.Warnf("handler does not exist for channel")
		return false
	}
	delete(hl.handlers, ch)
	return true
}

func (hl *channelHandlerList) foreach(e *pb.Event, action func(ch *channelHandler)) {
	for ch := range hl.handlers {
		action(ch)
	}
}
