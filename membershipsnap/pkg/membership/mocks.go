/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package membership

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/gossip/api"
	"github.com/hyperledger/fabric/gossip/comm"
	gossip2 "github.com/hyperledger/fabric/gossip/gossip"

	"github.com/hyperledger/fabric/gossip/common"
	gcommon "github.com/hyperledger/fabric/gossip/common"
	"github.com/hyperledger/fabric/gossip/discovery"
	"github.com/hyperledger/fabric/gossip/filter"
	"github.com/hyperledger/fabric/gossip/protoext"
	"github.com/hyperledger/fabric/gossip/service"
	cb "github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/gossip"
	msppb "github.com/hyperledger/fabric/protos/msp"
	"github.com/hyperledger/fabric/protos/transientstore"
)

// NewServiceWithMocks creates a membership service with the given mocks.
// - self is the local peer
// - members contains zero or more MSP network members
func NewServiceWithMocks(localMSPID []byte, self discovery.NetworkMember, members ...MspNetworkMembers) *Service {
	m := make(map[string]string)
	for _, member := range members {
		for _, netMember := range member.NetworkMembers {
			m[string(netMember.PKIid)] = string(member.MspID)
		}
	}

	return newServiceWithOpts(
		localMSPID,
		newMockGossipService(self, members...),
		newmockMSPIDMgr(m),
	)
}

type mockGossipService struct {
	mockGossip
}

func newMockGossipService(self discovery.NetworkMember, members ...MspNetworkMembers) *mockGossipService {
	return &mockGossipService{
		mockGossip: mockGossip{
			self:    self,
			members: members,
		},
	}
}

func (s *mockGossipService) sendMessage(msg protoext.ReceivedMessage) {
	go func() {
		if s.acceptor(msg) {
			s.msgCh <- msg
		}
	}()
}

func (s *mockGossipService) NewConfigEventer() service.ConfigProcessor {
	panic("not implemented")
}

func (s *mockGossipService) InitializeChannel(string, service.OrdererAddressConfig, service.Support) {
	panic("not implemented")
}

func (s *mockGossipService) GetBlock(chainID string, index uint64) *cb.Block {
	panic("not implemented")
}

func (s *mockGossipService) AddPayload(chainID string, payload *gossip.Payload) error {
	panic("not implemented")
}

func (s *mockGossipService) DistributePrivateData(chainID string, txID string, privateData *transientstore.TxPvtReadWriteSetWithConfigInfo, blkHt uint64) error {
	panic("not implemented")
}

func (s *mockGossipService) IdentityInfo() api.PeerIdentitySet {
	panic("not implemented")
}

func (s *mockGossipService) SelfChannelInfo(common.ChainID) *protoext.SignedGossipMessage {
	return &protoext.SignedGossipMessage{
		GossipMessage: &gossip.GossipMessage{
			Content: &gossip.GossipMessage_StateInfo{
				StateInfo: &gossip.StateInfo{
					Properties: s.self.Properties,
				},
			},
		},
	}
}

func (s *mockGossipService) SelfMembershipInfo() discovery.NetworkMember {
	return s.self
}

func (s *mockGossipService) UpdateChaincodes(chaincode []*gossip.Chaincode, chainID common.ChainID) {
	panic("not implemented")
}

func (s *mockGossipService) UpdateLedgerHeight(height uint64, chainID common.ChainID) {
	panic("not implemented")
}

type mockGossip struct {
	self     discovery.NetworkMember
	members  []MspNetworkMembers
	acceptor gcommon.MessageAcceptor
	msgCh    chan protoext.ReceivedMessage
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
		members = append(members, member.NetworkMembers...)
	}
	return members
}

func (s *mockGossip) PeersOfChannel(gcommon.ChainID) []discovery.NetworkMember {
	var members []discovery.NetworkMember
	for _, member := range s.members {
		members = append(members, member.NetworkMembers...)
	}
	return members
}

func (s *mockGossip) UpdateMetadata(metadata []byte) {
	panic("not implemented")
}

func (s *mockGossip) SendByCriteria(signedGossipMessage *protoext.SignedGossipMessage, sendCriteria gossip2.SendCriteria) error {
	panic("not implemented")
}

