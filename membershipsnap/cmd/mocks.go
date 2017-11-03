/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/policies"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/policy"
	policymocks "github.com/hyperledger/fabric/core/policy/mocks"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	gossip2 "github.com/hyperledger/fabric/gossip/gossip"

	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	"github.com/hyperledger/fabric/gossip/service"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/gossip"
	"github.com/hyperledger/fabric/protos/ledger/rwset"
	msppb "github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
)

// newMockStub creates a mock stub for the MSCC.
// - identity is the identity of the signer
// - identityDeserializer is the deserializer that validates and deserializes the identity
// - localMSPID is the ID of the peer's local MSP
// - localPeerAddress is the address (host:port) of the local peer
// - members contains zero or more MSP network members
func newMockStub(identity []byte, identityDeserializer msp.IdentityDeserializer, localMSPID api.OrgIdentityType, localPeerAddress string, members ...mspNetworkMembers) *shim.MockStub {
	// Override the MSCC initializer in order to inject our mocks
	initializer = func(mscc *MembershipSnap, stub shim.ChaincodeStubInterface) error {
		policyChecker := policy.NewPolicyChecker(
			&policymocks.MockChannelPolicyManagerGetter{
				Managers: map[string]policies.Manager{},
			},
			identityDeserializer,
			&policymocks.MockMSPPrincipalGetter{Principal: identity},
		)

		m := make(map[string]string)
		for _, member := range members {
			for _, netMember := range member.networkMembers {
				m[string(netMember.PKIid)] = string(member.mspID)
			}
		}

		mscc.localMSPID = localMSPID
		mscc.localPeerAddress = localPeerAddress
		mscc.gossipService = newMockGossipService(members...)
		mscc.mspprovider = newmockMSPIDMgr(m)
		mscc.policyChecker = policyChecker

		return nil
	}

	stub := shim.NewMockStub("MembershipSnap", New())
	stub.MockInit("txid", nil)

	return stub
}

type mockGossipService struct {
	mockGossip
}

func newMockGossipService(members ...mspNetworkMembers) *mockGossipService {
	return &mockGossipService{mockGossip: mockGossip{
		members: members,
	}}
}

func (s *mockGossipService) sendMessage(msg gossip.ReceivedMessage) {
	go func() {
		if s.acceptor(msg) {
			s.msgCh <- msg
		}
	}()
}

func (s *mockGossipService) NewConfigEventer() service.ConfigProcessor {
	panic("not implemented")
}

func (s *mockGossipService) InitializeChannel(string, []string, service.Support) {
	panic("not implemented")
}

func (s *mockGossipService) GetBlock(chainID string, index uint64) *common.Block {
	panic("not implemented")
}

func (s *mockGossipService) AddPayload(chainID string, payload *gossip.Payload) error {
	panic("not implemented")
}

func (s *mockGossipService) DistributePrivateData(chainID string, txID string, privateData *rwset.TxPvtReadWriteSet) error {
	panic("not implemented")
}

type mockGossip struct {
	members  []mspNetworkMembers
	acceptor gcommon.MessageAcceptor
	msgCh    chan gossip.ReceivedMessage
}

func (s *mockGossip) Send(msg *gossip.GossipMessage, peers ...*comm.RemotePeer) {
	panic("not implemented")
}

func (s *mockGossip) PeerFilter(channel gcommon.ChainID, messagePredicate api.SubChannelSelectionCriteria) (filter.RoutingFilter, error) {
	panic("not implemented")
}

func (s *mockGossip) Peers() []discovery.NetworkMember {
	var members []discovery.NetworkMember
	for _, member := range s.members {
		members = append(members, member.networkMembers...)
	}
	return members
}

func (s *mockGossip) PeersOfChannel(gcommon.ChainID) []discovery.NetworkMember {
	var members []discovery.NetworkMember
	for _, member := range s.members {
		members = append(members, member.networkMembers...)
	}
	return members
}

func (s *mockGossip) UpdateMetadata(metadata []byte) {
	panic("not implemented")
}

func (s *mockGossip) SendByCriteria(signedGossipMessage *gossip.SignedGossipMessage, sendCriteria gossip2.SendCriteria) error {
	panic("not implemented")
}

func (s *mockGossip) UpdateChannelMetadata(metadata []byte, chainID gcommon.ChainID) {
	panic("not implemented")
}

func (s *mockGossip) Gossip(msg *gossip.GossipMessage) {
	panic("not implemented")
}

func (s *mockGossip) Accept(acceptor gcommon.MessageAcceptor, passThrough bool) (<-chan *gossip.GossipMessage, <-chan gossip.ReceivedMessage) {
	s.acceptor = acceptor
	s.msgCh = make(chan gossip.ReceivedMessage)
	return nil, s.msgCh
}

func (s *mockGossip) JoinChan(joinMsg api.JoinChannelMessage, chainID gcommon.ChainID) {
	panic("not implemented")
}

func (s *mockGossip) Stop() {
	if s.msgCh != nil {
		close(s.msgCh)
	}
}

func (s *mockGossip) SuspectPeers(api.PeerSuspector) {
	panic("not implemented")
}

func (s *mockGossip) GetOrgOfPeer(PKIID gcommon.PKIidType) api.OrgIdentityType {
	// TODO: This function is deprecated and should be removed
	panic("not implemented")
}

func (s *mockGossip) LeaveChan(chainID gcommon.ChainID) {
	panic("not implemented")
}

