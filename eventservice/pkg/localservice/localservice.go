/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localservice

import (
	"sync"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/logging"
	"github.com/pkg/errors"
	eventapi "github.com/securekey/fabric-snaps/eventservice/api"
)

var logger = logging.NewLogger("eventservice/localservice")

var channelServices map[string]eventapi.EventService
var initonce sync.Once
var mutex sync.RWMutex

// Register sets the local event service instance on the peer for the given channel.
func Register(channelID string, service eventapi.EventService) error {
	initonce.Do(func() {
		channelServices = make(map[string]eventapi.EventService)
	})

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := channelServices[channelID]; ok {
		logger.Warnf("Event service already registered for channel [%s]\n", channelID)
		return errors.Errorf("event service already registered for channel [%s]", channelID)
	}

	channelServices[channelID] = service
	return nil
}

// Get returns the local event service for the given channel.
func Get(channelID string) eventapi.EventService {
	mutex.RLock()
	defer mutex.RUnlock()

	service, ok := channelServices[channelID]
	if !ok {
		return nil
	}
	return service
}
