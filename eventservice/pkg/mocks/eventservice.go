/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/options"
	eventsservice "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/dispatcher"
	servicemocks "github.com/hyperledger/fabric-sdk-go/pkg/fab/events/service/mocks"
)

type producerOpts struct {
	ledger *servicemocks.MockLedger
}

type producerOpt func(opts *producerOpts)

//WithFilteredBlockLedger ...
func WithFilteredBlockLedger() producerOpt {
	return func(opts *producerOpts) {
		opts.ledger = servicemocks.NewMockLedger(servicemocks.FilteredBlockEventFactory)
	}
}

//NewServiceWithMockProducer ...
func NewServiceWithMockProducer(opts []options.Opt, pOpts ...producerOpt) (*eventsservice.Service, *servicemocks.MockProducer, error) {
	service := eventsservice.New(dispatcher.New(opts...), opts...)
	if err := service.Start(); err != nil {
		return nil, nil, err
	}

	eventch, err := service.Dispatcher().EventCh()
	if err != nil {
		return nil, nil, err
	}

	popts := producerOpts{}
	for _, opt := range pOpts {
		opt(&popts)
	}

	ledger := popts.ledger
	if popts.ledger == nil {
		ledger = servicemocks.NewMockLedger(servicemocks.BlockEventFactory)
	}

	eventProducer := servicemocks.NewMockProducer(ledger)
	producerch := eventProducer.Register()

	go func() {
		for {
			event, ok := <-producerch
			if !ok {
				return
			}
			eventch <- event
		}
	}()

	return service, eventProducer, nil
}
