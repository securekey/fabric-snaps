/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package bddtests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel/invoke"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	fabApi "github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
)

var lastTxnID string

// EventSnapSteps ...
type EventSnapSteps struct {
	BDDContext *BDDContext
}

// NewEventSnapSteps ...
func NewEventSnapSteps(context *BDDContext) *EventSnapSteps {
	return &EventSnapSteps{BDDContext: context}
}

func newRegisterTxFilterHandler(channelID string, bddContext *BDDContext, chaincodeID string, next ...invoke.Handler) *registerTxFilterHandler {
	return &registerTxFilterHandler{channelID: channelID, bddContext: bddContext, chaincodeID: chaincodeID, next: getNext(next)}
}

type registerTxFilterHandler struct {
	next        invoke.Handler
	channelID   string
	bddContext  *BDDContext
	chaincodeID string
}

func (f *registerTxFilterHandler) Handle(requestContext *invoke.RequestContext, clientContext *invoke.ClientContext) {

	logger.Infof("Registering Tx Status event for Tx ID %s\n", string(requestContext.Response.TransactionID))

	err := queryEventConsumer(f.bddContext, "registertx", f.channelID, string(requestContext.Response.TransactionID))
	if err != nil {
		requestContext.Error = errors.Wrapf(err, "error querying chaincode %s", f.chaincodeID)
		return
	}

	logger.Infof("Successfully registered Tx Status event for Tx ID %s\n", string(requestContext.Response.TransactionID))

	//Delegate to next step if any
	if f.next != nil {
		f.next.Handle(requestContext, clientContext)
	}
}

func getNext(next []invoke.Handler) invoke.Handler {
	if len(next) > 0 {
		return next[0]
	}
	return nil
}

func (t *EventSnapSteps) invokeAndRegisterTxEvent(ccID, channelID string, strArgs string) error {
	args := strings.Split(strArgs, ",")

	chClient, err := t.BDDContext.OrgChannelClient(t.BDDContext.Orgs()[0], USER, channelID)
	if err != nil {
		return fmt.Errorf("NewChannelClient returned error: %v", err)
	}

	customExecuteHandler :=
		invoke.NewProposalProcessorHandler(
			invoke.NewEndorsementHandler(
				invoke.NewEndorsementValidationHandler(
					newRegisterTxFilterHandler(channelID, t.BDDContext, ccID,
						invoke.NewSignatureValidationHandler(
							invoke.NewCommitHandler(),
						),
					),
				),
			),
		)

	resp, err := chClient.InvokeHandler(customExecuteHandler, channel.Request{ChaincodeID: ccID, Fcn: args[0],
		Args: GetByteArgs(args[1:])}, channel.WithTimeout(fabApi.Execute, 10*time.Second))

	if err != nil {
		return errors.Wrapf(err, "error invoking chaincode %s", ccID)
	}

	lastTxnID = string(resp.TransactionID)

	return nil
}

func queryEventConsumer(ctx *BDDContext, fcn string, channelID string, args ...string) error {

	chClient, err := ctx.OrgChannelClient(ctx.Orgs()[0], USER, channelID)
	if err != nil {
		return fmt.Errorf("NewChannelClient returned error: %v", err)
	}

	var bargs [][]byte
	bargs = append(bargs, []byte(channelID))
	bargs = append(bargs, GetByteArgs(args)...)

	response, err := chClient.Query(
		channel.Request{
			ChaincodeID: "eventconsumersnap",
			Fcn:         fcn,
			Args:        bargs,
		}, channel.WithTimeout(fabApi.Execute, 10*time.Second))
	if err != nil {
		return errors.Wrap(err, "error querying eventconumersnap")
	}

	queryValue = string(response.Payload)

	return nil
}

func (t *EventSnapSteps) registerBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "registerblock", channelID)
}

func (t *EventSnapSteps) unregisterBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "unregisterblock", channelID)
}

func (t *EventSnapSteps) getBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "getblockevents", channelID)
}

func (t *EventSnapSteps) deleteBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "deleteblockevents", channelID)
}

func (t *EventSnapSteps) registerFilteredBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "registerfilteredblock", channelID)
}

func (t *EventSnapSteps) unregisterFilteredBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "unregisterfilteredblock", channelID)
}

func (t *EventSnapSteps) getFilteredBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "getfilteredblockevents", channelID)
}

func (t *EventSnapSteps) deleteFilteredBlockEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "deletefilteredblockevents", channelID)
}

func (t *EventSnapSteps) registerCCEvents(channelID, ccID, eventFilter string) error {
	return queryEventConsumer(t.BDDContext, "registercc", channelID, ccID, eventFilter)
}

func (t *EventSnapSteps) unregisterCCEvents(channelID, ccID, eventFilter string) error {
	return queryEventConsumer(t.BDDContext, "unregistercc", channelID, ccID, eventFilter)
}

func (t *EventSnapSteps) getCCEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "getccevents", channelID)
}

func (t *EventSnapSteps) deleteCCEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "deleteccevents", channelID)
}

func (t *EventSnapSteps) getTxEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "gettxevents", channelID)
}

func (t *EventSnapSteps) deleteTxEvents(channelID string) error {
	return queryEventConsumer(t.BDDContext, "deletetxevents", channelID)
}

func (t *EventSnapSteps) containsBlockEvents(num int) error {
	if queryValue == "" && num == 0 {
		return nil
	}
	events, err := getBlockEvents(queryValue)
	if err != nil {
		return err
	}
	numReceived := len(events)
	if numReceived != num {
		return errors.Errorf("expecting %d block events but got %d", num, numReceived)
	}
	return nil
}

