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
/*
Notice: This file has been modified for Hyperledger Fabric SDK Go usage.
Please review third_party pinning scripts and patches for more details.
*/

package shim

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/errors"
	pb "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/protos/peer"
	"github.com/looplab/fsm"
	pbinternal "github.com/securekey/fabric-snaps/internal/github.com/hyperledger/fabric/protos/peer"
)

// PeerChaincodeStream interface for stream between Peer and chaincode instance.
type PeerChaincodeStream interface {
	Send(*pbinternal.ChaincodeMessage) error
	Recv() (*pbinternal.ChaincodeMessage, error)
	CloseSend() error
}

type nextStateInfo struct {
	msg      *pbinternal.ChaincodeMessage
	sendToCC bool
}

func (handler *Handler) triggerNextState(msg *pbinternal.ChaincodeMessage, send bool) {
	handler.nextState <- &nextStateInfo{msg, send}
}

// Handler handler implementation for shim side of chaincode.
type Handler struct {
	sync.RWMutex
	//shim to peer grpc serializer. User only in serialSend
	serialLock sync.Mutex
	To         string
	ChatStream PeerChaincodeStream
	FSM        *fsm.FSM
	cc         Chaincode
	// Multiple queries (and one transaction) with different txids can be executing in parallel for this chaincode
	// responseChannel is the channel on which responses are communicated by the shim to the chaincodeStub.
	responseChannel map[string]chan pbinternal.ChaincodeMessage
	nextState       chan *nextStateInfo
}

func shorttxid(txid string) string {
	if len(txid) < 8 {
		return txid
	}
	return txid[0:8]
}

//serialSend serializes msgs so gRPC will be happy
func (handler *Handler) serialSend(msg *pbinternal.ChaincodeMessage) error {
	handler.serialLock.Lock()
	defer handler.serialLock.Unlock()

	err := handler.ChatStream.Send(msg)

	return err
}

//serialSendAsync serves the same purpose as serialSend (serialize msgs so gRPC will
//be happy). In addition, it is also asynchronous so send-remoterecv--localrecv loop
//can be nonblocking. Only errors need to be handled and these are handled by
//communication on supplied error channel. A typical use will be a non-blocking or
//nil channel
func (handler *Handler) serialSendAsync(msg *pbinternal.ChaincodeMessage, errc chan error) {
	go func() {
		err := handler.serialSend(msg)
		if errc != nil {
			errc <- err
		}
	}()
}

func (handler *Handler) createChannel(txid string) (chan pbinternal.ChaincodeMessage, error) {
	handler.Lock()
	defer handler.Unlock()
	if handler.responseChannel == nil {
		return nil, errors.Errorf("[%s]cannot create response channel", shorttxid(txid))
	}
	if handler.responseChannel[txid] != nil {
		return nil, errors.Errorf("[%s]channel exists", shorttxid(txid))
	}
	c := make(chan pbinternal.ChaincodeMessage)
	handler.responseChannel[txid] = c
	return c, nil
}

func (handler *Handler) sendChannel(msg *pbinternal.ChaincodeMessage) error {
	handler.Lock()
	defer handler.Unlock()
	if handler.responseChannel == nil {
		return errors.Errorf("[%s]Cannot send message response channel", shorttxid(msg.Txid))
	}
	if handler.responseChannel[msg.Txid] == nil {
		return errors.Errorf("[%s]sendChannel does not exist", shorttxid(msg.Txid))
	}

	chaincodeLogger.Debugf("[%s]before send", shorttxid(msg.Txid))
	handler.responseChannel[msg.Txid] <- *msg
	chaincodeLogger.Debugf("[%s]after send", shorttxid(msg.Txid))

	return nil
}

//sends a message and selects
func (handler *Handler) sendReceive(msg *pbinternal.ChaincodeMessage, c chan pbinternal.ChaincodeMessage) (pbinternal.ChaincodeMessage, error) {
	errc := make(chan error, 1)
	handler.serialSendAsync(msg, errc)

	//the serialsend above will send an err or nil
	//the select filters that first error(or nil)
	//and continues to wait for the response
	//it is possible that the response triggers first
	//in which case the errc obviously worked and is
	//ignored
	for {
		select {
		case err := <-errc:
			if err == nil {
				continue
			}
			//would have been logged, return false
			return pbinternal.ChaincodeMessage{}, err
		case outmsg, val := <-c:
			if !val {
				return pbinternal.ChaincodeMessage{}, errors.New("unexpected failure on receive")
			}
			return outmsg, nil
		}
	}
}

