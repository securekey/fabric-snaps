/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	logging "github.com/hyperledger/fabric-sdk-go/pkg/common/logging"
	configapi "github.com/securekey/fabric-snaps/configmanager/api"
	configservice "github.com/securekey/fabric-snaps/configmanager/pkg/service"
	"github.com/securekey/fabric-snaps/util/errors"
	"github.com/spf13/viper"
)

var logger = logging.NewLogger("eventsnap")

const (
	// EventSnapAppName is the name/ID of the eventsnap system chaincode
	EventSnapAppName = "eventsnap"

	peerConfigName        = "core"
	envPrefix             = "core"
	defaultPeerConfigPath = "/etc/hyperledger/fabric"
)

// EventSnapConfig contains the configuration for the EventSnap
type EventSnapConfig struct {
	MSPID string

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

	// TLSConfig contains the TLS certs and CA cert pool
	TLSConfig *tls.Config

	// channelConfigLoaded indicates whether the channel-specific configuration was loaded
	ChannelConfigLoaded bool

	Bytes []byte

	CryptoProvider string

	MSPConfigPath string
}

// New returns a new EventSnapConfig for the given channel
func New(channelID, peerConfigPath string) (*EventSnapConfig, error) {
	if channelID == "" {
		return nil, errors.New(errors.GeneralError, "channel ID is required")
	}

	peerConfig, err := newPeerViper(peerConfigPath)
	if err != nil {
		return nil, errors.Wrapf(errors.GeneralError, err, "error reading peer config")
	}

	peerID := peerConfig.GetString("peer.id")
	mspID := peerConfig.GetString("peer.localMspId")

	cryptoProvider := peerConfig.GetString("peer.BCCSP.Default")
	if cryptoProvider == "" {
		return nil, errors.New(errors.GeneralError, "BCCSP Default provider not found")
	}

	mspConfigPath := substGoPath(peerConfig.GetString("peer.mspConfigPath"))

	// Initialize from peer config
	eventSnapConfig := &EventSnapConfig{
		MSPID:          mspID,
		MSPConfigPath:  mspConfigPath,
		CryptoProvider: cryptoProvider,
		URL:            peerConfig.GetString("peer.listenAddress"),
	}

	logger.Debugf("Getting configuration from ledger for msp [%s], peer [%s], app [%s]", mspID, peerID, EventSnapAppName)

	configKey := configapi.ConfigKey{MspID: mspID, PeerID: peerID, AppName: EventSnapAppName}
	config, err := configservice.GetInstance().GetViper(channelID, configKey, configapi.YAML)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "error getting event snap configuration Viper")
	}

	bytes, err := configservice.GetInstance().Get(channelID, configKey)
	if err != nil {
		return nil, errors.Wrap(errors.GeneralError, err, "error getting event snap configuration bytes")
	}

	eventSnapConfig.Bytes = bytes
	eventSnapConfig.ChannelConfigLoaded = true
	eventSnapConfig.ResponseTimeout = config.GetDuration("eventsnap.responsetimeout")
	eventSnapConfig.EventDispatcherBufferSize = uint(config.GetInt("eventsnap.dispatcher.buffersize"))
	eventSnapConfig.EventConsumerBufferSize = uint(config.GetInt("eventsnap.consumer.buffersize"))
	eventSnapConfig.EventConsumerTimeout = config.GetDuration("eventsnap.consumer.timeout")

	tlsConfig, err := getTLSConfig(peerConfig, config)
	if err != nil {
		return nil, err
	}

	logger.Debugf("TLS Config: %s", tlsConfig)
	eventSnapConfig.TLSConfig = tlsConfig

	return eventSnapConfig, nil
}

func newPeerViper(peerConfigPath string) (*viper.Viper, error) {
	if peerConfigPath == "" {
		peerConfigPath = defaultPeerConfigPath
	}

	peerViper := viper.New()
	peerViper.AddConfigPath(peerConfigPath)
	peerViper.SetConfigName(peerConfigName)
	peerViper.SetEnvPrefix(envPrefix)
	peerViper.AutomaticEnv()
	peerViper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := peerViper.ReadInConfig(); err != nil {
		return nil, errors.WithMessage(errors.GeneralError, err, "snap_config_init_error")
	}
	return peerViper, nil
}

func getTLSConfig(peerConfig, config *viper.Viper) (*tls.Config, error) {

	tlsCaCertPool := x509.NewCertPool()
	if config.GetBool("eventsnap.tlsCerts.systemCertPool") == true {
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
			return nil, errors.Wrapf(errors.GeneralError, err, "error reading peer tls root cert file")
		}

		block, _ := pem.Decode(rawData)
		if block == nil {
			return nil, errors.Wrapf(errors.GeneralError, err, "pem data missing")
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrapf(errors.GeneralError, err, "parse certificate from block failed")
		}

		tlsCaCertPool.AddCert(cert)
	}

	logger.Debugf("server host override: %s", peerConfig.GetString("peer.tls.serverhostoverride"))

	var sn string
	if peerConfig.GetString("peer.tls.serverhostoverride") != "" {
		sn = peerConfig.GetString("peer.tls.serverhostoverride")
	}

	logger.Debugf("tls client embedded cert: %s", config.GetString("eventsnap.tlsCerts.client.certpem"))
	logger.Debugf("tls client file cert: %s", config.GetString("eventsnap.tlsCerts.client.certfile"))
	logger.Debugf("tls client key: %s", config.GetString("eventsnap.tlsCerts.client.keyfile"))

	var certificates []tls.Certificate
	// certpem is by default.. if it exists, load it, if not, check for certfile and load the cert
	// if both are not found then assumption is the client is not providing any cert to the server
	if config.GetString("eventsnap.tlsCerts.client.certpem") != "" {
		keyBytes, err := ioutil.ReadFile(config.GetString("eventsnap.tlsCerts.client.keyfile"))
		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "Error reading key TLS client credentials")
		}
		clientCerts, err := tls.X509KeyPair([]byte(config.GetString("eventsnap.tlsCerts.client.certpem")), keyBytes)
		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "Error loading embedded cert/key pair as TLS client credentials")
		}
		certificates = []tls.Certificate{clientCerts}
	} else if config.GetString("eventsnap.tlsCerts.client.certfile") != "" {
		clientCerts, err := tls.LoadX509KeyPair(config.GetString("eventsnap.tlsCerts.client.certfile"), config.GetString("eventsnap.tlsCerts.client.keyfile"))
		if err != nil {
			return nil, errors.Wrap(errors.GeneralError, err, "Error loading cert/key pair as TLS client credentials")
		}
		certificates = []tls.Certificate{clientCerts}
	}

	creds := &tls.Config{
		Certificates: certificates,
		RootCAs:      tlsCaCertPool,
		ServerName:   sn,
	}

	return creds, nil
}

// substGoPath replaces instances of '$GOPATH' with the GOPATH. If the system
// has multiple GOPATHs then the first is used.
func substGoPath(s string) string {
	gpDefault := build.Default.GOPATH
	gps := filepath.SplitList(gpDefault)

	return strings.Replace(s, "$GOPATH", gps[0], -1)
}
