/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package relay

import (
	"crypto/tls"
	"io"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/golang/protobuf/ptypes/timestamp"
	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/hyperledger/fabric/common/util"
	cutil "github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/comm"
	mspmgmt "github.com/hyperledger/fabric/msp/mgmt"
	ehpb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	"github.com/securekey/fabric-snaps/util/errors"
)

var consumerLogger = logging.NewLogger("eventsnap")

//EventsClient holds the stream and adapter for consumer to work with
type EventsClient struct {
	sync.RWMutex
	peerAddress    string
	regTimeout     time.Duration
	stream         ehpb.Events_ChatClient
	adapter        EventAdapter
	tlsCredentials credentials.TransportCredentials
	tlsCertHash    []byte
}

// RegistrationConfig holds the information to be used when registering for
// events from the eventhub
type RegistrationConfig struct {
	InterestedEvents []*ehpb.Interest
	Timestamp        *timestamp.Timestamp
}

//NewEventsClient Returns a new grpc.ClientConn to the configured local PEER.
func NewEventsClient(peerAddress string, regTimeout time.Duration, adapter EventAdapter, tlsConfig *tls.Config) (*EventsClient, error) {
	var err error
	if regTimeout < 100*time.Millisecond {
		regTimeout = 100 * time.Millisecond
		err = errors.New(errors.GeneralError, "regTimeout >= 0, setting to 100 msec")
	} else if regTimeout > 60*time.Second {
		regTimeout = 60 * time.Second
		err = errors.New(errors.GeneralError, "regTimeout > 60, setting to 60 sec")
	}
	return &EventsClient{
		RWMutex:        sync.RWMutex{},
		peerAddress:    peerAddress,
		regTimeout:     regTimeout,
		adapter:        adapter,
		tlsCredentials: credentials.NewTLS(tlsConfig),
		tlsCertHash:    tlsCertHash(tlsConfig.Certificates),
	}, err
}

//newEventsClientConnectionWithAddress Returns a new grpc.ClientConn to the configured local PEER.
func newEventsClientConnectionWithAddress(peerAddress string, tlsCredentials credentials.TransportCredentials) (*grpc.ClientConn, error) {
	if comm.TLSEnabled() {
		// Custom code begin
		// return comm.NewClientConnectionWithAddress(peerAddress, true, true, comm.InitTLSForPeer(), nil)
		consumerLogger.Debugf("tlsCredentials: %s", tlsCredentials)
		return comm.NewClientConnectionWithAddress(peerAddress, true, true, tlsCredentials, nil)
		// Custom code end
	}
	return comm.NewClientConnectionWithAddress(peerAddress, true, false, nil, nil)
}

func (ec *EventsClient) send(emsg *ehpb.Event) error {
	ec.Lock()
	defer ec.Unlock()

	// obtain the default signing identity for this peer; it will be used to sign the event
	localMsp := mspmgmt.GetLocalMSP()
	if localMsp == nil {
		return errors.New(errors.GeneralError, "nil local MSP manager")
	}

	signer, err := localMsp.GetDefaultSigningIdentity()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "could not obtain the default signing identity")
	}

	//pass the signer's cert to Creator
	signerCert, err := signer.Serialize()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "fail to serialize the default signing identity")
	}
	emsg.Creator = signerCert

	signedEvt, err := utils.GetSignedEvent(emsg, signer)
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "could not sign outgoing event")
	}

	return ec.stream.Send(signedEvt)
}

// RegisterAsync - registers interest in a event and doesn't wait for a response
func (ec *EventsClient) RegisterAsync(config *RegistrationConfig) error {
	creator, err := getCreatorFromLocalMSP()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error getting creator from MSP")
	}
	emsg := &ehpb.Event{
		Event:       &ehpb.Event_Register{Register: &ehpb.Register{Events: config.InterestedEvents}},
		Creator:     creator,
		Timestamp:   config.Timestamp,
		TlsCertHash: ec.tlsCertHash,
	}

	if err = ec.send(emsg); err != nil {
		consumerLogger.Errorf("error on Register send %s\n", err)
	}
	return err
}

