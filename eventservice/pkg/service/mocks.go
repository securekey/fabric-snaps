/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import "github.com/securekey/fabric-snaps/mocks/event/mockproducer"

// NewServiceWithMockProducer returns a new EventService using a mock event producer
func NewServiceWithMockProducer(channelID string, eventTypes []EventType, opts *Opts) (*EventService, *mockproducer.MockProducer, error) {
	service := NewService(opts, eventTypes)
	eventProducer := mockproducer.New()
	service.Start(eventProducer)
	return service, eventProducer, nil
}
