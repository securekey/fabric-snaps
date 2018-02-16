/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"golang.org/x/net/context"

	//rwsetutil "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	//kvrwset "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"

	"fmt"
	"net"

	"github.com/hyperledger/fabric-sdk-go/api/apifabclient"

	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"google.golang.org/grpc"
)

// MockEndorserServer mock endoreser server to process endorsement proposals
type MockEndorserServer struct {
	MockPeer *MockPeer
}

// ProcessProposal mock implementation that returns success if error is not set
// error if it is
func (m *MockEndorserServer) ProcessProposal(context context.Context,
	proposal *pb.SignedProposal) (*pb.ProposalResponse, error) {
	tp, err := m.MockPeer.ProcessTransactionProposal(apifabclient.TransactionProposal{})
	return tp.ProposalResponse, err

}

//func (m *MockEndorserServer) createProposalResponsePayload() []byte {

//	prp := &pb.ProposalResponsePayload{}
//	ccAction := &pb.ChaincodeAction{}
//	txRwSet := &rwsetutil.TxRwSet{}

//	if m.AddkvWrite {
//		txRwSet.NsRwSets = []*rwsetutil.NsRwSet{
//			&rwsetutil.NsRwSet{NameSpace: "ns1", KvRwSet: &kvrwset.KVRWSet{
//				Reads:  []*kvrwset.KVRead{&kvrwset.KVRead{Key: "key1", Version: &kvrwset.Version{BlockNum: 1, TxNum: 1}}},
//				Writes: []*kvrwset.KVWrite{&kvrwset.KVWrite{Key: "key2", IsDelete: false, Value: []byte("value2")}},
//			}}}
//	}

//	txRWSetBytes, err := txRwSet.ToProtoBytes()
//	if err != nil {
//		return nil
//	}
//	ccAction.Results = txRWSetBytes
//	ccActionBytes, err := proto.Marshal(ccAction)
//	if err != nil {
//		return nil
//	}
//	prp.Extension = ccActionBytes
//	prpBytes, err := proto.Marshal(prp)
//	if err != nil {
//		return nil
//	}
//	return prpBytes
//}

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
