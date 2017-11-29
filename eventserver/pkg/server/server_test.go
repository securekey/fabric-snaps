/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package server

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric/common/crypto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/localmsp"
	"github.com/hyperledger/fabric/common/util"
	"github.com/hyperledger/fabric/core/aclmgmt"
	"github.com/hyperledger/fabric/core/config"
	"github.com/hyperledger/fabric/core/ledger"
	coreutil "github.com/hyperledger/fabric/core/testutil"
	"github.com/hyperledger/fabric/msp"
	"github.com/hyperledger/fabric/msp/mgmt"
	"github.com/hyperledger/fabric/msp/mgmt/testtools"
	"github.com/hyperledger/fabric/protos/common"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/hyperledger/fabric/protos/utils"
	eventserverapi "github.com/securekey/fabric-snaps/eventserver/api"
	"github.com/securekey/fabric-snaps/eventserver/pkg/mocks"
	client "github.com/securekey/fabric-snaps/eventserver/pkg/server/testclient"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type Adapter struct {
	sync.RWMutex
	notfy chan struct{}
	count int
}

type mockACLProvider struct {
	retErr error
}

func (m *mockACLProvider) CheckACL(resName string, channelID string, idinfo interface{}) error {
	if channelID == "restrictedchannel" {
		return m.retErr
	}
	if strings.Contains(channelID, "filteredchannel") && resName == aclmgmt.BLOCKEVENT {
		return m.retErr
	}
	return nil
}

func (m *mockACLProvider) GenerateSimulationResults(txEnvelop *common.Envelope, simulator ledger.TxSimulator) error {
	return nil
}

var csServer *ChannelServer
var csClient *client.ChannelClient
var adapter *Adapter
var peerAddress = "0.0.0.0:60303"
var interestedChannels = []string{"testchainid"}
var interestedEvents = []*pb.Interest{&pb.Interest{EventType: pb.EventType_BLOCK},
	&pb.Interest{EventType: pb.EventType_FILTEREDBLOCK}}
var testLogger = flogging.MustGetLogger("test")

//GetInterestedChannels implements client.ChannelAdapter interface for
//registering interested channels
func (a *Adapter) GetInterestedChannels() ([]string, error) {
	return interestedChannels, nil
}

func (a *Adapter) GetInterestedEvents() ([]*pb.Interest, error) {
	return interestedEvents, nil
}

func (a *Adapter) updateCountNotify() {
	a.Lock()
	a.count--
	if a.count <= 0 {
		a.notfy <- struct{}{}
	}
	a.Unlock()
}

func (a *Adapter) Recv(msg *eventserverapi.ChannelServiceResponse) (bool, error) {
	switch x := msg.Response.(type) {
	case *eventserverapi.ChannelServiceResponse_Event, *eventserverapi.ChannelServiceResponse_Result:
		a.updateCountNotify()
	case nil:
		// The field is not set.
		return false, fmt.Errorf("event not set")
	default:
		return false, fmt.Errorf("unexpected type %T", x)
	}
	return true, nil
}

func (a *Adapter) Disconnected(err error) {
	if err != nil {
		testLogger.Errorf("Error: %s\n", err)
	}
}

var r *rand.Rand

func corrupt(bytes []byte) {
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().Unix()))
	}

	bytes[r.Int31n(int32(len(bytes)))]--
}

type testCase struct {
	name                      string
	register                  bool
	expectRegistrationSuccess bool
	interestedChannels        []string
	interestedEvents          []*pb.Interest
	expectedEventTypes        []pb.EventType
	timestamp                 *timestamp.Timestamp
}

