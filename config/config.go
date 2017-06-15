/* Copyright SecureKey Technologies Inc.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.*/

package config

import (
	"fmt"
	"os"

	logging "github.com/op/go-logging"
	"github.com/securekey/fabric-snaps/snaps/interfaces"
	"github.com/securekey/fabric-snaps/snaps/example"
	"github.com/spf13/viper"
	"strings"
	"github.com/fsnotify/fsnotify"
)

const (
	configFileName = "config"
	peerConfigFileName = "core"
	cmdRootPrefix  = "core"
	devConfigPath  = "$GOPATH/src/github.com/securekey/fabric-snaps/config/sampleconfig"
)

var peerConfig = viper.New()
var logger = logging.MustGetLogger("snap-config")
var logFormat = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} [%{module}] %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)
// SnapConfig defines the metadata needed to initialize the code
// when the fabric comes up. SnapConfigs are installed by adding an
// entry in config.yaml and creating the new SnapConfig implementation
type SnapConfig struct {
	// Enabled a convenient switch to enable/disable SnapConfig chaincode without
	// having to remove entry from importsysccs.go
	Enabled bool

	//Unique name of the snap code, it should match the SnapConfig implementation class name
	Name string

	//Path to the snap code; currently not used
	Path string

	//String representation for InitArgs read by yaml
	InitArgsStr []string

	//InitArgs initialization arguments to startup the snap chaincode
	InitArgs [][]byte

	// SnapConfig is the actual SnapConfig object
	Snap interfaces.Snap
}

// SnapConfigArray represents the list of snaps configurations from YAML
type SnapConfigArray struct {
	SnapConfigs []SnapConfig
}

var Snaps = []*SnapConfig{
	{
		Enabled:  true,
		Name:     "examplesnamp",
		Path:     "github.com/securekey/fabric-SnapConfigs/SnapConfigs/example",
		InitArgs: [][]byte{[]byte("")},
		Snap:     &example.SampleSnap{},
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

	logger.Debug("Snaps are ready to be used.",len(Snaps),"snaps configs are added from the config.")

	//keep monitoring configs for any changes
	//yamlUpdateEvent := make(chan fsnotify.Event)
	go func() {
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			logger.Info("Config file changed:", e.Name, " re initializing snaps..")
			Snaps = Snaps[:1] // resetting snaps
			err = initializeSnapConfigs()
			if err != nil {
				logger.Errorf("Error initializing snaps: %s", err)
				return
			}
			logger.Debug("Sanp count after initializing:", len(Snaps))
		})
	}()

	return nil
}

func initializeLogging() error {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, logFormat)
	level, err := logging.LogLevel(viper.GetString("snapDaemon.loglevel"))

	if err != nil {
		return fmt.Errorf("Error initializing log level: %s", err)
	}

	logging.SetBackend(backendFormatter).SetLevel(level, "")

	logger.Debugf("SnapConfigs Logger initialized. Log level: %s", logging.GetLevel(""))

	return nil
}

func initializeSnapConfigs() (error) {
	snapConfig := &SnapConfigArray{}
	err := viper.UnmarshalKey("snaps", &snapConfig.SnapConfigs)

	if err != nil {
		return err
	}

	logger.Debug("Found", len(snapConfig.SnapConfigs), "snaps config(s) in yaml file.")

	// append snaps to snapsArray
	for _, snapConfigCopy := range snapConfig.SnapConfigs {
		var snapMetaData SnapConfig = resolveSnapInitAndImplementation(&snapConfigCopy)
		logger.Debug("Adding Snap config:", snapMetaData.Name)
		Snaps = append(Snaps, &snapMetaData)
	}

	return nil
}

func resolveSnapInitAndImplementation(sp *SnapConfig) (SnapConfig) {
	for _, initArgVal := range sp.InitArgsStr {
		logger.Debugf("Appending init arg: %s, concatenating as a byte array: %s\n", initArgVal, []byte(initArgVal))
		sp.InitArgs = append(sp.InitArgs, []byte(initArgVal))
	}
	logger.Debug(len(sp.InitArgs), "InitArgs for snap", sp.Name, "configured.")

	return *sp
}