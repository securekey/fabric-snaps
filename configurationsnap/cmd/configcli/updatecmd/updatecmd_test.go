/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package updatecmd

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
)

const (
	clientConfigPath = "../testdata/clientconfig/config.yaml"
)

func TestInvalidClientConfig(t *testing.T) {
	execute(t, true, "--clientconfig", "invalidconfig.yaml")
}

func TestNoConfig(t *testing.T) {
	execute(t, true, "--clientconfig", clientConfigPath)
}

func TestInvalidConfigString(t *testing.T) {
	invalidConfigString := `{"MspID":"Org1MSP"}`
	execute(t, true, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--config", invalidConfigString, "--noprompt")
}

func TestValidConfigString(t *testing.T) {
	configString := `{"MspID":"Org1MSP","Peers":[{"PeerID":"peer0.org1.example.com","App":[{"AppName":"myapp","Version":"1","Config":"embedded config"}]}]}`
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--config", configString, "--noprompt")
}

func TestValidConfigWithAppsNoPeerConfig(t *testing.T) {
	configString := `{"MspID":"Org1MSP", "Apps": [{"AppName": "publickey", "Version": "1", "Config": "{type:a, key:b}" }]}`
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--config", configString, "--noprompt")
}

func TestValidConfigWithAppsAndComponentsConfig(t *testing.T) {
	configString := `{"MspID":"general", "Apps": [{"AppName": "publickey", "Version": "1", "Components": [{"Name":"sk-td","Config":"{abc}"}] }]}`
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--config", configString, "--noprompt")
}

func TestInvalidConfigFile(t *testing.T) {
	execute(t, true, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configfile", "invalid-config.json", "--noprompt")
}

func TestValidConfigFile(t *testing.T) {
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configfile", "../sampleconfig/org1-config.json", "--noprompt")
}

func execute(t *testing.T, expectError bool, args ...string) {
	cmd := newCmd(newMockAction())
	action.InitGlobalFlags(cmd.PersistentFlags())

	cmd.SetArgs(args)
	err := cmd.Execute()
	if expectError && err == nil {
		t.Fatalf("expecting error but got none")
	} else if !expectError && err != nil {
		t.Fatalf("got error %s", err)
	}
}

func newMockAction() *action.MockAction {
	return &action.MockAction{
		Invoker: func(chaincodeID, fctn string, args [][]byte) ([]byte, error) {
			if chaincodeID != cliconfig.ConfigSnapID {
				return nil, errors.Errorf("expecting chaincode ID [%s] but got [%s]", cliconfig.ConfigSnapID, chaincodeID)
			}
			if fctn == "save" {
				if len(args) == 0 {
					return nil, errors.New("expecting one arg but got none")
				}
				configMessage, err := unmarshal(args[0])
				if err != nil {
					return nil, errors.Wrap(err, "got error unmarshalling config message arg")
				}
				if err := configMessage.IsValid(); err != nil {
					return nil, errors.Wrap(err, "invalid config message")
				}
			} else if fctn == "refresh" {
				if len(args) != 0 {
					return nil, errors.New("expecting zero arg for refresh")
				}
			} else {
				return nil, errors.Errorf("expecting function [save] or [refresh] but got [%s]", fctn)
			}
			return nil, nil
		},
	}
}