func TestChannelServiceRegisterAndReceive(t *testing.T) {
	var tc []testCase
	interestBandFBEvent := []*pb.Interest{&pb.Interest{EventType: pb.EventType_BLOCK}, &pb.Interest{EventType: pb.EventType_FILTEREDBLOCK}}
	interestBEvent := []*pb.Interest{&pb.Interest{EventType: pb.EventType_BLOCK}}
	interestFBEvent := []*pb.Interest{&pb.Interest{EventType: pb.EventType_FILTEREDBLOCK}}
	noInterest := []*pb.Interest{}

	tc = append(tc,
		testCase{"testchainid", false, true, []string{"testchainid"}, interestBandFBEvent, []pb.EventType{pb.EventType_BLOCK, pb.EventType_FILTEREDBLOCK}, util.CreateUtcTimestamp()},
		testCase{"bchannel", true, true, []string{"bchannel"}, interestBEvent, []pb.EventType{pb.EventType_BLOCK}, util.CreateUtcTimestamp()},
		testCase{"disinterested", true, true, []string{"disinterested"}, noInterest, []pb.EventType{}, util.CreateUtcTimestamp()},
		testCase{"filteredchannel", true, true, []string{"filteredchannel"}, interestFBEvent, []pb.EventType{pb.EventType_FILTEREDBLOCK}, util.CreateUtcTimestamp()},
		testCase{"restrictedchannel", true, true, []string{"restrictedchannel"}, interestBandFBEvent, []pb.EventType{}, util.CreateUtcTimestamp()},
		testCase{"registeroutoftimewindow", true, false, []string{"timewindowchannel"}, interestBEvent, nil, &timestamp.Timestamp{Seconds: 0}},
	)

	for i := 0; i < len(tc); i++ {
		bevent, fbevent := createBlockEventsForTesting(t, tc[i].interestedChannels[0])
		if len(tc[i].interestedChannels) <= 1 {
			if tc[i].register && tc[i].expectRegistrationSuccess {
				t.Run(tc[i].name, func(t *testing.T) {
					registerAndReceiveEventsSingleChannel(t, tc[i].interestedChannels, tc[i].interestedEvents, tc[i].expectedEventTypes, bevent, fbevent, tc[i].timestamp, tc[i].expectRegistrationSuccess)
				})
			} else if tc[i].register && !tc[i].expectRegistrationSuccess {
				registerSingleChannel(t, nil, nil, tc[i].timestamp, tc[i].expectRegistrationSuccess)
			} else {
				t.Run(tc[i].name, func(t *testing.T) {
					receiveEventsSingleChannel(t, tc[i].interestedChannels, tc[i].interestedEvents, tc[i].expectedEventTypes, bevent, fbevent)
				})
			}
		}
	}
}

func registerSingleChannel(t *testing.T, ic []string, ie []*pb.Interest, timestamp *timestamp.Timestamp, expectRegistrationSuccess bool) {
	csreq := &eventserverapi.ChannelServiceRequest{Request: &eventserverapi.ChannelServiceRequest_RegisterChannel{RegisterChannel: &eventserverapi.RegisterChannel{ChannelIds: ic, Events: ie}}}

	// typical path
	if expectRegistrationSuccess {
		adapter.count = 1
		if err := csClient.Send(csreq); err != nil {
			t.Fail()
			t.Logf("Error sending register channel event: %s", err)
		}

		select {
		case <-adapter.notfy:
		case <-time.After(2 * time.Second):
			t.Fail()
			t.Logf("timed out on message")
		}
	} else {
		// force timestamp out of window in envelope
		env, _ := csClient.CreateEnvelopeForChannelServiceRequest(csreq)
		env = overwriteTimestampInEnvelope(env, timestamp, localmsp.NewSigner())

		adapter.count = 1
		if err := csClient.SendEnvelope(env); err != nil {
			t.Fail()
			t.Logf("Error sending register channel event: %s", err)
		}

		select {
		case e := <-adapter.notfy:
			t.Fail()
			t.Logf("received message: %v", e)
		case <-time.After(5 * time.Second):
		}
	}

}

func receiveEventsSingleChannel(t *testing.T, ic []string, ie []*pb.Interest, expectedEventTypes []pb.EventType, bevent *pb.Event, fbevent *pb.Event) {
	adapter.count = 1

	if err := csServer.Send(bevent); err != nil {
		t.Fail()
		t.Logf("Error sending block event: %s", err)
	}

	if contains(expectedEventTypes, pb.EventType_BLOCK) {
		select {
		case <-adapter.notfy:
		case <-time.After(2 * time.Second):
			t.Fail()
			t.Logf("timed out on message")
		}
	} else {
		select {
		case <-adapter.notfy:
			t.Logf("received block event when not expected")
			t.FailNow()
		case <-time.After(2 * time.Second):
		}
	}

	adapter.count = 1
	fb := fbevent.Event.(*pb.Event_FilteredBlock).FilteredBlock
	fb.ChannelId = ic[0]
	if err := csServer.Send(fbevent); err != nil {
		t.Fail()
		t.Logf("Error sending filtered block event: %s", err)
	}

	if contains(expectedEventTypes, pb.EventType_FILTEREDBLOCK) {
		select {
		case <-adapter.notfy:
		case <-time.After(2 * time.Second):
			t.Fail()
			t.Logf("timed out on message")
		}
	} else {
		select {
		case <-adapter.notfy:
			t.Logf("received filtered block event when not expected")
			t.Fatal()
		case <-time.After(2 * time.Second):
		}
	}
}
func registerAndReceiveEventsSingleChannel(t *testing.T, ic []string, ie []*pb.Interest, expectedEventTypes []pb.EventType, bevent *pb.Event, fbevent *pb.Event, timestamp *timestamp.Timestamp, expectRegistrationSuccess bool) {
	registerSingleChannel(t, ic, ie, timestamp, expectRegistrationSuccess)
	receiveEventsSingleChannel(t, ic, ie, expectedEventTypes, bevent, fbevent)
}

