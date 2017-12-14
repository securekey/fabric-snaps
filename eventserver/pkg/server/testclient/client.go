/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testclient

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	serverapi "github.com/securekey/fabric-snaps/eventserver/api"
)

var logger = flogging.MustGetLogger("channelservice/client")

// ChannelClient holds the stream and adapter for client to work with
type ChannelClient struct {
	sync.RWMutex
	peerAddress string
	regTimeout  time.Duration
	stream      serverapi.Channel_ChatClient
	adapter     ChannelAdapter
}

// RegistrationConfig holds the information used when registering for events
// from the channel service
type RegistrationConfig struct {
	InterestedChannels []string
	InterestedEvents   []*pb.Interest
	Timestamp          *timestamp.Timestamp
}

// NewChannelClient returns a new grpc.ClientConn to the configured local peer
func NewChannelClient(peerAddress string, regTimeout time.Duration, adapter ChannelAdapter) (*ChannelClient, error) {
	var err error
	if regTimeout < 100*time.Millisecond {
		regTimeout = 100 * time.Millisecond
		err = fmt.Errorf("regTimeout >= 0, setting to 100 msec")
	} else if regTimeout > 60*time.Second {
		regTimeout = 60 * time.Second
		err = fmt.Errorf("regTimeout > 60, setting to 60 sec")
	}
	return &ChannelClient{sync.RWMutex{}, peerAddress, regTimeout, nil, adapter}, err
}

// newChannelClientConnectionWithAddress returns a new grpc.ClientConn to the
// configured local peer
func newChannelClientConnectionWithAddress(peerAddress string) (*grpc.ClientConn, error) {
	if comm.TLSEnabled() {
		return comm.NewClientConnectionWithAddress(peerAddress, true, true, comm.InitTLSForPeer(), nil)
	}
	return comm.NewClientConnectionWithAddress(peerAddress, true, false, nil, nil)
}

// Send takes a ChannelServiceRequest, creates a signed envelope, and sends it
// to the connected ChannelServer
func (cc *ChannelClient) Send(csreq *serverapi.ChannelServiceRequest) error {
	env, err := cc.CreateEnvelopeForChannelServiceRequest(csreq)
	if err != nil {
		return fmt.Errorf("error creating envelope for channel service request: %s", err)
	}
	return cc.streamSend(env)
}

// CreateEnvelopeForChannelServiceRequest creates a signed envelope for a given
// ChannelServiceRequest
func (cc *ChannelClient) CreateEnvelopeForChannelServiceRequest(csreq *serverapi.ChannelServiceRequest) (*common.Envelope, error) {
	if csreq == nil {
		return nil, fmt.Errorf("cannot send nil channel service request")
	}

	msgVersion := int32(0)
	epoch := uint64(0)
	env, err := utils.CreateSignedEnvelope(common.HeaderType(serverapi.HeaderType_CHANNEL_SERVICE_REQUEST), "", localmsp.NewSigner(), csreq, msgVersion, epoch)

	if err != nil {
		return nil, err
	}
	return env, nil
}

// SendEnvelope sends a signed envelope to the connected ChannelServer
func (cc *ChannelClient) SendEnvelope(env *common.Envelope) error {
	if env == nil {
		return fmt.Errorf("cannot send nil envelope")
	}
	return cc.streamSend(env)
}

func (cc *ChannelClient) streamSend(env *common.Envelope) error {
	cc.Lock()
	defer cc.Unlock()
	return cc.stream.Send(env)
}

// register registers interest in based on the provided RegistrationConfig
func (cc *ChannelClient) register(config *RegistrationConfig) error {
	var err error
	if err = cc.RegisterAsync(config); err != nil {
		return err
	}

	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := cc.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Response.(type) {
		case *serverapi.ChannelServiceResponse_Result:
		case nil:
			err = fmt.Errorf("invalid nil object for register")
		default:
			err = fmt.Errorf("invalid registration object")
		}
	}()
	select {
	case <-regChan:
	case <-time.After(cc.regTimeout):
		err = fmt.Errorf("timeout waiting for registration")
	}
	return err
}

