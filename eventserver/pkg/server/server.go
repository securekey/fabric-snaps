/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package server

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	eventserverapi "github.com/securekey/fabric-snaps/eventserver/api"
)

var logger = flogging.MustGetLogger("eventserver")

// ChannelServer implementation of the channel service
type ChannelServer struct {
	gEventProcessor *eventProcessor
}

// ChannelServerConfig contains the setup config for the channel server
type ChannelServerConfig struct {
	BufferSize uint
	Timeout    time.Duration
	TimeWindow time.Duration
}

// NewChannelServer creates and returns a new Channel server instance.
func NewChannelServer(config *ChannelServerConfig) *ChannelServer {
	globalChannelServer := new(ChannelServer)
	globalChannelServer.gEventProcessor = initializeEventProcessor(config)
	return globalChannelServer
}

// Chat a new client on the channel
func (c *ChannelServer) Chat(stream eventserverapi.Channel_ChatServer) error {
	logger.Debugf("************ Initiating new Chat\n")

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			logger.Debug("received EOF, ending channel service chat")
			break
		}
		if err != nil {
			logger.Debug("error during channel service chat, stopping handler: %s", err)
			break
		}
		err = c.handleMessage(stream, in)
		if err != nil {
			// log error message but keep Chat alive
			logger.Errorf("error handling message: %s", err)
		}
	}
	c.gEventProcessor.deregisterStream(stream)
	return nil
}

// Send sends an event to the event processor
func (c *ChannelServer) Send(evt *pb.Event) error {
	logger.Debugf("************ Sending: %s\n", evt)
	return c.gEventProcessor.send(evt)
}

// HandleMessage handles the messages for the peer
func (c *ChannelServer) handleMessage(stream eventserverapi.Channel_ChatServer, env *common.Envelope) error {
	logger.Debugf("******************* Handle Message\n")

	csreq := &eventserverapi.ChannelServiceRequest{}
	chdr, err := utils.UnmarshalEnvelopeOfType(env, common.HeaderType(eventserverapi.HeaderType_CHANNEL_SERVICE_REQUEST), csreq)
	if err != nil {
		logger.Warningf("error unmarshaling channel service request: %s", err)
		return nil
	}
	if err = c.validateTimestamp(chdr.GetTimestamp()); err != nil {
		return err
	}

	logger.Debugf("******************* Handling request: %s\n", csreq)

	var response *eventserverapi.ChannelServiceResponse

	if csreq.GetRegisterChannel() != nil {
		logger.Debugf("******************* Handling registration request\n")
		response = c.processRegistration(csreq.GetRegisterChannel().ChannelIds, csreq.GetRegisterChannel().Events, env)
		if response.GetResult().Success {
			for _, channelResult := range response.GetResult().ChannelResults {
				c.createHandler(channelResult.ChannelId, channelResult.RegisteredEvents, stream)
				logger.Debugf("******************* Successfully registered for events on channel: %s\n", channelResult.ChannelId)
			}
		} else {
			logger.Errorf("******************* Error processing registration: %s\n", response.GetResult().ChannelResults)
		}
	} else if csreq.GetDeregisterChannel() != nil {
		response = c.processDeregistration(csreq.GetDeregisterChannel().GetChannelIds(), stream)
	} else {
		logger.Warningf("received empty channel service request from client. Expected request containing RegisterChannel or DeregisterChannel message")
		return nil
	}

	logger.Debugf("sending channel service response: %v", response)
	if err := stream.Send(response); err != nil {
		return fmt.Errorf("error sending registration response %v:  %s", response, err)
	}

	return nil
}

// validateRequestTimestamp checks the timestamp provided against the configured
// timewindow for the channel service
func (c *ChannelServer) validateTimestamp(timestamp *timestamp.Timestamp) error {
	evtTime := time.Unix(timestamp.Seconds, int64(timestamp.Nanos)).UTC()
	peerTime := time.Now()
	if math.Abs(float64(peerTime.UnixNano()-evtTime.UnixNano())) > float64(c.gEventProcessor.timeWindow.Nanoseconds()) {
		logger.Warningf("event timestamp %s is more than the %s `peer.channelservice.timewindow` difference above/below peer time %s. either the peer and client clocks are out of sync or a replay attack has been attempted", evtTime, c.gEventProcessor.timeWindow, peerTime)
		return fmt.Errorf("event timestamp out of acceptable range. must be within %s above/below peer time", c.gEventProcessor.timeWindow)
	}
	return nil
}

