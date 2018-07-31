/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package updatecmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"strings"

	"github.com/pkg/errors"
	mgmtapi "github.com/securekey/fabric-snaps/configmanager/api"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/securekey/fabric-snaps/sanitize-master"
	"github.com/spf13/cobra"
)

const description = `
The update command allows a client to update the configuration of one or more applications.
Configuration can be specified direcly on the command-line as a JSON string (using the --config option)
or a configuration file may be specified (using the --configfile option).

The format of the configuration for config with peer is as follows:

{
  "MspID": "msp.one",
  "Peers": [
    {
      "PeerID": "peer1",
      "App": [
        {
          "AppName": "app1",
          "Version": "1",
          "Config": "config for app1"
        },
        {
          "AppName": "app1",
          "Version": "2",
          "Config": "config for app1 v2"
           
        },
        {
          "AppName": "app2",
          "Version": "1",
          "Config": "file://path_to_config.yaml"
        }
      ]
    },
    {
      "PeerID":"peer2",
      . . .
	}
  ]
}

The format of the configuration for peer-less config is listed below:
{
  "MspID": "Org1MSP",
  "Apps": [
    {
      "AppName": "app1",
      "Version": "1",
      "Config": "{config goes here}"
    }
    {
      "AppName": "app2",
      "Version": "1",
      "Config": "",
      "Components": [
        {
          "Name": "comp1",
          "Config": "{comp1 data ver 1}",
          "Version": "1"
        },
        {
          "Name": "comp1",
          "Config": "{comp1 data ver 2}",
          "Version": "2"
        },
        {
          "Name": "comp2",
          "Config": "{comp2 data ver 1}",
          "Version": "1"
        }
      ]
    }.....
  ]
}
The configuration may be embedded direcly in the "Config" element or the Config element may reference a file containing the configuration.

`

const examples = `
- Send the update to all peers within the MSP, "Org1MSP" using a configuration file:
    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --configfile ./sampleconfig/org1-config.json

- Send the update to a single peer:
    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --peerurl grpcs://localhost:7051 --configfile ./sampleconfig/org1-config.json

- Send an update using a configuration string specified in the command-line:
    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --config '{"MspID":"Org1MSP","Peers":[{"PeerID":"peer0.org1.example.com","App":[{"AppName":"myapp","Version":"1","Config":"embedded config"}]}]}'

- Send an update using a peer-less configuration string specified in the command-line:
    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --config '{"MspID":"Org1MSP","Apps":[{"AppName":"myapp","Version":"1","Config":"embedded config"}]}'

- Send an update using a peer-less configuration string specified in the command-line:
    $ ./configcli update --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --config '{"MspID":"general", "Apps": [{"AppName": "publickey", "Version": "1", "Components": [{"Name":"sk-td","Config":"{abc}"}] }]}'

`

// Cmd returns the Update command
func Cmd() *cobra.Command {
	return newCmd(action.New())
}

type updateAction struct {
	action.Action
}

func newCmd(baseAction action.Action) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update configuration",
		Long:    description,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			action, err := newUpdateAction(baseAction)
			if err != nil {
				return errors.Wrapf(err, "Error while initializing updateAction")
			}
			if cliconfig.Config().ConfigString() == "" && cliconfig.Config().ClientConfigFile() == "" {
				return errors.New("Please provide a configuration string or a path to a configuration file")
			}
			if len(action.Peers()) == 0 {
				return errors.New("Please specify an orgid, mspid, or a peer to connect to")
			}
			return action.update()
		},
	}

	flags := cmd.Flags()

	cliconfig.InitPeerURL(flags)
	cliconfig.InitChannelID(flags)
	cliconfig.InitConfigString(flags)
	cliconfig.InitConfigFile(flags)
	cliconfig.InitNoPrompt(flags)

	return cmd
}

func newUpdateAction(baseAction action.Action) (*updateAction, error) {
	action := &updateAction{
		Action: baseAction,
	}
	err := action.Initialize()
	return action, err
}

func (a *updateAction) update() error {

	var configString string
	var configFilePath string
	if cliconfig.Config().ConfigString() != "" {
		configString = cliconfig.Config().ConfigString()
	} else {
		configFilePath = cliconfig.Config().ConfigFile()
		if configFilePath == "" {
			return errors.New("you must either specify a config string or a config file")
		}
		var err error
		configString, err = readFile(configFilePath)
		if err != nil {
			return errors.Wrap(err, "error reading config file")
		}
	}

	configMsg, err := configFromString(configString, configFilePath)
	if err != nil {
		return err
	}

	if err := configMsg.IsValid(); err != nil {
		return errors.Wrap(err, "invalid config message")
	}

	configBytes, err := json.Marshal(configMsg)
	if err != nil {
		return errors.Wrapf(err, "error marshalling configuration")
	}

	if !cliconfig.Config().NoPrompt() {
		if !action.YesNoPrompt("Update the configuration for %s?", configMsg.MspID) {
			fmt.Printf("Aborted\n")
			return nil
		}
	}

	if err := a.ExecuteTx(cliconfig.ConfigSnapID, "save", [][]byte{[]byte(configBytes)}); err != nil {
		fmt.Printf("Error invoking chaincode: %s\n", err)
		return errors.Wrap(err, "Update command returned with error")
	}
	fmt.Println("Configuration successfully updated!")

	return nil
}

