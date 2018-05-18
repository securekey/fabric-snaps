/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package querycmd

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/configkeyutil"
)

const (
	clientConfigPath = "../testdata/clientconfig/config.yaml"
	configKV         = `[{"Key":{"MspID":"Org1MSP","PeerID":"peer0.org1.example.com","AppName":"myapp","Version":"1"},"Value":"ZW1iZWRkZWQgY29uZmln"}]`
)

func TestInvalidClientConfig(t *testing.T) {
	execute(t, true, nil, "--clientconfig", "invalidconfig.yaml")
}

func TestNoConfigKey(t *testing.T) {
	execute(t, true, nil, "--clientconfig", clientConfigPath)
}

func TestInvalidConfigKey(t *testing.T) {
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configkey", "{")
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configkey", "{}")
}

func TestValidConfigKey(t *testing.T) {
	configKey := `{"MspID":"Org1MSP"}`
	execute(t, false, []byte(configKV), "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--configkey", configKey)
}

func TestValidConfigKeyOptions(t *testing.T) {
	// Uses Org1MSP
	execute(t, false, []byte(configKV), "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1")
	// Display in raw format
	execute(t, false, []byte(configKV), "--clientconfig", clientConfigPath, "--cid", "mychannel", "--mspid", "Org1MSP", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1", "--format", "raw")
	// Uses default org
	execute(t, false, []byte(configKV), "--clientconfig", clientConfigPath, "--cid", "mychannel", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1")
	// Uses org2
	execute(t, false, []byte(configKV), "--clientconfig", clientConfigPath, "--cid", "mychannel", "--orgid", "org2", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1")
	// Uses peer URL grpcs://peer0.org1.example.com:7053
	execute(t, false, []byte(configKV), "--clientconfig", clientConfigPath, "--cid", "mychannel", "--peerurl", "grpcs://peer0.org1.example.com:7051", "--peerid", "peer0.org1.example.com", "--appname", "myapp", "--configver", "1")
}

func execute(t *testing.T, expectError bool, response []byte, args ...string) {
	cmd := newCmd(newMockAction(response))
	action.InitGlobalFlags(cmd.PersistentFlags())

	cmd.SetArgs(args)
	err := cmd.Execute()
	if expectError && err == nil {
		t.Fatalf("expecting error but got none")
	} else if !expectError && err != nil {
		t.Fatalf("got error %s", err)
	}
}

func newMockAction(response []byte) *action.MockAction {
	return &action.MockAction{
		Response: response,
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
			return response, nil
		},
	}
}