func (c *ChannelServer) processRegistration(channelIDs []string, interestedEvents []*pb.Interest, env *common.Envelope) *eventserverapi.ChannelServiceResponse {
	response := &eventserverapi.ChannelServiceResponse{Response: &eventserverapi.ChannelServiceResponse_Result{Result: &eventserverapi.ChannelServiceResult{Action: "RegisterChannel", Success: true}}}
	resultsArray := make([]*eventserverapi.ChannelResult, 0)

	for _, channelID := range channelIDs {
		result := &eventserverapi.ChannelResult{ChannelId: channelID}
		channelRegisteredEvents := make([]string, 0)
		if len(interestedEvents) == 0 {
			logger.Debugf("no interested events specified for channel [%s]. handlers will not be created", channelID)
			response.GetResult().Success = false
			result.ErrorMsg = "no interested events specified"
		} else {
			for _, interest := range interestedEvents {
				if interest.EventType == pb.EventType_BLOCK {
					if err := checkACL(aclmgmt.BLOCKEVENT, channelID, env); err != nil {
						logger.Errorf("%s", err)
					} else {
						channelRegisteredEvents = append(channelRegisteredEvents, aclmgmt.BLOCKEVENT)
					}
				} else if interest.EventType == pb.EventType_FILTEREDBLOCK {
					if err := checkACL(aclmgmt.FILTEREDBLOCKEVENT, channelID, env); err != nil {
						logger.Errorf("%s", err)
					} else {
						channelRegisteredEvents = append(channelRegisteredEvents, aclmgmt.FILTEREDBLOCKEVENT)
					}
				} else {
					logger.Debugf("ignoring unexpected event type %s in registration request", interest.EventType)
				}
			}
			if len(channelRegisteredEvents) == 0 {
				logger.Debugf("not authorized to receive events for channel [%s]. handlers will not be created", channelID)
				response.GetResult().Success = false
				result.ErrorMsg = "not authorized to receive events for channel"
			}
		}
		result.RegisteredEvents = channelRegisteredEvents
		resultsArray = append(resultsArray, result)
	}
	response.GetResult().ChannelResults = resultsArray

	return response
}

func (c *ChannelServer) processDeregistration(channelIDs []string, stream eventserverapi.Channel_ChatServer) *eventserverapi.ChannelServiceResponse {
	response := &eventserverapi.ChannelServiceResponse{Response: &eventserverapi.ChannelServiceResponse_Result{Result: &eventserverapi.ChannelServiceResult{Action: "DeregisterChannel", Success: true}}}
	resultsArray := make([]*eventserverapi.ChannelResult, 0)
	for _, channelID := range channelIDs {
		result := &eventserverapi.ChannelResult{ChannelId: channelID}
		handler := c.getChannelHandlerIfExists(channelID, stream)
		if handler == nil {
			result.ErrorMsg = fmt.Sprintf("not registered for channel: %s", channelID)
			logger.Debug(result.ErrorMsg)
		} else if err := handler.deregister(channelID); err != nil {
			result.ErrorMsg = err.Error()
			logger.Debug(err.Error())
		}
		resultsArray = append(resultsArray, result)
	}
	response.GetResult().ChannelResults = resultsArray
	return response
}

func (c *ChannelServer) createHandler(channelID string, allowedEvents []string, stream eventserverapi.Channel_ChatServer) *channelHandler {
	handler := newChannelHandler(c, stream)

	for _, event := range allowedEvents {
		handler.addInterestedEvent(event)
	}
	handler.register(channelID)
	return handler
}

func checkACL(res string, cid string, env *common.Envelope) error {
	if err := aclmgmt.GetACLProvider().CheckACL(res, cid, env); err != nil {
		return fmt.Errorf("authorization request for [%s] on channel [%s] failed: [%s]", res, cid, err)
	}
	return nil
}

func (c *ChannelServer) getChannelHandlerIfExists(channelID string, stream eventserverapi.Channel_ChatServer) *channelHandler {
	if hl, ok := c.gEventProcessor.registeredListeners[channelID]; ok {
		//handler list exists, now check for a registered handler for the stream
		for cHandler := range hl.handlers {
			if cHandler.ChatStream == stream {
				return cHandler
			}
		}
	}
	return nil
}
