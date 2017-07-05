/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"

	api "github.com/hyperledger/fabric-sdk-go/api"

	bccspFactory "github.com/hyperledger/fabric/bccsp/factory"
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var myViper = viper.New()
var log = logging.MustGetLogger("fabric_sdk_go")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{level:.4s} : %{color:reset} %{message}`,
)

const cmdRoot = "fabric_sdk"

type config struct {
	networkConfig       *api.NetworkConfig
	networkConfigCached bool
}

// InitConfig ...
// initConfig reads in config file
func InitConfig(configFile string) (api.Config, error) {
	return InitConfigWithCmdRoot(configFile, cmdRoot)
}

// InitConfigWithCmdRoot reads in a config file and allows the
// environment variable prefixed to be specified
func InitConfigWithCmdRoot(configFile string, cmdRootPrefix string) (api.Config, error) {
	myViper.SetEnvPrefix(cmdRootPrefix)
	myViper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	myViper.SetEnvKeyReplacer(replacer)

	if configFile != "" {
		// create new viper
		myViper.SetConfigFile(configFile)
		// If a config file is found, read it in.
		err := myViper.ReadInConfig()

		if err == nil {
			log.Infof("Using config file: %s", myViper.ConfigFileUsed())
		} else {
			return nil, fmt.Errorf("Fatal error config file: %v", err)
		}
	}
	log.Debug(myViper.GetString("client.fabricCA.serverURL"))
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)

	loggingLevelString := myViper.GetString("client.logging.level")
	logLevel := logging.INFO
	if loggingLevelString != "" {
		log.Infof("fabric_sdk_go Logging level: %v", loggingLevelString)
		var err error
		logLevel, err = logging.LogLevel(loggingLevelString)
		if err != nil {
			panic(err)
		}
	}
	logging.SetBackend(backendFormatter).SetLevel(logging.Level(logLevel), "fabric_sdk_go")

	return &config{}, nil

}

func (c *config) CAConfig(org string) (*api.CAConfig, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return nil, err
	}
	caConfig := config.Organizations[org].CA

	return &caConfig, nil
}

//GetCAServerCertFiles Read configuration option for the server certificate files
func (c *config) CAServerCertFiles(org string) ([]string, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return nil, err
	}

	certFiles := strings.Split(config.Organizations[org].CA.TLS.Certfiles, ",")

	certFileModPath := make([]string, len(certFiles))
	for i, v := range certFiles {
		certFileModPath[i] = strings.Replace(v, "$GOPATH", os.Getenv("GOPATH"), -1)
	}
	return certFileModPath, nil
}

//GetCAClientKeyFile Read configuration option for the fabric CA client key file
func (c *config) CAClientKeyFile(org string) (string, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return "", err
	}

	return strings.Replace(config.Organizations[org].CA.TLS.Client.Keyfile,
		"$GOPATH", os.Getenv("GOPATH"), -1), nil
}

//GetCAClientCertFile Read configuration option for the fabric CA client cert file
func (c *config) CAClientCertFile(org string) (string, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return "", err
	}

	return strings.Replace(config.Organizations[org].CA.TLS.Client.Certfile,
		"$GOPATH", os.Getenv("GOPATH"), -1), nil
}

// MspID returns the MSP ID for the requested organization
func (c *config) MspID(org string) (string, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return "", err
	}
	mspID := config.Organizations[org].MspID
	if mspID == "" {
		return "", fmt.Errorf("MSP ID is empty for org: %s", org)
	}

	return mspID, nil
}

// FabricClientViper returns the internal viper instance used by the
// SDK to read configuration options
func (c *config) FabricClientViper() *viper.Viper {
	return myViper
}

func (c *config) cacheNetworkConfiguration() error {
	err := myViper.UnmarshalKey("client.network", &c.networkConfig)
	if err == nil {
		c.networkConfigCached = true
		return nil
	}

	return err
}

// GetOrderersConfig returns a list of defined orderers
func (c *config) OrderersConfig() ([]api.OrdererConfig, error) {
	orderers := []api.OrdererConfig{}
	config, err := c.NetworkConfig()
	if err != nil {
		return nil, err
	}

	for _, orderer := range config.Orderers {
		orderer.TLS.Certificate = strings.Replace(orderer.TLS.Certificate, "$GOPATH",
			os.Getenv("GOPATH"), -1)
		orderers = append(orderers, orderer)
	}

	return orderers, nil
}

// RandomOrdererConfig returns a pseudo-random orderer from the network config
func (c *config) RandomOrdererConfig() (*api.OrdererConfig, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return nil, err
	}

	rs := rand.NewSource(time.Now().Unix())
	r := rand.New(rs)
	randomNumber := r.Intn(len(config.Orderers))

	var i int
	for _, value := range config.Orderers {
		value.TLS.Certificate = strings.Replace(value.TLS.Certificate, "$GOPATH",
			os.Getenv("GOPATH"), -1)
		if i == randomNumber {
			return &value, nil
		}
		i++
	}

	return nil, nil
}

// OrdererConfig returns the requested orderer
func (c *config) OrdererConfig(name string) (*api.OrdererConfig, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return nil, err
	}
	orderer := config.Orderers[name]

	orderer.TLS.Certificate = strings.Replace(orderer.TLS.Certificate, "$GOPATH",
		os.Getenv("GOPATH"), -1)

	return &orderer, nil
}

// PeersConfig Retrieves the fabric peers for the specified org from the
// config file provided
func (c *config) PeersConfig(org string) ([]api.PeerConfig, error) {
	config, err := c.NetworkConfig()
	if err != nil {
		return nil, err
	}

	peersConfig := config.Organizations[org].Peers
	peers := []api.PeerConfig{}

	for key, p := range peersConfig {
		if p.Host == "" {
			return nil, fmt.Errorf("host key not exist or empty for peer %s", key)
		}
		if p.Port == 0 {
			return nil, fmt.Errorf("port key not exist or empty for peer %s", key)
		}
		if c.IsTLSEnabled() && p.TLS.Certificate == "" {
			return nil, fmt.Errorf("tls.certificate not exist or empty for peer %s", key)
		}
		p.TLS.Certificate = strings.Replace(p.TLS.Certificate, "$GOPATH",
			os.Getenv("GOPATH"), -1)
		peers = append(peers, p)
	}
	return peers, nil
}

// NetworkConfig returns the network configuration defined in the config file
func (c *config) NetworkConfig() (*api.NetworkConfig, error) {
	if c.networkConfigCached {
		return c.networkConfig, nil
	}

	if err := c.cacheNetworkConfiguration(); err != nil {
		return nil, fmt.Errorf("Error reading network configuration: %s", err)
	}
	return c.networkConfig, nil
}

// IsTLSEnabled is TLS enabled?
func (c *config) IsTLSEnabled() bool {
	return myViper.GetBool("client.tls.enabled")
}

// TLSCACertPool ...
// TODO: Should be related to configuration.
func (c *config) TLSCACertPool(tlsCertificate string) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	if tlsCertificate != "" {
		rawData, err := ioutil.ReadFile(tlsCertificate)
		if err != nil {
			return nil, err
		}

		cert, err := loadCAKey(rawData)
		if err != nil {
			return nil, err
		}

		certPool.AddCert(cert)
	}

	return certPool, nil
}

// TLSCACertPoolFromRoots ...
func (c *config) TLSCACertPoolFromRoots(ordererRootCAs [][]byte) (*x509.CertPool, error) {
	certPool := x509.NewCertPool()

	for _, root := range ordererRootCAs {
		cert, err := loadCAKey(root)
		if err != nil {
			return nil, err
		}

		certPool.AddCert(cert)
	}

	return certPool, nil
}

// IsSecurityEnabled ...
func (c *config) IsSecurityEnabled() bool {
	return myViper.GetBool("client.security.enabled")
}

// TcertBatchSize ...
func (c *config) TcertBatchSize() int {
	return myViper.GetInt("client.tcert.batch.size")
}

// SecurityAlgorithm ...
func (c *config) SecurityAlgorithm() string {
	return myViper.GetString("client.security.hashAlgorithm")
}

// SecurityLevel ...
func (c *config) SecurityLevel() int {
	return myViper.GetInt("client.security.level")

}

// KeyStorePath returns the keystore path used by BCCSP
func (c *config) KeyStorePath() string {
	keystorePath := strings.Replace(myViper.GetString("client.keystore.path"),
		"$GOPATH", os.Getenv("GOPATH"), -1)
	return path.Join(keystorePath, "keystore")
}

// CAKeystorePath returns the same path as KeyStorePath() without the
// 'keystore' directory added. This is done because the fabric-ca-client
// adds this to the path
func (c *config) CAKeyStorePath() string {
	return strings.Replace(myViper.GetString("client.keystore.path"),
		"$GOPATH", os.Getenv("GOPATH"), -1)
}

// CryptoConfigPath ...
func (c *config) CryptoConfigPath() string {
	return strings.Replace(myViper.GetString("client.cryptoconfig.path"),
		"$GOPATH", os.Getenv("GOPATH"), -1)
}

// loadCAKey
func loadCAKey(rawData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(rawData)

	if block != nil {
		pub, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.New("Failed to parse certificate: " + err.Error())
		}

		return pub, nil
	}
	return nil, errors.New("No pem data found")
}

// CSPConfig ...
func (c *config) CSPConfig() *bccspFactory.FactoryOpts {
	return &bccspFactory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &bccspFactory.SwOpts{
			HashFamily: c.SecurityAlgorithm(),
			SecLevel:   c.SecurityLevel(),
			FileKeystore: &bccspFactory.FileKeystoreOpts{
				KeyStorePath: c.KeyStorePath(),
			},
			Ephemeral: false,
		},
	}
}
