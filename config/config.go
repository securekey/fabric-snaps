/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"sync"

	"github.com/fsnotify/fsnotify"
	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/snaps/examples/examplesnap"
	"github.com/securekey/fabric-snaps/snaps/interfaces"
	"github.com/spf13/viper"
)

const (
	configFileName     = "config"
	peerConfigFileName = "core"
	cmdRootPrefix      = "core"
	devConfigPath      = "$GOPATH/src/github.com/securekey/fabric-snaps/config/sampleconfig"
)

var peerConfig = viper.New()
var logger = logging.MustGetLogger("snap-config")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)
var mutex = &sync.Mutex{}

// SnapConfig defines the metadata needed to initialize the code
// when the fabric comes up. SnapConfigs are installed by adding an
// entry in config.yaml and creating the new SnapConfig implementation
type SnapConfig struct {
	// Enabled a convenient switch to enable/disable SnapConfig without
	// having to remove entry from Snaps array below
	Enabled bool

	//Unique name of the snap code, it should match the SnapConfig implementation class name
	Name string

	//String representation for InitArgs read by yaml
	InitArgsStr []string

	//InitArgs initialization arguments to startup the snap chaincode
	InitArgs [][]byte

	// SnapConfig is the actual SnapConfig object
	Snap interfaces.Snap

	// SnapUrl to locate remote Snaps
	SnapUrl string

	// to identify if the snap is remote or local
	isRemote bool
}

// SnapConfigArray represents the list of snaps configurations from YAML
type SnapConfigArray struct {
	SnapConfigs []SnapConfig
}

var Snaps = []*SnapConfig{
	{
		Enabled:  true,
		Name:     "example",
		InitArgs: [][]byte{[]byte("")},
		Snap:     &examplesnap.SnapImpl{},
		isRemote: false,
	},
}

// Init configuration and logging for SnapConfigs. By default, we look for
// configuration files at a path described by the environment variable
// "FABRIC_CFG_PATH". This is where the configuration is expected to be set in
// a production SnapConfigs image. For testing and development, a GOPATH, project
// relative path is used. Optionally, a path override parameter can be passed in
// @param {string} [OPTIONAL] configPathOverride
// @returns {error} error, if any
func Init(configPathOverride string) error {
	var configPath = os.Getenv("FABRIC_CFG_PATH")
	replacer := strings.NewReplacer(".", "_")

	if configPath != "" {
		viper.AddConfigPath(configPath)
		peerConfig.AddConfigPath(configPath)
	} else {
		if configPathOverride == "" {
			configPathOverride = devConfigPath
		}
		viper.AddConfigPath(configPathOverride)
		peerConfig.AddConfigPath(configPathOverride)
	}
	viper.SetConfigName(configFileName)
	viper.SetEnvPrefix(cmdRootPrefix)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(replacer)

	peerConfig.SetConfigName(peerConfigFileName)
	peerConfig.SetEnvPrefix(cmdRootPrefix)
	peerConfig.AutomaticEnv()
	peerConfig.SetEnvKeyReplacer(replacer)

	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error reading snap config file: %s", err)
	}

	err = peerConfig.ReadInConfig()
	if err != nil {
		return fmt.Errorf("Fatal error reading peer config file: %s", err)
	}

	err = initializeLogging()
	if err != nil {
		return fmt.Errorf("Error initializing logging: %s", err)
	}

	err = initializeSnapConfigs()
	if err != nil {
		return fmt.Errorf("Error initializing snaps: %s", err)
	}

	logger.Debug("Snaps are ready to be used.", len(Snaps), "snaps configs are added from the config.")

	//keep monitoring configs for any changes
	go func() {
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			logger.Info("Config file changed:", e.Name, " re initializing snaps..")
			// access Snaps from the routine should be locked
			mutex.Lock()
			defer mutex.Unlock()
			Snaps = Snaps[:1] // resetting snaps, increase the slice if hard coding new snaps in the array definition above
			err = initializeSnapConfigs()
			if err != nil {
				logger.Errorf("Error initializing snaps following yaml update: %s", err)
				return
			}
			logger.Debug("Snap count after initializing following yaml update:", len(Snaps))
		})
	}()
	
	return nil
}

func initializeLogging() error {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	level, err := logging.LogLevel(viper.GetString("snap.daemon.loglevel"))

	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetBackend(backendFormatter).SetLevel(level, "")

	logger.Debugf("SnapConfigs Logger initialized. Log level: %s", logging.GetLevel(""))

	return nil
}

func initializeSnapConfigs() error {
	snapConfig := &SnapConfigArray{}
	err := viper.UnmarshalKey("snaps", &snapConfig.SnapConfigs)

	if err != nil {
		return err
	}

	logger.Debug("Found", len(snapConfig.SnapConfigs), "snaps config(s) in yaml file.")

	// append snaps to snapsArray
	for _, snapConfigCopy := range snapConfig.SnapConfigs {
		var snapMetaData SnapConfig = resolveSnapInitAndImplementation(&snapConfigCopy)
		if len(snapMetaData.SnapUrl) > 0 {
			snapMetaData.isRemote = true
		}
		logger.Debug("Adding Snap config:", snapMetaData.Name, " Remote?", snapMetaData.isRemote)
		Snaps = append(Snaps, &snapMetaData)
	}

	return nil
}

func resolveSnapInitAndImplementation(sp *SnapConfig) SnapConfig {
	for _, initArgVal := range sp.InitArgsStr {
		logger.Debugf("Appending init arg: %s, concatenating as a byte array: %s\n", initArgVal, []byte(initArgVal))
		sp.InitArgs = append(sp.InitArgs, []byte(initArgVal))
	}
	logger.Debug(len(sp.InitArgs), "InitArgs for snap", sp.Name, "configured.")

	return *sp
}

// IsTLSEnabled is TLS enabled?
func IsTLSEnabled() bool {
	return peerConfig.GetBool("snap.server.tls.enabled")
}

// GetTLSRootCertPath returns absolute path to the TLS root certificate
func GetTLSRootCertPath() string {
	return GetConfigPath(peerConfig.GetString("snap.server.tls.rootcert.file"))
}

// GetTLSCertPath returns absolute path to the TLS certificate
func GetTLSCertPath() string {
	return GetConfigPath(peerConfig.GetString("snap.server.tls.cert.file"))
}

// GetTLSKeyPath returns absolute path to the TLS key
func GetTLSKeyPath() string {
	return GetConfigPath(peerConfig.GetString("snap.server.tls.key.file"))
}

// GetSnapServerPort returns snap server port
func GetSnapServerPort() string {
	return viper.GetString("snap.server.port")
}

//GetSnapConfig
func GetSnapConfig(snapName string) *SnapConfig {

	mutex.Lock()
	defer mutex.Unlock()
	//registeredSnaps := config.GetSnapArray()
	for _, registeredSnap := range Snaps {
		if registeredSnap.Name == snapName {
			logger.Debugf("Found registered snap %s", registeredSnap.Name)
			return registeredSnap
		}
	}

	return nil
}

// GetConfigPath returns the absolute value of the given path that is
// relative to the config file
// For example, if the config file is at /etc/hyperledger/config.yaml,
// calling GetConfigPath("tls/cert") will return /etc/hyperledger/tls/cert
func GetConfigPath(path string) string {
	basePath := filepath.Dir(viper.ConfigFileUsed())

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(basePath, path)
}
