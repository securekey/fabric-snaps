/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generateCSR

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-sdk-go/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
)

const (
	clientConfigPath = "../testdata/clientconfig/config.yaml"
)

func TestValidRequestParameters(t *testing.T) {
	execute(t, false, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512")
}

func TestAnEmptyKey(t *testing.T) {
	//key type is mandatory field
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512")
}

func TestInvalidKey(t *testing.T) {
	//key type is mandatory field
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "FAKE", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512")
}
func TestInvalidEphemeralFlag(t *testing.T) {
	//ephemeral flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "", "--ephemeral", "false-FAKE", "--sigAlg", "ECDSAWithSHA512")
}

func TestAnEmptySigAlg(t *testing.T) {
	//ephemeral flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "")
}
func TestInvalidSigAlg(t *testing.T) {
	//ephemeral flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "ABC")
}

func execute(t *testing.T, expectError bool, response []byte, args ...string) {
	cmd := newCmd(newMockAction(response))
	action.InitGlobalFlags(cmd.PersistentFlags())
	fmt.Printf("%s ", args)
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
			if fctn != "generateCSR" {
				return nil, errors.Errorf("expecting function [get] but got [%s]", fctn)
			}
			if len(args) == 0 {
				return nil, errors.New("expecting one arg but got none")
			}

			return response, nil
		},
	}
}
