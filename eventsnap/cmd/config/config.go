/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"strings"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("eventsnap/config")

const (
	configName            = "config"
	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
	defaultConfigPath     = "/opt/extsysccs/config/eventsnap"
)

// EventSnapConfig contains the configuration for the EventSnap
type EventSnapConfig struct {
	// EventHubAddress is the address of the event hub that the Event Relay connects to for events
	EventHubAddress string

	// EventHubRegTimeout is the timeout for registering for events with the Event Hub
	EventHubRegTimeout time.Duration

	// EventRelayTimeout is the timeout when relaying events to the registered event channel.
	// If < 0, if buffer full, unblocks immediately and does not send.
	// If 0, if buffer full, will block and guarantee the event will be sent out.
	// If > 0, if buffer full, blocks util timeout.
	EventRelayTimeout time.Duration

	// EventServerBufferSize is the size of the registered consumer's event channel.
	EventServerBufferSize uint
	// EventServerTimeout is the timeout when sending events to a registered consumer.
	// If < 0, if buffer full, unblocks immediately and does not send.
	// If 0, if buffer full, will block and guarantee the event will be sent out.
	// If > 0, if buffer full, blocks util timeout.
	EventServerTimeout time.Duration

	// EventServerTimeWindow is the acceptable difference between the peer's current
	// time and the client's time as specified in a registration event
	EventServerTimeWindow time.Duration

	// EventDispatcherBufferSize is the size of the event dispatcher channel buffer.
	EventDispatcherBufferSize uint

	// EventConsumerBufferSize is the size of the registered consumer's event channel.
	EventConsumerBufferSize uint

	// EventConsumerTimeout is the timeout when sending events to a registered consumer.
	// If < 0, if buffer full, unblocks immediately and does not send.
	// If 0, if buffer full, will block and guarantee the event will be sent out.
	// If > 0, if buffer full, blocks util timeout.
	EventConsumerTimeout time.Duration
}

// New returns a new EventSnapConfig
// TODO: Integrate with the Configuration Service
func New(configPathOverride string) (*EventSnapConfig, error) {
	var configPath string
	var peerConfigPath string
	if configPathOverride == "" {
		configPath = defaultConfigPath
		peerConfigPath = defaultPeerConfigPath
	} else {
		configPath = configPathOverride
		peerConfigPath = configPathOverride
	}

	logger.Infof("Initializing config - Path: %s, Peer Path: %s\n", configPath, peerConfigPath)

	peerViper, err := newViper(peerConfigPath, peerConfigName, envPrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading peer config")
	}

	eventSnapViper, err := newViper(configPath, configName, envPrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading event snap config")
	}

	return &EventSnapConfig{
		EventHubAddress: peerViper.GetString("peer.events.address"),

		EventHubRegTimeout:        eventSnapViper.GetDuration("eventsnap.eventhub.regtimeout"),
		EventRelayTimeout:         eventSnapViper.GetDuration("eventsnap.relay.timeout"),
		EventDispatcherBufferSize: uint(eventSnapViper.GetInt("eventsnap.dispatcher.buffersize")),
		EventConsumerBufferSize:   uint(eventSnapViper.GetInt("eventsnap.consumer.buffersize")),
		EventConsumerTimeout:      eventSnapViper.GetDuration("eventsnap.consumer.timeout"),
		EventServerBufferSize:     uint(eventSnapViper.GetInt("eventsnap.server.buffersize")),
		EventServerTimeout:        eventSnapViper.GetDuration("eventsnap.server.timeout"),
		EventServerTimeWindow:     eventSnapViper.GetDuration("eventsnap.server.timewindow"),
	}, nil
}

func newViper(configPath, configName, envPrefix string) (*viper.Viper, error) {
	peerViper := viper.New()
	peerViper.AddConfigPath(configPath)
	peerViper.SetConfigName(configName)
	peerViper.SetEnvPrefix(envPrefix)
	peerViper.AutomaticEnv()
	peerViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := peerViper.ReadInConfig(); err != nil {
		return nil, err
	}
	return peerViper, nil
}