func (handler *Handler) deleteChannel(txid string) {
	handler.Lock()
	defer handler.Unlock()
	if handler.responseChannel != nil {
		delete(handler.responseChannel, txid)
	}
}

// NewChaincodeHandler returns a new instance of the shim side handler.
func newChaincodeHandler(peerChatStream PeerChaincodeStream, chaincode Chaincode) *Handler {
	v := &Handler{
		ChatStream: peerChatStream,
		cc:         chaincode,
	}
	v.responseChannel = make(map[string]chan pbinternal.ChaincodeMessage)
	v.nextState = make(chan *nextStateInfo)

	// Create the shim side FSM
	v.FSM = fsm.NewFSM(
		"created",
		fsm.Events{
			{Name: pbinternal.ChaincodeMessage_REGISTERED.String(), Src: []string{"created"}, Dst: "established"},
			{Name: pbinternal.ChaincodeMessage_READY.String(), Src: []string{"established"}, Dst: "ready"},
			{Name: pbinternal.ChaincodeMessage_ERROR.String(), Src: []string{"init"}, Dst: "established"},
			{Name: pbinternal.ChaincodeMessage_RESPONSE.String(), Src: []string{"init"}, Dst: "init"},
			{Name: pbinternal.ChaincodeMessage_INIT.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pbinternal.ChaincodeMessage_TRANSACTION.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pbinternal.ChaincodeMessage_RESPONSE.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pbinternal.ChaincodeMessage_ERROR.String(), Src: []string{"ready"}, Dst: "ready"},
			{Name: pbinternal.ChaincodeMessage_COMPLETED.String(), Src: []string{"init"}, Dst: "ready"},
			{Name: pbinternal.ChaincodeMessage_COMPLETED.String(), Src: []string{"ready"}, Dst: "ready"},
		},
		fsm.Callbacks{
			"before_" + pbinternal.ChaincodeMessage_REGISTERED.String():  func(e *fsm.Event) { v.beforeRegistered(e) },
			"after_" + pbinternal.ChaincodeMessage_RESPONSE.String():     func(e *fsm.Event) { v.afterResponse(e) },
			"after_" + pbinternal.ChaincodeMessage_ERROR.String():        func(e *fsm.Event) { v.afterError(e) },
			"before_" + pbinternal.ChaincodeMessage_INIT.String():        func(e *fsm.Event) { v.beforeInit(e) },
			"before_" + pbinternal.ChaincodeMessage_TRANSACTION.String(): func(e *fsm.Event) { v.beforeTransaction(e) },
		},
	)
	return v
}

// beforeRegistered is called to handle the REGISTERED message.
func (handler *Handler) beforeRegistered(e *fsm.Event) {
	if _, ok := e.Args[0].(*pbinternal.ChaincodeMessage); !ok {
		e.Cancel(errors.New("Received unexpected message type"))
		return
	}
	chaincodeLogger.Debugf("Received %s, ready for invocations", pbinternal.ChaincodeMessage_REGISTERED)
}

