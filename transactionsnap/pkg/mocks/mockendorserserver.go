/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"fmt"
	"net"
	"sync"

	"github.com/golang/protobuf/proto"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// MockEndorserServer mock endoreser server to process endorsement proposals
type MockEndorserServer struct {
	mockPeer     *MockPeer
	RequestCount int
	LastRequest  *pb.SignedProposal
	lck          sync.RWMutex
}

// ProcessProposal mock implementation that returns success if error is not set
// error if it is
func (m *MockEndorserServer) ProcessProposal(context context.Context,
	proposal *pb.SignedProposal) (*pb.ProposalResponse, error) {
	m.RequestCount++

	pr := &peer.Proposal{}
	err := proto.Unmarshal(proposal.GetProposalBytes(), pr)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling proposal: %v", err)
	}
	cpp := &peer.ChaincodeProposalPayload{}
	err = proto.Unmarshal(pr.Payload, cpp)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling proposal: %v", err)
	}

	cic := &peer.ChaincodeInvocationSpec{}
	err = proto.Unmarshal(cpp.Input, cic)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling proposal: %v", err)
	}

	tp, err := m.GetMockPeer().ProcessTransactionProposal(fabApi.TransactionProposal{}, cic.ChaincodeSpec.Input.Args[0])
	m.LastRequest = proposal

	return tp.ProposalResponse, err

}

//StartEndorserServer starts mock server for unit testing purpose
func StartEndorserServer(endorserTestURL string) *MockEndorserServer {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", endorserTestURL)
	if err != nil {
		panic(fmt.Sprintf("Error starting endorser server: %s", err))
	}
	endorserServer := &MockEndorserServer{}
	pb.RegisterEndorserServer(grpcServer, endorserServer)
	fmt.Printf("Test endorser server started\n")
	go grpcServer.Serve(lis)
	return endorserServer
}

// GetMockPeer will return the mock endorser's mock peer in a thread safe way
func (m *MockEndorserServer) GetMockPeer() *MockPeer {
	m.lck.RLock()
	defer m.lck.RUnlock()
	return m.mockPeer
}

// SetMockPeer will write the mock endorser's mock peer in a thread safe way
func (m *MockEndorserServer) SetMockPeer(mPeer *MockPeer) {
	m.lck.Lock()
	defer m.lck.Unlock()
	m.mockPeer = mPeer
}