func contains(expectedEventTypes []pb.EventType, et pb.EventType) bool {
	for _, a := range expectedEventTypes {
		if a == et {
			return true
		}
	}
	return false
}

func createBlockEventsForTesting(t *testing.T, channelID string) (bevent *pb.Event, fbevent *pb.Event) {
	return mocks.NewMockBlockEvent(channelID), mocks.NewMockFilteredBlockEvent(channelID)
}

type testCaseDeregister struct {
	name                        string
	interestedChannels          []string
	timestamp                   *timestamp.Timestamp
	expectDeregistrationSuccess bool
}

func TestChannelServiceDeregister(t *testing.T) {
	var tc []testCaseDeregister

	tc = append(tc,
		testCaseDeregister{"deregisterfakechannel", []string{"fakechannel"}, util.CreateUtcTimestamp(), false},
		testCaseDeregister{"testchainid", []string{"testchainid"}, util.CreateUtcTimestamp(), true},
	)

	for i := 0; i < len(tc); i++ {
		t.Run(tc[i].name, func(t *testing.T) {
			deregisterSingleChannel(t, tc[i].interestedChannels, tc[i].timestamp, tc[i].expectDeregistrationSuccess)
		})

	}
}
func deregisterSingleChannel(t *testing.T, ic []string, timestamp *timestamp.Timestamp, expectDeregistrationSuccess bool) {
	var listenerCount int
	if expectDeregistrationSuccess {
		listenerCount = len(csServer.gEventProcessor.registeredListeners[ic[0]].handlers)
	}

	emsg := &eventserverapi.ChannelServiceRequest{Request: &eventserverapi.ChannelServiceRequest_DeregisterChannel{DeregisterChannel: &eventserverapi.DeregisterChannel{ChannelIds: ic}}}
	adapter.count = 1
	if err := csClient.Send(emsg); err != nil {
		t.Fail()
		t.Logf("Error sending deregister channel event: %s", err)
	}

	select {
	case <-adapter.notfy:
	case <-time.After(5 * time.Second):
		t.Fail()
		t.Logf("timed out on message")
	}
	if expectDeregistrationSuccess {
		assert.Equal(t, listenerCount-1, len(csServer.gEventProcessor.registeredListeners[ic[0]].handlers), "Listener count should have decreased by one")
	}
}

func TestEventProcessor(t *testing.T) {
	test := func(duration time.Duration) {
		t.Log(duration)
		f := func() {
			csServer.Send(nil)
		}
		assert.Panics(t, f)
		csServer.Send(&pb.Event{})
		gEventProcessorBck := csServer.gEventProcessor
		csServer.gEventProcessor = nil
		bevent, _ := createBlockEventsForTesting(t, "channelid")
		csServer.Send(bevent)
		csServer.gEventProcessor = gEventProcessorBck
		csServer.Send(bevent)
		bevent.Event = nil
		csServer.Send(bevent)
	}
	prevTimeout := csServer.gEventProcessor.timeout
	for _, timeout := range []time.Duration{0, -1, 1} {
		csServer.gEventProcessor.timeout = timeout
		test(timeout)
	}
	csServer.gEventProcessor.timeout = prevTimeout
}

type mockstream struct {
	c chan *streamEnvelope
}

type streamEnvelope struct {
	envelope *common.Envelope
	err      error
}

func (*mockstream) Send(*eventserverapi.ChannelServiceResponse) error {
	return nil
}

func (s *mockstream) Recv() (*common.Envelope, error) {
	se := <-s.c
	if se.err == nil {
		return se.envelope, nil
	}
	return nil, se.err
}

func (*mockstream) SetHeader(metadata.MD) error {
	panic("not implemented")
}

func (*mockstream) SendHeader(metadata.MD) error {
	panic("not implemented")
}