// handleInit handles request to initialize chaincode.
func (handler *Handler) handleInit(msg *pbinternal.ChaincodeMessage) {
	// The defer followed by triggering a go routine dance is needed to ensure that the previous state transition
	// is completed before the next one is triggered. The previous state transition is deemed complete only when
	// the beforeInit function is exited. Interesting bug fix!!
	go func() {
		var nextStateMsg *pbinternal.ChaincodeMessage

		send := true

		defer func() {
			handler.triggerNextState(nextStateMsg, send)
		}()

		errFunc := func(err error, payload []byte, ce *pb.ChaincodeEvent, errFmt string, args ...interface{}) *pbinternal.ChaincodeMessage {
			if err != nil {
				// Send ERROR message to chaincode support and change state
				if payload == nil {
					payload = []byte(err.Error())
				}
				chaincodeLogger.Errorf(errFmt, args...)
				return &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_ERROR, Payload: payload, Txid: msg.Txid, ChaincodeEvent: ce}
			}
			return nil
		}
		// Get the function and args from Payload
		input := &pb.ChaincodeInput{}
		unmarshalErr := proto.Unmarshal(msg.Payload, input)
		if nextStateMsg = errFunc(unmarshalErr, nil, nil, "[%s]Incorrect payload format. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Call chaincode's Run
		// Create the ChaincodeStub which the chaincode can use to callback
		stub := new(ChaincodeStub)
		err := stub.init(handler, msg.Txid, input, msg.Proposal)
		if nextStateMsg = errFunc(err, nil, stub.chaincodeEvent, "[%s]Init get error response [%s]. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		res := handler.cc.Init(stub)
		chaincodeLogger.Debugf("[%s]Init get response status: %d", shorttxid(msg.Txid), res.Status)

		if res.Status >= ERROR {
			err = errors.New(res.Message)
			if nextStateMsg = errFunc(err, []byte(res.Message), stub.chaincodeEvent, "[%s]Init get error response [%s]. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
				return
			}
		}

		resBytes, err := proto.Marshal(&res)
		if nextStateMsg = errFunc(err, nil, stub.chaincodeEvent, "[%s]Init marshal response error [%s]. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Send COMPLETED message to chaincode support and change state
		nextStateMsg = &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_COMPLETED, Payload: resBytes, Txid: msg.Txid, ChaincodeEvent: stub.chaincodeEvent}
		chaincodeLogger.Debugf("[%s]Init succeeded. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_COMPLETED)
	}()
}

// beforeInit will initialize the chaincode if entering init from established.
func (handler *Handler) beforeInit(e *fsm.Event) {
	chaincodeLogger.Debugf("Entered state %s", handler.FSM.Current())
	msg, ok := e.Args[0].(*pbinternal.ChaincodeMessage)
	if !ok {
		e.Cancel(errors.New("received unexpected message type"))
		return
	}
	chaincodeLogger.Debugf("[%s]Received %s, initializing chaincode", shorttxid(msg.Txid), msg.Type.String())
	if msg.Type.String() == pbinternal.ChaincodeMessage_INIT.String() {
		// Call the chaincode's Run function to initialize
		handler.handleInit(msg)
	}
}

// handleTransaction Handles request to execute a transaction.
func (handler *Handler) handleTransaction(msg *pbinternal.ChaincodeMessage) {
	// The defer followed by triggering a go routine dance is needed to ensure that the previous state transition
	// is completed before the next one is triggered. The previous state transition is deemed complete only when
	// the beforeInit function is exited. Interesting bug fix!!
	go func() {
		//better not be nil
		var nextStateMsg *pbinternal.ChaincodeMessage

		send := true

		defer func() {
			handler.triggerNextState(nextStateMsg, send)
		}()

		errFunc := func(err error, ce *pb.ChaincodeEvent, errStr string, args ...interface{}) *pbinternal.ChaincodeMessage {
			if err != nil {
				payload := []byte(err.Error())
				chaincodeLogger.Errorf(errStr, args...)
				return &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_ERROR, Payload: payload, Txid: msg.Txid, ChaincodeEvent: ce}
			}
			return nil
		}

		// Get the function and args from Payload
		input := &pb.ChaincodeInput{}
		unmarshalErr := proto.Unmarshal(msg.Payload, input)
		if nextStateMsg = errFunc(unmarshalErr, nil, "[%s]Incorrect payload format. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Call chaincode's Run
		// Create the ChaincodeStub which the chaincode can use to callback
		stub := new(ChaincodeStub)
		err := stub.init(handler, msg.Txid, input, msg.Proposal)
		if nextStateMsg = errFunc(err, stub.chaincodeEvent, "[%s]Transaction execution failed. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		res := handler.cc.Invoke(stub)

		// Endorser will handle error contained in Response.
		resBytes, err := proto.Marshal(&res)
		if nextStateMsg = errFunc(err, stub.chaincodeEvent, "[%s]Transaction execution failed. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_ERROR.String()); nextStateMsg != nil {
			return
		}

		// Send COMPLETED message to chaincode support and change state
		chaincodeLogger.Debugf("[%s]Transaction completed. Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_COMPLETED)
		nextStateMsg = &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_COMPLETED, Payload: resBytes, Txid: msg.Txid, ChaincodeEvent: stub.chaincodeEvent}
	}()
}

// beforeTransaction will execute chaincode's Run if coming from a TRANSACTION event.
func (handler *Handler) beforeTransaction(e *fsm.Event) {
	msg, ok := e.Args[0].(*pbinternal.ChaincodeMessage)
	if !ok {
		e.Cancel(errors.New("Received unexpected message type"))
		return
	}
	chaincodeLogger.Debugf("[%s]Received %s, invoking transaction on chaincode(Src:%s, Dst:%s)", shorttxid(msg.Txid), msg.Type.String(), e.Src, e.Dst)
	if msg.Type.String() == pbinternal.ChaincodeMessage_TRANSACTION.String() {
		// Call the chaincode's Run function to invoke transaction
		handler.handleTransaction(msg)
	}
}

// afterResponse is called to deliver a response or error to the chaincode stub.
func (handler *Handler) afterResponse(e *fsm.Event) {
	msg, ok := e.Args[0].(*pbinternal.ChaincodeMessage)
	if !ok {
		e.Cancel(errors.New("received unexpected message type"))
		return
	}

	if err := handler.sendChannel(msg); err != nil {
		chaincodeLogger.Errorf("[%s]error sending %s (state:%s): %s", shorttxid(msg.Txid), msg.Type, handler.FSM.Current(), err)
	} else {
		chaincodeLogger.Debugf("[%s]Received %s, communicated (state:%s)", shorttxid(msg.Txid), msg.Type, handler.FSM.Current())
	}
}

func (handler *Handler) afterError(e *fsm.Event) {
	msg, ok := e.Args[0].(*pbinternal.ChaincodeMessage)
	if !ok {
		e.Cancel(errors.New("Received unexpected message type"))
		return
	}

	/* TODO- revisit. This may no longer be needed with the serialized/streamlined messaging model
	 * There are two situations in which the ERROR event can be triggered:
	 * 1. When an error is encountered within handleInit or handleTransaction - some issue at the chaincode side; In this case there will be no responseChannel and the message has been sent to the validator.
	 * 2. The chaincode has initiated a request (get/put/del state) to the validator and is expecting a response on the responseChannel; If ERROR is received from validator, this needs to be notified on the responseChannel.
	 */
	if err := handler.sendChannel(msg); err == nil {
		chaincodeLogger.Debugf("[%s]Error received from validator %s, communicated(state:%s)", shorttxid(msg.Txid), msg.Type, handler.FSM.Current())
	}
}

// TODO: Implement method to get and put entire state map and not one key at a time?
// handleGetState communicates with the validator to fetch the requested state information from the ledger.
func (handler *Handler) handleGetState(key string, txid string) ([]byte, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send GET_STATE message to validator chaincode support
	payload := []byte(key)
	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_GET_STATE, Payload: payload, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_STATE)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("[%s]error sending GET_STATE", shorttxid(txid)))
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]GetState received payload %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)
		return responseMsg.Payload, nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]GetState received error %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.Errorf("[%s]incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

// handlePutState communicates with the validator to put state information into the ledger.
func (handler *Handler) handlePutState(key string, value []byte, txid string) error {
	// Check if this is a transaction
	chaincodeLogger.Debugf("[%s]Inside putstate", shorttxid(txid))

	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pbinternal.PutStateInfo{Key: key, Value: value})

	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return err
	}

	defer handler.deleteChannel(txid)

	// Send PUT_STATE message to validator chaincode support
	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_PUT_STATE, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_PUT_STATE)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return errors.WithMessage(err, fmt.Sprintf("[%s]error sending PUT_STATE", msg.Txid))
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully updated state", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)
		return nil
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s. Payload: %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR, responseMsg.Payload)
		return errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return errors.Errorf("[%s]incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

// handleDelState communicates with the validator to delete a key from the state in the ledger.
func (handler *Handler) handleDelState(key string, txid string) error {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return err
	}

	defer handler.deleteChannel(txid)

	// Send DEL_STATE message to validator chaincode support
	payload := []byte(key)
	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_DEL_STATE, Payload: payload, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_DEL_STATE)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return errors.Errorf("[%s]error sending DEL_STATE %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_DEL_STATE)
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully deleted state", msg.Txid, pbinternal.ChaincodeMessage_RESPONSE)
		return nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s. Payload: %s", msg.Txid, pbinternal.ChaincodeMessage_ERROR, responseMsg.Payload)
		return errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return errors.Errorf("[%s]incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

func (handler *Handler) handleGetStateByRange(startKey, endKey string, txid string) (*pbinternal.QueryResponse, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send GET_STATE_BY_RANGE message to validator chaincode support
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pbinternal.GetStateByRange{StartKey: startKey, EndKey: endKey})

	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_GET_STATE_BY_RANGE, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_STATE_BY_RANGE)

	var responseMsg pbinternal.ChaincodeMessage
	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.Errorf("[%s]error sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_STATE_BY_RANGE)
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully got range", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)

		rangeQueryResponse := &pbinternal.QueryResponse{}
		if err = proto.Unmarshal(responseMsg.Payload, rangeQueryResponse); err != nil {
			return nil, errors.Errorf("[%s]GetStateByRangeResponse unmarshall error", shorttxid(responseMsg.Txid))
		}

		return rangeQueryResponse, nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.Errorf("incorrect chaincode message %s received. Expecting %s or %s", responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

func (handler *Handler) handleQueryStateNext(id, txid string) (*pbinternal.QueryResponse, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send QUERY_STATE_NEXT message to validator chaincode support
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pbinternal.QueryStateNext{Id: id})

	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_QUERY_STATE_NEXT, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_QUERY_STATE_NEXT)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.Errorf("[%s]error sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_QUERY_STATE_NEXT)
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully got range", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)

		queryResponse := &pbinternal.QueryResponse{}
		if err = proto.Unmarshal(responseMsg.Payload, queryResponse); err != nil {
			return nil, errors.Errorf("[%s]unmarshal error", shorttxid(responseMsg.Txid))
		}

		return queryResponse, nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.Errorf("incorrect chaincode message %s received. Expecting %s or %s", responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

func (handler *Handler) handleQueryStateClose(id, txid string) (*pbinternal.QueryResponse, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send QUERY_STATE_CLOSE message to validator chaincode support
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pbinternal.QueryStateClose{Id: id})

	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_QUERY_STATE_CLOSE, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_QUERY_STATE_CLOSE)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.Errorf("[%s]error sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_QUERY_STATE_CLOSE)
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully got range", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)

		queryResponse := &pbinternal.QueryResponse{}
		if err = proto.Unmarshal(responseMsg.Payload, queryResponse); err != nil {
			return nil, errors.Errorf("[%s]unmarshal error", shorttxid(responseMsg.Txid))
		}

		return queryResponse, nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.Errorf("incorrect chaincode message %s received. Expecting %s or %s", responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

func (handler *Handler) handleGetQueryResult(query string, txid string) (*pbinternal.QueryResponse, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send GET_QUERY_RESULT message to validator chaincode support
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pbinternal.GetQueryResult{Query: query})

	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_GET_QUERY_RESULT, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_QUERY_RESULT)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.Errorf("[%s]error sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_QUERY_RESULT)
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully got range", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)

		executeQueryResponse := &pbinternal.QueryResponse{}
		if err = proto.Unmarshal(responseMsg.Payload, executeQueryResponse); err != nil {
			return nil, errors.Errorf("[%s]unmarshal error", shorttxid(responseMsg.Txid))
		}

		return executeQueryResponse, nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.Errorf("incorrect chaincode message %s received. Expecting %s or %s", responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

func (handler *Handler) handleGetHistoryForKey(key string, txid string) (*pbinternal.QueryResponse, error) {
	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return nil, err
	}

	defer handler.deleteChannel(txid)

	// Send GET_HISTORY_FOR_KEY message to validator chaincode support
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pbinternal.GetHistoryForKey{Key: key})

	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_GET_HISTORY_FOR_KEY, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_HISTORY_FOR_KEY)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return nil, errors.Errorf("[%s]error sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_GET_HISTORY_FOR_KEY)
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully got range", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)

		getHistoryForKeyResponse := &pbinternal.QueryResponse{}
		if err = proto.Unmarshal(responseMsg.Payload, getHistoryForKeyResponse); err != nil {
			return nil, errors.Errorf("[%s]unmarshal error", shorttxid(responseMsg.Txid))
		}

		return getHistoryForKeyResponse, nil
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return nil, errors.New(string(responseMsg.Payload[:]))
	}

	// Incorrect chaincode message received
	return nil, errors.Errorf("incorrect chaincode message %s received. Expecting %s or %s", responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)
}

func (handler *Handler) createResponse(status int32, payload []byte) pb.Response {
	return pb.Response{Status: status, Payload: payload}
}

// handleInvokeChaincode communicates with the validator to invoke another chaincode.
func (handler *Handler) handleInvokeChaincode(chaincodeName string, args [][]byte, txid string) pb.Response {
	//we constructed a valid object. No need to check for error
	payloadBytes, _ := proto.Marshal(&pb.ChaincodeSpec{ChaincodeId: &pb.ChaincodeID{Name: chaincodeName}, Input: &pb.ChaincodeInput{Args: args}})

	// Create the channel on which to communicate the response from validating peer
	var respChan chan pbinternal.ChaincodeMessage
	var err error
	if respChan, err = handler.createChannel(txid); err != nil {
		return handler.createResponse(ERROR, []byte(err.Error()))
	}

	defer handler.deleteChannel(txid)

	// Send INVOKE_CHAINCODE message to validator chaincode support
	msg := &pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_INVOKE_CHAINCODE, Payload: payloadBytes, Txid: txid}
	chaincodeLogger.Debugf("[%s]Sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_INVOKE_CHAINCODE)

	var responseMsg pbinternal.ChaincodeMessage

	if responseMsg, err = handler.sendReceive(msg, respChan); err != nil {
		return handler.createResponse(ERROR, []byte(fmt.Sprintf("[%s]error sending %s", shorttxid(msg.Txid), pbinternal.ChaincodeMessage_INVOKE_CHAINCODE)))
	}

	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_RESPONSE.String() {
		// Success response
		chaincodeLogger.Debugf("[%s]Received %s. Successfully invoked chaincode", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)
		respMsg := &pbinternal.ChaincodeMessage{}
		if err = proto.Unmarshal(responseMsg.Payload, respMsg); err != nil {
			return handler.createResponse(ERROR, []byte(err.Error()))
		}
		if respMsg.Type == pbinternal.ChaincodeMessage_COMPLETED {
			// Success response
			chaincodeLogger.Debugf("[%s]Received %s. Successfully invoked chaincode", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_RESPONSE)
			res := &pb.Response{}
			if err = proto.Unmarshal(respMsg.Payload, res); err != nil {
				return handler.createResponse(ERROR, []byte(err.Error()))
			}
			return *res
		}
		chaincodeLogger.Errorf("[%s]Received %s. Error from chaincode", shorttxid(responseMsg.Txid), respMsg.Type.String())
		return handler.createResponse(ERROR, responseMsg.Payload)
	}
	if responseMsg.Type.String() == pbinternal.ChaincodeMessage_ERROR.String() {
		// Error response
		chaincodeLogger.Errorf("[%s]Received %s.", shorttxid(responseMsg.Txid), pbinternal.ChaincodeMessage_ERROR)
		return handler.createResponse(ERROR, responseMsg.Payload)
	}

	// Incorrect chaincode message received
	return handler.createResponse(ERROR, []byte(fmt.Sprintf("[%s]Incorrect chaincode message %s received. Expecting %s or %s", shorttxid(responseMsg.Txid), responseMsg.Type, pbinternal.ChaincodeMessage_RESPONSE, pbinternal.ChaincodeMessage_ERROR)))
}

// handleMessage message handles loop for shim side of chaincode/validator stream.
func (handler *Handler) handleMessage(msg *pbinternal.ChaincodeMessage) error {
	if msg.Type == pbinternal.ChaincodeMessage_KEEPALIVE {
		// Received a keep alive message, we don't do anything with it for now
		// and it does not touch the state machine
		return nil
	}
	chaincodeLogger.Debugf("[%s]Handling ChaincodeMessage of type: %s(state:%s)", shorttxid(msg.Txid), msg.Type, handler.FSM.Current())
	if handler.FSM.Cannot(msg.Type.String()) {
		err := errors.Errorf("[%s]chaincode handler FSM cannot handle message (%s) with payload size (%d) while in state: %s", msg.Txid, msg.Type.String(), len(msg.Payload), handler.FSM.Current())
		handler.serialSend(&pbinternal.ChaincodeMessage{Type: pbinternal.ChaincodeMessage_ERROR, Payload: []byte(err.Error()), Txid: msg.Txid})
		return err
	}
	err := handler.FSM.Event(msg.Type.String(), msg)
	return filterError(err)
}

// filterError filters the errors to allow NoTransitionError and CanceledError to not propagate for cases where embedded Err == nil.
func filterError(errFromFSMEvent error) error {
	if errFromFSMEvent != nil {
		if noTransitionErr, ok := errFromFSMEvent.(*fsm.NoTransitionError); ok {
			if noTransitionErr.Err != nil {
				// Only allow NoTransitionError's, all others are considered true error.
				return errFromFSMEvent
			}
		}
		if canceledErr, ok := errFromFSMEvent.(*fsm.CanceledError); ok {
			if canceledErr.Err != nil {
				// Only allow NoTransitionError's, all others are considered true error.
				return canceledErr
				//t.Error("expected only 'NoTransitionError'")
			}
			chaincodeLogger.Debugf("Ignoring CanceledError: %s", canceledErr)
		}
	}
	return nil
}