// RegisterAsync registers interest based on the provided RegistrationConfig
// and doesn't wait for a response
func (cc *ChannelClient) RegisterAsync(config *RegistrationConfig) error {
	csreq := &serverapi.ChannelServiceRequest{Request: &serverapi.ChannelServiceRequest_RegisterChannel{RegisterChannel: &serverapi.RegisterChannel{ChannelIds: config.InterestedChannels, Events: config.InterestedEvents}}}
	if err := cc.Send(csreq); err != nil {
		logger.Errorf("error sending register event with registration config %v: %s\n", config, err)
		return err
	}
	return nil
}

// DeregisterAsync deregisters interest in channel and doesn't wait for a response
func (cc *ChannelClient) DeregisterAsync(channels []string) error {
	csreq := &serverapi.ChannelServiceRequest{Request: &serverapi.ChannelServiceRequest_DeregisterChannel{DeregisterChannel: &serverapi.DeregisterChannel{ChannelIds: channels}}}
	if err := cc.Send(csreq); err != nil {
		logger.Errorf("error sending deregister event for channels %s: %s\n", channels, err)
		return err
	}
	return nil
}

// Recv receives next event - use when client has not called Start
func (cc *ChannelClient) Recv() (*serverapi.ChannelServiceResponse, error) {
	in, err := cc.stream.Recv()
	if err == io.EOF {
		// read done.
		if cc.adapter != nil {
			cc.adapter.Disconnected(nil)
		}
		return nil, err
	}
	if err != nil {
		if cc.adapter != nil {
			cc.adapter.Disconnected(err)
		}
		return nil, err
	}
	return in, nil
}

func (cc *ChannelClient) processEvents() error {
	defer cc.stream.CloseSend()
	for {
		in, err := cc.stream.Recv()
		if err == io.EOF {
			// read done.
			if cc.adapter != nil {
				cc.adapter.Disconnected(nil)
			}
			return nil
		}
		if err != nil {
			if cc.adapter != nil {
				cc.adapter.Disconnected(err)
			}
			return err
		}
		if cc.adapter != nil {
			cont, err := cc.adapter.Recv(in)
			if !cont {
				return err
			}
		}
	}
}

// Start establishes connection with channel service server and registers
// for interested channels
func (cc *ChannelClient) Start() error {
	conn, err := newChannelClientConnectionWithAddress(cc.peerAddress)
	if err != nil {
		return fmt.Errorf("could not create client conn to %s:%s", cc.peerAddress, err)
	}

	ic, err := cc.adapter.GetInterestedChannels()
	if err != nil {
		return fmt.Errorf("error getting interested channels:%s", err)
	}

	if len(ic) == 0 {
		return fmt.Errorf("must supply interested channels")
	}

	ies, err := cc.adapter.GetInterestedEvents()
	if err != nil {
		return fmt.Errorf("error getting interested events:%s", err)
	}

	if len(ies) == 0 {
		return fmt.Errorf("must supply interested events")
	}

	serverClient := serverapi.NewChannelClient(conn)
	cc.stream, err = serverClient.Chat(context.Background())
	if err != nil {
		return fmt.Errorf("could not create client conn to %s:%s", cc.peerAddress, err)
	}

	regConfig := &RegistrationConfig{InterestedChannels: ic, InterestedEvents: ies, Timestamp: util.CreateUtcTimestamp()}
	if err = cc.RegisterAsync(regConfig); err != nil {
		return err
	}

	go cc.processEvents()

	return nil
}

// Stop terminates connection with the channel service server
func (cc *ChannelClient) Stop() error {
	if cc.stream == nil {
		// in case the steam/chat server has not been established earlier, we assume that it's closed, successfully
		return nil
	}
	return cc.stream.CloseSend()
}
