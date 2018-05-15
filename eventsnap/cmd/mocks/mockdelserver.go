/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"io"
	"sync"

	"fmt"
	"net"

	cb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// MockDeliverServer is a mock deliver server
type MockDeliverServer struct {
	sync.RWMutex
	status     cb.Status
	disconnErr error
	grpcServer *grpc.Server
}

// StartMockDeliverServer will start mock deliver server for unit testing purpose
func StartMockDeliverServer(testAddress string) (*MockDeliverServer, error) {
	grpcServer := grpc.NewServer()
	grpcServer.GetServiceInfo()
	lis, err := net.Listen("tcp", testAddress)
	if err != nil {
		return nil, fmt.Errorf("Error starting test server %s", err)
	}
	eventServer := &MockDeliverServer{grpcServer: grpcServer, status: cb.Status_UNKNOWN}
	pb.RegisterDeliverServer(grpcServer, eventServer)
	fmt.Printf("Starting mock deliver server\n")
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			fmt.Printf("StartMockDeliverServer failed %v", err.Error())
		}
	}()

	return eventServer, nil
}

// SetStatus sets the status to return when calling Deliver or DeliverFiltered
func (s *MockDeliverServer) SetStatus(status cb.Status) {
	s.Lock()
	defer s.Unlock()
	s.status = status
}

// Status returns the status that's returned when calling Deliver or DeliverFiltered
func (s *MockDeliverServer) Status() cb.Status {
	s.RLock()
	defer s.RUnlock()
	return s.status
}

// Disconnect terminates the stream and returns the given error to the client
func (s *MockDeliverServer) Disconnect(err error) {
	s.Lock()
	defer s.Unlock()
	s.disconnErr = err
}

func (s *MockDeliverServer) disconnectErr() error {
	s.RLock()
	defer s.RUnlock()
	return s.disconnErr
}

// Deliver delivers a stream of blocks
func (s *MockDeliverServer) Deliver(srv pb.Deliver_DeliverServer) error {
	status := s.Status()
	if status != cb.Status_UNKNOWN {
		err := srv.Send(&pb.DeliverResponse{
			Type: &pb.DeliverResponse_Status{
				Status: status,
			},
		})
		return errors.Errorf("returning error status: %s %v", status, err)
	}

	for {
		envelope, err := srv.Recv()
		if err == io.EOF || envelope == nil {
			break
		}

		err = s.disconnectErr()
		if err != nil {
			return err
		}

		err1 := srv.Send(&pb.DeliverResponse{
			Type: &pb.DeliverResponse_Block{
				Block: &cb.Block{},
			},
		})
		if err1 != nil {
			return err1
		}
	}
	return nil
}

// DeliverFiltered delivers a stream of filtered blocks
func (s *MockDeliverServer) DeliverFiltered(srv pb.Deliver_DeliverFilteredServer) error {
	if s.status != cb.Status_UNKNOWN {
		err1 := srv.Send(&pb.DeliverResponse{
			Type: &pb.DeliverResponse_Status{
				Status: s.status,
			},
		})
		return errors.Errorf("returning error status: %s %v", s.status, err1)
	}

	for {
		envelope, err := srv.Recv()
		if err == io.EOF || envelope == nil {
			break
		}

		err = s.disconnectErr()
		if err != nil {
			return err
		}

		err1 := srv.Send(&pb.DeliverResponse{
			Type: &pb.DeliverResponse_FilteredBlock{
				FilteredBlock: &pb.FilteredBlock{},
			},
		})
		if err1 != nil {
			return err1
		}
	}
	return nil
}

// Stop mock deliver
func (s *MockDeliverServer) Stop() {
	s.grpcServer.Stop()
}