func (t *EventSnapSteps) containsFilteredBlockEvents(num int) error {
	if queryValue == "" && num == 0 {
		return nil
	}
	events, err := getFilteredBlockEvents(queryValue)
	if err != nil {
		return err
	}
	numReceived := len(events)
	if numReceived != num {
		return errors.Errorf("expecting %d filtered block events but got %d", num, numReceived)
	}
	return nil
}

func (t *EventSnapSteps) containsCCEvents(num int, ccID, eventFilter string) error {
	if queryValue == "" && num == 0 {
		return nil
	}
	events, err := getCCEvents(queryValue)
	if err != nil {
		return err
	}

	numReceived := len(events)
	if numReceived != num {
		return errors.Errorf("expecting %d chaincode events but got %d", num, numReceived)
	}

	regExp, err := regexp.Compile(eventFilter)
	if err != nil {
		return errors.Wrapf(err, "invalid event filter [%s] for chaincode [%s]", eventFilter, ccID)
	}

	for _, event := range events {
		if event.ChaincodeID != ccID {
			return errors.Errorf("expecting chaincode event for chaincode %s but got %s", ccID, event.ChaincodeID)
		}
		if !regExp.MatchString(event.EventName) {
			return errors.Errorf("expecting a chaincode event that matches event filter %s but got event %s", eventFilter, event.EventName)
		}
	}
	return nil
}

func (t *EventSnapSteps) containsTxEvent() error {
	events, err := getTxEvents(queryValue)
	if err != nil {
		return err
	}
	for _, event := range events {
		if lastTxnID == event.TxID {
			return nil
		}
	}
	return errors.Errorf("could not find a Tx Status event that matches the last Tx [%s]", lastTxnID)
}

func getBlockEvents(jsonstr string) ([]*fab.BlockEvent, error) {
	var events []*fab.BlockEvent
	if err := json.Unmarshal([]byte(jsonstr), &events); err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.Block == nil {
			return nil, errors.New("invalid block event")
		}
	}
	return events, nil
}

func getFilteredBlockEvents(jsonstr string) ([]*fab.FilteredBlockEvent, error) {
	var events []*fab.FilteredBlockEvent
	if err := json.Unmarshal([]byte(jsonstr), &events); err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.FilteredBlock == nil {
			return nil, errors.New("invalid filtered block event")
		}
	}
	return events, nil
}

func getCCEvents(jsonstr string) ([]*fab.CCEvent, error) {
	var events []*fab.CCEvent
	if err := json.Unmarshal([]byte(jsonstr), &events); err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.ChaincodeID == "" {
			return nil, errors.New("invalid chaincode event")
		}
	}
	return events, nil
}

func getTxEvents(jsonstr string) ([]*fab.TxStatusEvent, error) {
	var events []*fab.TxStatusEvent
	if err := json.Unmarshal([]byte(jsonstr), &events); err != nil {
		return nil, err
	}
	for _, event := range events {
		if event.TxID == "" {
			return nil, errors.New("invalid Tx status event")
		}
	}
	return events, nil
}

func (t *EventSnapSteps) registerSteps(s *godog.Suite) {
	s.BeforeScenario(t.BDDContext.BeforeScenario)
	s.AfterScenario(t.BDDContext.AfterScenario)
	s.Step(`^client C1 queries for block events on channel "([^"]*)"$`, t.getBlockEvents)
	s.Step(`^client C1 queries for filtered block events on channel "([^"]*)"$`, t.getFilteredBlockEvents)
	s.Step(`^client C1 queries for chaincode events on channel "([^"]*)"$`, t.getCCEvents)
	s.Step(`^client C1 queries for Tx status events on channel "([^"]*)"$`, t.getTxEvents)
	s.Step(`^client C1 deletes all block events on channel "([^"]*)"$`, t.deleteBlockEvents)
	s.Step(`^client C1 deletes all filtered block events on channel "([^"]*)"$`, t.deleteFilteredBlockEvents)
	s.Step(`^client C1 deletes all chaincode events on channel "([^"]*)"$`, t.deleteCCEvents)
	s.Step(`^client C1 deletes all Tx status events on channel "([^"]*)"$`, t.deleteTxEvents)
	s.Step(`^client C1 receives a response containing (\d+) block events$`, t.containsBlockEvents)
	s.Step(`^client C1 receives a response containing (\d+) filtered block events$`, t.containsFilteredBlockEvents)
	s.Step(`^client C1 receives a response containing (\d+) chaincode events for chaincode "([^"]*)" and event filter "([^"]*)"$`, t.containsCCEvents)
	s.Step(`^client C1 receives a response containing a Tx Status event for the last transaction ID$`, t.containsTxEvent)
	s.Step(`^client C1 registers for block events on channel "([^"]*)"$`, t.registerBlockEvents)
	s.Step(`^client C1 unregisters for block events on channel "([^"]*)"$`, t.unregisterBlockEvents)
	s.Step(`^client C1 registers for filtered block events on channel "([^"]*)"$`, t.registerFilteredBlockEvents)
	s.Step(`^client C1 unregisters for filtered block events on channel "([^"]*)"$`, t.unregisterFilteredBlockEvents)
	s.Step(`^client C1 registers for chaincode events on channel "([^"]*)" for chaincode "([^"]*)" and event filter "([^"]*)"$`, t.registerCCEvents)
	s.Step(`^client C1 unregisters for chaincode events on channel "([^"]*)" for chaincode "([^"]*)" and event filter "([^"]*)"$`, t.unregisterCCEvents)
	s.Step(`^client C1 invokes chaincode "([^"]*)" on channel "([^"]*)" with args "([^"]*)" and registers for a Tx event$`, t.invokeAndRegisterTxEvent)
}
