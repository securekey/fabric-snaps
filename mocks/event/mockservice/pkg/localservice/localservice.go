/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package localservice

import (
	"sync"

	"strings"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
)

var logger = logging.NewLogger("eventservice/localservice")

var channelServices map[string]fab.EventService
var initonce sync.Once
var mutex sync.RWMutex

// Register sets the local event service instance on the peer for the given channel.
func Register(channelID string, service fab.EventService) error {
	initonce.Do(func() {
		channelServices = make(map[string]fab.EventService)
	})

	if service == nil {
		return errors.Errorf("invalid event service being registered for channel [%s]", channelID)
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := channelServices[strings.ToLower(channelID)]; ok {
		logger.Warnf("Event service already registered for channel [%s]\n", channelID)
		return errors.Errorf("event service already registered for channel [%s]", channelID)
	}

	channelServices[strings.ToLower(channelID)] = service
	return nil
}

// Get returns the local event service for the given channel.
func Get(channelID string) fab.EventService {
	mutex.RLock()
	defer mutex.RUnlock()

	if len(channelServices) == 0 {
		logger.Debug("Event service list is empty")
	}

	service, ok := channelServices[strings.ToLower(channelID)]
	if !ok {
		return nil
	}
	return service
}