// configFromString constructs a ConfigMessage from the given config string.
// - configString - Contains the actual configuration
// - baseFilePath - Is the path of the config file, or empty string if the config did not come from a file.
//                  This is used to resolve any relative paths of files referenced within the config.
func configFromString(configString string, baseFilePath string) (*mgmtapi.ConfigMessage, error) {
	configMsg, err := unmarshal([]byte(configString))
	if err != nil {
		return nil, errors.Errorf("Invalid configuration: %s", err)
	}
	newConfigMsg := &mgmtapi.ConfigMessage{
		MspID: configMsg.MspID,
	}
	cliconfig.Config().Logger().Debugf("Config message: %s\n", configMsg)
	for _, peerConfig := range configMsg.Peers {
		newPeerConfig := mgmtapi.PeerConfig{
			PeerID: peerConfig.PeerID,
		}
		for _, appConfig := range peerConfig.App {
			newAppConfig := &appConfig
			// Substitute all of the file refs with the actual contents of the file
			fileRef := appConfig.Config[0:7]
			if fileRef == "file://" {
				refFilePath := newAppConfig.Config[7:]
				contents, err := readFileRef(baseFilePath, refFilePath)
				if err != nil {
					return nil, errors.Wrapf(err, "error retrieving contents of file [%s]", refFilePath)
				}
				newAppConfig.Config = contents
			}
			newPeerConfig.App = append(newPeerConfig.App, *newAppConfig)
		}
		newConfigMsg.Peers = append(newConfigMsg.Peers, newPeerConfig)
	}

	_, err = appConfigInfo(configMsg, baseFilePath, newConfigMsg)
	if err != nil {
		return nil, err
	}

	return newConfigMsg, nil
}

func appConfigInfo(configMsg *mgmtapi.ConfigMessage, baseFilePath string, newConfigMsg *mgmtapi.ConfigMessage) (*mgmtapi.ConfigMessage, error) {

	for _, appConfig := range configMsg.Apps {
		newAppConfig := mgmtapi.AppConfig{
			AppName: appConfig.AppName,
			Version: appConfig.Version,
			Config:  appConfig.Config,
		}
		// Substitute all of the file refs with the actual contents of the file
		if strings.HasPrefix(appConfig.Config, "file://") {
			refFilePath := newAppConfig.Config[7:]
			contents, err := readFileRef(baseFilePath, refFilePath)
			if err != nil {
				return nil, errors.Wrapf(err, "error retrieving contents of file [%s]", refFilePath)
			}
			newAppConfig.Config = contents
		}

		for _, compConfig := range appConfig.Components {
			newCompConfig := &compConfig
			// Substitute all of the file refs with the actual contents of the file
			if strings.HasPrefix(compConfig.Config, "file://") {
				refFilePath := newCompConfig.Config[7:]
				contents, err := readFileRef(baseFilePath, refFilePath)
				if err != nil {
					return nil, errors.Wrapf(err, "error retrieving contents of file [%s]", refFilePath)
				}
				newCompConfig.Config = contents
			}
			newAppConfig.Components = append(newAppConfig.Components, *newCompConfig)
		}
		newConfigMsg.Apps = append(newConfigMsg.Apps, newAppConfig)
	}
	return newConfigMsg, nil
}

func readFile(filePath string) (string, error) {
	cliconfig.Config().Logger().Debugf("Reading file [%s]\n", filePath)
	sanitize.Name(filePath)
	file, err := os.Open(filePath) //nolint: gas
	if err != nil {
		return "", errors.Wrapf(err, "error opening file [%s]", filePath)
	}
	defer file.Close()

	configBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return "", errors.Wrapf(err, "error reading config file [%s]", filePath)
	}
	return string(configBytes), nil
}

func readFileRef(rootFilePath, refFilePath string) (string, error) {
	var path string
	if filepath.IsAbs(refFilePath) {
		path = refFilePath
	} else {
		path = filepath.Join(filepath.Dir(rootFilePath), refFilePath)
	}
	return readFile(path)
}

func unmarshal(updateMsgBytes []byte) (*mgmtapi.ConfigMessage, error) {
	if len(updateMsgBytes) == 0 {
		return nil, errors.New("config message is empty")
	}
	configMsg := &mgmtapi.ConfigMessage{}
	if err := json.Unmarshal(updateMsgBytes, &configMsg); err != nil {
		return nil, errors.Wrap(err, "error unmarshalling config message")
	}
	return configMsg, nil
}
