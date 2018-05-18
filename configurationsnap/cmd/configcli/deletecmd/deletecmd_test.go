/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deletecmd

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/configkeyutil"
)

const (
	clientConfigPath = "../testdata/clientconfig/config.yaml"
)

func TestInvalidClientConfig(t *testing.T) {
	execute(t, true, "--clientconfig", "invalidconfig.yaml")
}

func TestNoConfigKey(t *testing.T) {
	execute(t, true, "--clientconfig", clientConfigPath)
}

func TestInvalidConfigKey(t *testing.T) {
	execute(t, true, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configkey", "{")
	execute(t, true, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configkey", "{}")
}

func TestValidConfigKey(t *testing.T) {
	configKey := `{"MspID":"Org1MSP"}`
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configkey", configKey, "--noprompt")
}

func TestValidConfigKeyOptions(t *testing.T) {
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1", "--noprompt")
	execute(t, false, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1", "--noprompt")
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
			if fctn != "get" {
				return nil, errors.Errorf("expecting function [get] but got [%s]", fctn)
			}
			if len(args) == 0 {
				return nil, errors.New("expecting one arg but got none")
			}
			key, err := configkeyutil.Unmarshal(args[0])
			if err != nil {
				return nil, errors.Wrap(err, "got error unmarshalling config key arg")
			}
			if key.MspID == "" {
				return nil, errors.New("MSP ID must be provided in config key")
			}
			return nil, nil
		},
	}
}
