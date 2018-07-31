/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	configapi "github.com/securekey/fabric-snaps/configmanager/api"
	configservice "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/util/configcache"
	"github.com/securekey/fabric-snaps/util/errors"
)

var logger = logging.NewLogger("eventsnap")

const (
	// EventSnapAppName is the name/ID of the eventsnap system chaincode
	EventSnapAppName      = "eventsnap"
	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
	defaultLogLevel       = "info"
)

var peerConfigCache = configcache.New(peerConfigName, envPrefix, defaultPeerConfigPath)

// EventSnapConfig contains the configuration for the EventSnap
type EventSnapConfig struct {
	// URL is the URL of the peer
	URL string

	// ResponseTimeout is the timeout for responses from the event service
	ResponseTimeout time.Duration

	// EventDispatcherBufferSize is the size of the event dispatcher channel buffer.
	EventDispatcherBufferSize uint

	// EventConsumerBufferSize is the size of the registered consumer's event channel.
	EventConsumerBufferSize uint

	// EventConsumerTimeout is the timeout when sending events to a registered consumer.
	// If < 0, if buffer full, unblocks immediately and does not send.
	// If 0, if buffer full, will block and guarantee the event will be sent out.
	// If > 0, if buffer full, blocks util timeout.
	EventConsumerTimeout time.Duration

	Bytes []byte
}

// New returns a new EventSnapConfig for the given channel
func New(channelID, peerConfigPath string) (*EventSnapConfig, error) {
	if channelID == "" {
		return nil, errors.New(errors.InitializeConfigError, "channel ID is required")
	}

	peerConfig, err := peerConfigCache.Get(peerConfigPath)
	if err != nil {
		return nil, errors.Wrapf(errors.PeerConfigError, err, "error reading peer config")
	}

	peerID := peerConfig.GetString("peer.id")
	mspID := peerConfig.GetString("peer.localMspId")

	// Initialize from peer config
	eventSnapConfig := &EventSnapConfig{
		URL: peerConfig.GetString("peer.listenAddress"),
	}

	logger.Debugf("Getting configuration from ledger for msp [%s], peer [%s], app [%s]", mspID, peerID, EventSnapAppName)

	configKey := configapi.ConfigKey{MspID: mspID, PeerID: peerID, AppName: EventSnapAppName}
	config, dirty, err := configservice.GetInstance().GetViper(channelID, configKey, configapi.YAML)
	if err != nil {
		return nil, errors.Wrap(errors.InitializeConfigError, err, "error getting event snap configuration Viper")
	}
	if config == nil {
		return nil, errors.New(errors.MissingConfigDataError, "config data is empty")
	}

	bytes, _, err := configservice.GetInstance().Get(channelID, configKey)
	if err != nil {
		return nil, errors.Wrap(errors.GetConfigError, err, "error getting event snap configuration bytes")
	}

	eventSnapConfig.Bytes = bytes
	eventSnapConfig.ResponseTimeout = config.GetDuration("eventsnap.responsetimeout")
	eventSnapConfig.EventDispatcherBufferSize = uint(config.GetInt("eventsnap.dispatcher.buffersize"))
	eventSnapConfig.EventConsumerBufferSize = uint(config.GetInt("eventsnap.consumer.buffersize"))
	eventSnapConfig.EventConsumerTimeout = config.GetDuration("eventsnap.consumer.timeout")

	if dirty {
		logLevel := config.GetString("eventsnap.loglevel")
		if logLevel == "" {
			logLevel = defaultLogLevel
		}
		level, err := logging.LogLevel(logLevel)
		if err != nil {
			return nil, errors.WithMessage(errors.InitializeLoggingError, err, "Error initializing log level")
		}
		logging.SetLevel(EventSnapAppName, level)
		logger.Debugf("Eventsnap logging initialized. Log level: %s", logLevel)
	}

	return eventSnapConfig, nil
}
