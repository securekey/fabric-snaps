/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package generateCSR

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
)

const (
	clientConfigPath = "../testdata/clientconfig/config.yaml"
)

func TestValidRequestParameters(t *testing.T) {
	execute(t, false, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512", "--csrCommonName", "something")
}

func TestAnEmptyKey(t *testing.T) {
	//key type is mandatory field
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512", "--csrCommonName", "something")
}

func TestInvalidKey(t *testing.T) {
	//key type is mandatory field
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "FAKE", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512", "--csrCommonName", "something")
}
func TestInvalidEphemeralFlag(t *testing.T) {
	//ephemeral flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "", "--ephemeral", "false-FAKE", "--sigAlg", "ECDSAWithSHA512", "--csrCommonName", "something")
}

func TestAnEmptySigAlg(t *testing.T) {
	//SigAlg flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "", "--csrCommonName", "something")
}
func TestInvalidSigAlg(t *testing.T) {
	//SigAlg flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "ABC", "--csrCommonName", "something")
}

func TestValidCSRName(t *testing.T) {
	//csr common name flag should be set
	execute(t, false, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512", "--csrCommonName", "something")
}

func TestInValidCSRName(t *testing.T) {
	//csr common name flag should be set
	execute(t, true, nil, "--clientconfig", clientConfigPath, "--cid", "mychannel", "--keyType", "ECDSA", "--ephemeral", "false", "--sigAlg", "ECDSAWithSHA512", "--csrCommonName", "")
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
