/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"golang.org/x/net/context"

	"fmt"
	"net"

	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"

	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"google.golang.org/grpc"
)

// MockEndorserServer mock endoreser server to process endorsement proposals
type MockEndorserServer struct {
	MockPeer     *MockPeer
	RequestCount int
	LastRequest  *pb.SignedProposal
}

// ProcessProposal mock implementation that returns success if error is not set
// error if it is
func (m *MockEndorserServer) ProcessProposal(context context.Context,
	proposal *pb.SignedProposal) (*pb.ProposalResponse, error) {
	m.RequestCount++
	tp, err := m.MockPeer.ProcessTransactionProposal(apifabclient.TransactionProposal{})
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
