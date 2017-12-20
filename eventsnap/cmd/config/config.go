/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"strings"
	"time"

	"google.golang.org/grpc/credentials"

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

	// TransportCredentials is the credentials used for connecting with peer event service
	TransportCredentials credentials.TransportCredentials

	// channelConfigLoaded indicates whether the channel-specific configuration was loaded
	ChannelConfigLoaded bool
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

		logger.Debugf("Getting configuration from ledger for msp [%s], peer [%s], app [%s]", mspID, peerID, EventSnapAppName)

		config, err := configservice.GetInstance().GetViper(channelID, configapi.ConfigKey{MspID: mspID, PeerID: peerID, AppName: EventSnapAppName}, configapi.YAML)
		if err != nil {
			return nil, errors.Wrap(err, "error getting event snap configuration")
		}
		if config != nil {

			logger.Debugf("Using configuration from ledger for event snap for channel [%s]\n", channelID)
			eventSnapConfig.ChannelConfigLoaded = true
			eventSnapConfig.EventHubRegTimeout = config.GetDuration("eventsnap.eventhub.regtimeout")
			eventSnapConfig.EventRelayTimeout = config.GetDuration("eventsnap.relay.timeout")
			eventSnapConfig.EventDispatcherBufferSize = uint(config.GetInt("eventsnap.dispatcher.buffersize"))
			eventSnapConfig.EventConsumerBufferSize = uint(config.GetInt("eventsnap.consumer.buffersize"))
			eventSnapConfig.EventConsumerTimeout = config.GetDuration("eventsnap.consumer.timeout")
			tlsCredentials, err := getTLSCredentials(peerConfig, config)
			if err != nil {
				return nil, err
			}

			logger.Debugf("TLS Credentials: %s", tlsCredentials)
			eventSnapConfig.TransportCredentials = tlsCredentials
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

func getTLSCredentials(peerConfig, config *viper.Viper) (credentials.TransportCredentials, error) {

	tlsCaCertPool := x509.NewCertPool()
	if config.GetBool("eventsnap.eventhub.tlsCerts.systemCertPool") == true {
		var err error
		if tlsCaCertPool, err = x509.SystemCertPool(); err != nil {
			return nil, err
		}
		logger.Debugf("Loaded system cert pool of size: %d", len(tlsCaCertPool.Subjects()))
	}

	logger.Debugf("tls rootcert: %s", peerConfig.GetString("peer.tls.rootcert.file"))

	if peerConfig.GetString("peer.tls.rootcert.file") != "" {

		rawData, err := ioutil.ReadFile(peerConfig.GetString("peer.tls.rootcert.file"))
		if err != nil {
			return nil, errors.Wrapf(err, "error reading peer tls root cert file")
		}

		block, _ := pem.Decode(rawData)
		if block == nil {
			return nil, errors.Wrapf(err, "pem data missing")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrapf(err, "parse certificate from block failed")
		}

		tlsCaCertPool.AddCert(cert)
	}

	logger.Debugf("server host override: %s", peerConfig.GetString("peer.tls.serverhostoverride"))

	var sn string
	if peerConfig.GetString("peer.tls.serverhostoverride") != "" {
		sn = peerConfig.GetString("peer.tls.serverhostoverride")
	}

	logger.Debugf("tls client embedded cert: %s", config.GetString("eventsnap.eventhub.tlsCerts.client.certpem"))
	logger.Debugf("tls client file cert: %s", config.GetString("eventsnap.eventhub.tlsCerts.client.certfile"))
	logger.Debugf("tls client key: %s", config.GetString("eventsnap.eventhub.tlsCerts.client.keyfile"))

	var certificates []tls.Certificate
	// certpem is by default.. if it exists, load it, if not, check for certfile and load the cert
	// if both are not found then assumption is tls is disabled
	if config.GetString("eventsnap.eventhub.tlsCerts.client.certpem") != "" {
		keyBytes, err := ioutil.ReadFile(config.GetString("eventsnap.eventhub.tlsCerts.client.keyfile"))
		if err != nil {
			return nil, errors.Errorf("Error reading key TLS client credentials: %v", err)
		}
		clientCerts, err := tls.X509KeyPair([]byte(config.GetString("eventsnap.eventhub.tlsCerts.client.certpem")), keyBytes)
		if err != nil {
			return nil, errors.Errorf("Error loading embedded cert/key pair as TLS client credentials: %v", err)
		}
		certificates = []tls.Certificate{clientCerts}
	} else if config.GetString("eventsnap.eventhub.tlsCerts.client.certfile") != "" {
		clientCerts, err := tls.LoadX509KeyPair(config.GetString("eventsnap.eventhub.tlsCerts.client.certfile"), config.GetString("eventsnap.eventhub.tlsCerts.client.keyfile"))
		if err != nil {
			return nil, errors.Errorf("Error loading cert/key pair as TLS client credentials: %v", err)
		}
		certificates = []tls.Certificate{clientCerts}
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: certificates,
		RootCAs:      tlsCaCertPool,
		ServerName:   sn,
	})

	return creds, nil
}