func (s *mockGossip) UpdateChannelMetadata(metadata []byte, chainID gcommon.ChainID) {
	panic("not implemented")
}

func (s *mockGossip) Gossip(msg *gossip.GossipMessage) {
	panic("not implemented")
}

func (s *mockGossip) Accept(acceptor gcommon.MessageAcceptor, passThrough bool) (<-chan *gossip.GossipMessage, <-chan protoext.ReceivedMessage) {
	s.acceptor = acceptor
	s.msgCh = make(chan protoext.ReceivedMessage)
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

func (s *mockGossip) IsInMyOrg(member discovery.NetworkMember) bool {
	panic("not implemented")
}

func newMockIdentity() []byte { //nolint: deadcode
	return []byte("Some Identity")
}

// NewNetworkMember creates a new NetworkMember instance
func NewNetworkMember(pkiID gcommon.PKIidType, endpoint string) discovery.NetworkMember {
	return discovery.NetworkMember{
		PKIid:    pkiID,
		Endpoint: endpoint,
	}
}

// NewNetworkChannelMember creates a new NetworkMember instance
func NewNetworkChannelMember(pkiID gcommon.PKIidType, endpoint string, ledgerHeight uint64) discovery.NetworkMember {
	return discovery.NetworkMember{
		PKIid:    pkiID,
		Endpoint: endpoint,
		Properties: &gossip.Properties{
			LedgerHeight: ledgerHeight,
		},
	}
}

// NewLocalNetworkChannelMember creates a new NetworkMember instance for a local peer
func NewLocalNetworkChannelMember(endpoint string, ledgerHeight uint64) discovery.NetworkMember {
	return discovery.NetworkMember{
		Endpoint: endpoint,
		Properties: &gossip.Properties{
			LedgerHeight: ledgerHeight,
		},
	}
}

// MspNetworkMembers contains an array of network members for a given MSP
type MspNetworkMembers struct {
	MspID          api.OrgIdentityType
	NetworkMembers []discovery.NetworkMember
}

// NewMSPNetworkMembers creates a new MspNetworkMembers instance
func NewMSPNetworkMembers(mspID []byte, networkMembers ...discovery.NetworkMember) MspNetworkMembers {
	return MspNetworkMembers{
		MspID:          mspID,
		NetworkMembers: networkMembers,
	}
}

func newIdentityMsg(pkiID gcommon.PKIidType, sID *msppb.SerializedIdentity) protoext.ReceivedMessage { //nolint: deadcode , interfacer
	return newReceivedMessage(newSignedGossipMessage(
		&gossip.GossipMessage{
			Channel: []byte("testchannel"),
			Tag:     gossip.GossipMessage_EMPTY,
			Content: newDataUpdateMsg(pkiID, sID),
		}))
}

func newAliveMsg() protoext.ReceivedMessage { //nolint: deadcode
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

func newSignedGossipMessage(gossipMsg *gossip.GossipMessage) *protoext.SignedGossipMessage {
	return &protoext.SignedGossipMessage{
		GossipMessage: gossipMsg,
	}
}

type receivedMessage struct {
	gossipMsg *protoext.SignedGossipMessage
}

func newReceivedMessage(gossipMsg *protoext.SignedGossipMessage) *receivedMessage {
	return &receivedMessage{
		gossipMsg: gossipMsg,
	}
}

// Respond sends a GossipMessage to the origin from which this ReceivedMessage was sent from
func (m *receivedMessage) Respond(msg *gossip.GossipMessage) {
	panic("not implemented")
}

// GetGossipMessage returns the underlying GossipMessage
func (m *receivedMessage) GetGossipMessage() *protoext.SignedGossipMessage {
	return m.gossipMsg
}

// GetSourceMessage Returns the Envelope the ReceivedMessage was
// constructed with
func (m *receivedMessage) GetSourceEnvelope() *gossip.Envelope {
	panic("not implemented")
}

// GetConnectionInfo returns information about the remote peer
// that sent the message
func (m *receivedMessage) GetConnectionInfo() *protoext.ConnectionInfo {
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
		logger.Warnf("MSP ID not found for PKI ID [%s]", pkiID)
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
