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
	configapi "github.com/securekey/fabric-snaps/configmanager/api"
	configservice "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/spf13/viper"
)

var logger = flogging.MustGetLogger("eventsnap/config")

const (
	// EventSnapAppName is the name/ID of the eventsnap system chaincode
	EventSnapAppName = "eventsnap"

	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"

	defaultEventHubRegTimeout        = 2 * time.Second
	defaultEventRelayTimeout         = 2 * time.Second
	defaultEventDispatcherBufferSize = 100
	defaultEventConsumerBufferSize   = 100
	defaultEventConsumerTimeout      = 10 * time.Millisecond
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

// New returns a new EventSnapConfig for the given channel
func New(channelID, peerConfigPathOverride string) (*EventSnapConfig, error) {
	var peerConfigPath string
	if peerConfigPathOverride == "" {
		peerConfigPath = defaultPeerConfigPath
	} else {
		peerConfigPath = peerConfigPathOverride
	}

	peerConfig, err := newPeerViper(peerConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading peer config")
	}

	peerID := peerConfig.GetString("peer.id")
	mspID := peerConfig.GetString("peer.localMspId")

	// Initialize from peer config
	eventSnapConfig := &EventSnapConfig{
		EventHubAddress:       peerConfig.GetString("peer.events.address"),
		EventServerBufferSize: uint(peerConfig.GetInt("peer.channelserver.buffersize")),
		EventServerTimeout:    peerConfig.GetDuration("peer.channelserver.timeout"),
		EventServerTimeWindow: peerConfig.GetDuration("peer.channelserver.timewindow"),
	}

	if channelID != "" {
		config, err := configservice.GetInstance().GetViper(channelID, configapi.ConfigKey{MspID: mspID, PeerID: peerID, AppName: EventSnapAppName}, configapi.YAML)
		if err != nil {
			return nil, errors.Wrap(err, "error getting event snap configuration")
		}
		if config == nil {
			// No config yet. The peer must have just joined the channel.  Use default values for now.
			// After the config has been uploaded to the ledger the new values will take effect.
			logger.Warningf("Using default configuration for event snap since the configuration does not yet exist in the ledger for channel [%s]\n", channelID)

			eventSnapConfig.EventHubRegTimeout = defaultEventHubRegTimeout
			eventSnapConfig.EventRelayTimeout = defaultEventRelayTimeout
			eventSnapConfig.EventDispatcherBufferSize = defaultEventDispatcherBufferSize
			eventSnapConfig.EventConsumerBufferSize = defaultEventConsumerBufferSize
			eventSnapConfig.EventConsumerTimeout = defaultEventConsumerTimeout
		} else {
			eventSnapConfig.EventHubRegTimeout = config.GetDuration("eventsnap.eventhub.regtimeout")
			eventSnapConfig.EventRelayTimeout = config.GetDuration("eventsnap.relay.timeout")
			eventSnapConfig.EventDispatcherBufferSize = uint(config.GetInt("eventsnap.dispatcher.buffersize"))
			eventSnapConfig.EventConsumerBufferSize = uint(config.GetInt("eventsnap.consumer.buffersize"))
			eventSnapConfig.EventConsumerTimeout = config.GetDuration("eventsnap.consumer.timeout")
		}
	}

	return eventSnapConfig, nil
}

func newPeerViper(configPath string) (*viper.Viper, error) {
	peerViper := viper.New()
	peerViper.AddConfigPath(configPath)
	peerViper.SetConfigName(peerConfigName)
	peerViper.SetEnvPrefix(envPrefix)
	peerViper.AutomaticEnv()
	peerViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := peerViper.ReadInConfig(); err != nil {
		return nil, err
	}
	return peerViper, nil
}