func (*mockstream) SetTrailer(metadata.MD) {
	panic("not implemented")
}

func (*mockstream) Context() context.Context {
	panic("not implemented")
}

func (*mockstream) SendMsg(m interface{}) error {
	panic("not implemented")
}

func (*mockstream) RecvMsg(m interface{}) error {
	panic("not implemented")
}

func TestChat(t *testing.T) {
	recvChan := make(chan *streamEnvelope)
	stream := &mockstream{c: recvChan}
	go csServer.Chat(stream)
	recvChan <- &streamEnvelope{envelope: &common.Envelope{}}
	go csServer.Chat(stream)
	emptyRequestEnv, _ := csClient.CreateEnvelopeForChannelServiceRequest(&eventserverapi.ChannelServiceRequest{})
	recvChan <- &streamEnvelope{envelope: emptyRequestEnv}
	go csServer.Chat(stream)
	recvChan <- &streamEnvelope{err: io.EOF}
	go csServer.Chat(stream)
	recvChan <- &streamEnvelope{err: errors.New("err")}
	time.Sleep(time.Second)
}

func overwriteTimestampInEnvelope(envelope *common.Envelope, timestamp *timestamp.Timestamp, signer crypto.LocalSigner) *common.Envelope {
	payload, _ := utils.UnmarshalPayload(envelope.Payload)
	chdr, _ := utils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
	chdr.Timestamp = timestamp
	chdrBytes, _ := proto.Marshal(chdr)
	header := &common.Header{ChannelHeader: chdrBytes, SignatureHeader: payload.Header.SignatureHeader}

	payload.Header = header
	paylBytes, _ := proto.Marshal(payload)
	var sig []byte
	if signer != nil {
		sig, _ = signer.Sign(paylBytes)
	}

	return &common.Envelope{Payload: paylBytes, Signature: sig}
}

var signer msp.SigningIdentity
var signerSerialized []byte

func TestMain(m *testing.M) {
	os.Setenv("GOPATH", "../../cmd/test/fixtures")

	mockACLProvider := &mockACLProvider{retErr: fmt.Errorf("badacl")}
	aclmgmt.RegisterACLProvider(mockACLProvider)
	// setup crypto algorithms
	// setup the MSP manager so that we can sign/verify
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		fmt.Printf("Could not initialize msp, err %s", err)
		os.Exit(-1)
		return
	}

	signer, err = mgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		fmt.Println("Could not get signer")
		os.Exit(-1)
		return
	}

	signerSerialized, err = signer.Serialize()
	if err != nil {
		fmt.Println("Could not serialize identity")
		os.Exit(-1)
		return
	}
	coreutil.SetupTestConfig()
	var opts []grpc.ServerOption
	if viper.GetBool("peer.tls.enabled") {
		creds, err := credentials.NewServerTLSFromFile(config.GetPath("peer.tls.cert.file"), config.GetPath("peer.tls.key.file"))
		if err != nil {
			grpclog.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)

	//use a different address than what we usually use for "peer"
	//we override the peerAddress set in chaincode_support.go
	peerAddress = "0.0.0.0:60303"

	lis, err := net.Listen("tcp", peerAddress)
	if err != nil {
		fmt.Printf("Error starting events listener %s....not doing tests", err)
		return
	}

	// Register channel service server
	// use a buffer of 100 and blocking timeout
	viper.Set("peer.channelservice.buffersize", 100)
	viper.Set("peer.channelservice.timeout", 0)
	timeWindow, _ := time.ParseDuration("15m")
	viper.Set("peer.channelservice.timewindow", timeWindow)

	csConfig := &ChannelServerConfig{
		BufferSize: uint(viper.GetInt("peer.channelservice.buffersize")),
		Timeout:    viper.GetDuration("peer.channelservice.timeout"),
		TimeWindow: viper.GetDuration("peer.channelservice.timewindow"),
	}
	csServer = NewChannelServer(csConfig)
	// csprovider.RegisterChannelServiceProvider(csServer)
	eventserverapi.RegisterChannelServer(grpcServer, csServer)

	go grpcServer.Serve(lis)

	var regTimeout = 5 * time.Second
	done := make(chan struct{})
	adapter = &Adapter{notfy: done}
	csClient, _ = client.NewChannelClient(peerAddress, regTimeout, adapter)
	if err = csClient.Start(); err != nil {
		fmt.Printf("could not start chat %v\n", err)
		csClient.Stop()
		return
	}

	time.Sleep(2 * time.Second)
	os.Exit(m.Run())
}