// register - registers interest in a event
func (ec *EventsClient) register(config *RegistrationConfig) error {
	var err error
	if err = ec.RegisterAsync(config); err != nil {
		return err
	}

	regChan := make(chan struct{})
	go func() {
		defer close(regChan)
		in, inerr := ec.stream.Recv()
		if inerr != nil {
			err = inerr
			return
		}
		switch in.Event.(type) {
		case *ehpb.Event_Register:
		case nil:
			err = errors.New(errors.GeneralError, "invalid nil object for register")
		default:
			err = errors.New(errors.GeneralError, "invalid registration object")
		}
	}()
	select {
	case <-regChan:
	case <-time.After(ec.regTimeout):
		err = errors.New(errors.GeneralError, "timeout waiting for registration")
	}
	return err
}

// UnregisterAsync - Unregisters interest in a event and doesn't wait for a response
func (ec *EventsClient) UnregisterAsync(ies []*ehpb.Interest) error {
	creator, err := getCreatorFromLocalMSP()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error getting creator from MSP")
	}
	emsg := &ehpb.Event{Event: &ehpb.Event_Unregister{Unregister: &ehpb.Unregister{Events: ies}}, Creator: creator}

	if err = ec.send(emsg); err != nil {
		err = errors.WithMessage(errors.GeneralError, err, "error on unregister send")
	}

	return err
}

// Recv receives next event - use when client has not called Start
func (ec *EventsClient) Recv() (*ehpb.Event, error) {
	in, err := ec.stream.Recv()
	if err == io.EOF {
		// read done.
		if ec.adapter != nil {
			ec.adapter.Disconnected(nil)
		}
		return nil, err
	}
	if err != nil {
		if ec.adapter != nil {
			ec.adapter.Disconnected(err)
		}
		return nil, err
	}
	return in, nil
}
func (ec *EventsClient) processEvents() error {
	defer ec.stream.CloseSend()
	for {
		in, err := ec.stream.Recv()
		if err == io.EOF {
			// read done.
			if ec.adapter != nil {
				ec.adapter.Disconnected(nil)
			}
			return nil
		}
		if err != nil {
			if ec.adapter != nil {
				ec.adapter.Disconnected(err)
			}
			return err
		}
		if ec.adapter != nil {
			cont, err := ec.adapter.Recv(in)
			if !cont {
				return err
			}
		}
	}
}

//Start establishes connection with Event hub and registers interested events with it
func (ec *EventsClient) Start() error {
	conn, err := newEventsClientConnectionWithAddress(ec.peerAddress, ec.tlsCredentials)
	if err != nil {
		return errors.Errorf(errors.GeneralError, "could not create client conn to %s:%s", ec.peerAddress, err)
	}

	ies, err := ec.adapter.GetInterestedEvents()
	if err != nil {
		return errors.WithMessage(errors.GeneralError, err, "error getting interested events")
	}

	if len(ies) == 0 {
		return errors.New(errors.GeneralError, "must supply interested events")
	}

	serverClient := ehpb.NewEventsClient(conn)
	ec.stream, err = serverClient.Chat(context.Background())
	if err != nil {
		return errors.Errorf(errors.GeneralError, "could not create client conn to %s:%s", ec.peerAddress, err)
	}

	regConfig := &RegistrationConfig{InterestedEvents: ies, Timestamp: util.CreateUtcTimestamp()}
	if err = ec.register(regConfig); err != nil {
		return err
	}

	go ec.processEvents()

	return nil
}

//Stop terminates connection with event hub
func (ec *EventsClient) Stop() error {
	if ec.stream == nil {
		// in case the steam/chat server has not been established earlier, we assume that it's closed, successfully
		return nil
	}
	return ec.stream.CloseSend()
}

func getCreatorFromLocalMSP() ([]byte, error) {
	localMsp := mspmgmt.GetLocalMSP()
	if localMsp == nil {
		return nil, errors.New(errors.GeneralError, "nil local MSP manager")
	}
	signer, err := localMsp.GetDefaultSigningIdentity()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "could not obtain the default signing identity")
	}
	creator, err := signer.Serialize()
	if err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "error serializing the signer")
	}
	return creator, nil
}

func tlsCertHash(certs []tls.Certificate) []byte {
	if len(certs) == 0 {
		return nil
	}

	cert := certs[0]
	if len(cert.Certificate) == 0 {
		return nil
	}
	return cutil.ComputeSHA256(cert.Certificate[0])
}