func newMockIdentity() []byte {
	return []byte("Some Identity")
}

func newMockSignedProposal(identity []byte) (*pb.SignedProposal, msp.IdentityDeserializer) {
	sProp, _ := utils.MockSignedEndorserProposalOrPanic("", &pb.ChaincodeSpec{}, identity, nil)
	sProp.Signature = sProp.ProposalBytes
	identityDeserializer := &policymocks.MockIdentityDeserializer{
		Identity: identity,
		Msg:      sProp.ProposalBytes,
	}
	return sProp, identityDeserializer
}

func newNetworkMember(pkiID gcommon.PKIidType, endpoint, internalEndpoint string) discovery.NetworkMember {
	return discovery.NetworkMember{
		PKIid:            pkiID,
		Endpoint:         endpoint,
		InternalEndpoint: internalEndpoint,
	}
}

// mspNetworkMembers contains an array of network members for a given MSP
type mspNetworkMembers struct {
	mspID          api.OrgIdentityType
	networkMembers []discovery.NetworkMember
}

func newMSPNetworkMembers(mspID []byte, networkMembers ...discovery.NetworkMember) mspNetworkMembers {
	return mspNetworkMembers{
		mspID:          mspID,
		networkMembers: networkMembers,
	}
}

func newIdentityMsg(pkiID gcommon.PKIidType, sID *msppb.SerializedIdentity) gossip.ReceivedMessage {
	return newReceivedMessage(newSignedGossipMessage(
		&gossip.GossipMessage{
			Channel: []byte("testchannel"),
			Tag:     gossip.GossipMessage_EMPTY,
			Content: newDataUpdateMsg(pkiID, sID),
		}))
}

func newAliveMsg() gossip.ReceivedMessage {
	return newReceivedMessage(newSignedGossipMessage(
		&gossip.GossipMessage{
			Channel: []byte("testchannel"),
			Tag:     gossip.GossipMessage_EMPTY,
			Content: &gossip.GossipMessage_AliveMsg{
				AliveMsg: &gossip.AliveMessage{},
			},
		}))
}

func newDataUpdateMsg(pkiID gcommon.PKIidType, sID *msppb.SerializedIdentity) *gossip.GossipMessage_DataUpdate {
	return &gossip.GossipMessage_DataUpdate{
		DataUpdate: &gossip.DataUpdate{
			MsgType: gossip.PullMsgType_IDENTITY_MSG,
			Nonce:   0,
			Data: []*gossip.Envelope{
				newEnvelope(marshal(
					&gossip.GossipMessage{
						Content: newPeerIdentityMsg(pkiID, sID),
					},
				)),
			},
		},
	}
}

func newEnvelope(payload []byte) *gossip.Envelope {
	return &gossip.Envelope{Payload: payload}
}

func newPeerIdentityMsg(pkiID gcommon.PKIidType, sID *msppb.SerializedIdentity) *gossip.GossipMessage_PeerIdentity {
	return &gossip.GossipMessage_PeerIdentity{
		PeerIdentity: &gossip.PeerIdentity{
			PkiId:    pkiID,
			Cert:     marshal(sID),
			Metadata: nil,
		},
	}
}

func newSignedGossipMessage(gossipMsg *gossip.GossipMessage) *gossip.SignedGossipMessage {
	return &gossip.SignedGossipMessage{
		GossipMessage: gossipMsg,
	}
}

type receivedMessage struct {
	gossipMsg *gossip.SignedGossipMessage
}

func newReceivedMessage(gossipMsg *gossip.SignedGossipMessage) *receivedMessage {
	return &receivedMessage{
		gossipMsg: gossipMsg,
	}
}

// Respond sends a GossipMessage to the origin from which this ReceivedMessage was sent from
func (m *receivedMessage) Respond(msg *gossip.GossipMessage) {
	panic("not implemented")
}

// GetGossipMessage returns the underlying GossipMessage
func (m *receivedMessage) GetGossipMessage() *gossip.SignedGossipMessage {
	return m.gossipMsg
}

// GetSourceMessage Returns the Envelope the ReceivedMessage was
// constructed with
func (m *receivedMessage) GetSourceEnvelope() *gossip.Envelope {
	panic("not implemented")
}

// GetConnectionInfo returns information about the remote peer
// that sent the message
func (m *receivedMessage) GetConnectionInfo() *gossip.ConnectionInfo {
	panic("not implemented")
}

// Ack returns to the sender an acknowledgement for the message
// An ack can receive an error that indicates that the operation related
// to the message has failed
func (m *receivedMessage) Ack(err error) {
	panic("not implemented")
}

type mockMSPIDMgr struct {
	mspIDMgr
}

func newmockMSPIDMgr(m map[string]string) *mockMSPIDMgr {
	return &mockMSPIDMgr{
		mspIDMgr: mspIDMgr{
			mspIDMap: m,
		},
	}
}

// GetMSPID returns the MSP ID for the given PKI ID
func (m *mockMSPIDMgr) GetMSPID(pkiID gcommon.PKIidType) string {
	mspID, ok := m.mspIDMap[string(pkiID)]
	if !ok {
		logger.Warningf("MSP ID not found for PKI ID [%s]", pkiID)
	}
	return mspID
}

func marshal(pb proto.Message) []byte {
	bytes, err := proto.Marshal(pb)
	if err != nil {
		panic(fmt.Sprintf("error marshalling gossip message: %s", err))
	}
	return bytes
}
