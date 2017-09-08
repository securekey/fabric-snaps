/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"fmt"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/protos/peer"
	"google.golang.org/grpc"
)

// MockEventServer ...
type MockEventServer struct {
	server     peer.Events_ChatServer
	grpcServer *grpc.Server
	channel    chan *peer.Event
}

// StartMockEventServer will start server
func StartMockEventServer(testAddress string) (*MockEventServer, error) {
	grpcServer := grpc.NewServer()
	grpcServer.GetServiceInfo()
	lis, err := net.Listen("tcp", testAddress)
	eventServer := &MockEventServer{grpcServer: grpcServer}
	peer.RegisterEventsServer(grpcServer, eventServer)
	if err != nil {
		return nil, fmt.Errorf("Error starting test server %s", err)
	}
	fmt.Printf("Starting test server\n")
	go grpcServer.Serve(lis)

	return eventServer, nil
}

// Chat ...
func (m *MockEventServer) Chat(srv peer.Events_ChatServer) error {
	m.server = srv
	m.channel = make(chan *peer.Event)
	in, _ := srv.Recv()
	evt := &peer.Event{}
	err := proto.Unmarshal(in.EventBytes, evt)
	if err != nil {
		return fmt.Errorf("error unmarshaling the event bytes in the SignedEvent: %s", err)
	}
	switch evt.Event.(type) {
	case *peer.Event_Register:
		srv.Send(&peer.Event{Event: &peer.Event_Register{Register: &peer.Register{}}})
	}
	for {
		event := <-m.channel
		srv.Send(event)
	}
}

// SendMockEvent ...
func (m *MockEventServer) SendMockEvent(event *peer.Event) {
	m.channel <- event
}

// Stop mock event
func (m *MockEventServer) Stop() {
	m.grpcServer.Stop()
}
