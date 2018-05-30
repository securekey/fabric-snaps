/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package deletecmd

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/action"
	"github.com/securekey/fabric-snaps/configurationsnap/cmd/configcli/cliconfig"
	"github.com/spf13/cobra"
)

const description = `
The delete command allows the client to delete the org's configuration using a Config Key.
The Config Key consists of:

* MspID (mandatory) - The MSP ID of the organization
* PeerID (optional) - The ID of the peer
* AppName (optional) - The application name
* AppVer (optional) - The application version
* ComponentName (optional) - The component name
* ComponentVer (optional) - The component version


The Config Key may be specified as a JSON string (using the --configkey option) or it may be
specified using the options: --mspid, --peerid, --appname, --appver, --componentname and --componentver. A specific application
configuration may be deleted if PeerID and AppName are specified, or the org's entire configuration
may be deleted if only MspID is specified.
`

const examples = `
- Delete a the configuration of a particular application:
    $ ./configcli delete --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --peerid peer0.org1.example.com --appname myapp --appver 1

- Delete a the configuration of a particular component:
    $ ./configcli delete --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP --appname myapp --appver 1 --componentname comp1 --componentver 1

- Delete all configuration in Org1MSP:
    $ ./configcli delete --clientconfig ../../../bddtests/fixtures/clientconfig/config.yaml --cid mychannel --mspid Org1MSP
`

// Cmd returns the Delete command
func Cmd() *cobra.Command {
	return newCmd(action.New())
}

type deleteAction struct {
	action.Action
}

func newCmd(baseAction action.Action) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Short:   "Delete configuration",
		Long:    description,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			action, err := newDeleteAction(baseAction)
			if err != nil {
				return errors.Errorf("Error while initializing deleteAction: %s", err)
			}
			if len(action.Peers()) == 0 {
				return errors.Errorf("Please specify an orgid, mspid, or a peer to connect to")
			}
			return action.delete()
		},
	}

	flags := cmd.Flags()

	cliconfig.InitPeerURL(flags)
	cliconfig.InitChannelID(flags)
	cliconfig.InitConfigKey(flags)
	cliconfig.InitPeerID(flags)
	cliconfig.InitAppName(flags)
	cliconfig.InitAppVer(flags)
	cliconfig.InitComponentName(flags)
	cliconfig.InitComponentVer(flags)
	cliconfig.InitNoPrompt(flags)

	return cmd
}

func newDeleteAction(baseAction action.Action) (*deleteAction, error) {
	action := &deleteAction{
		Action: baseAction,
	}
	err := action.Initialize()
	return action, err
}

func (a *deleteAction) delete() error {
	key, err := a.ConfigKey()
	if err != nil {
		return err
	}

	if key.MspID == "" {
		return errors.New("invalid config key: MspID not specified")
	}

	configKeyBytes, err := json.Marshal(key)
	if err != nil {
		return errors.Wrapf(err, "error marshalling config key")
	}

	cliconfig.Config().Logger().Debugf("Using config key: %s\n", configKeyBytes)

	if !cliconfig.Config().NoPrompt() {
		if !action.YesNoPrompt("Delete the configuration for %s?", configKeyBytes) {
			fmt.Printf("Aborted\n")
			return nil
		}
	}

	if err := a.ExecuteTx(cliconfig.ConfigSnapID, "delete", [][]byte{[]byte(configKeyBytes)}); err != nil {
		fmt.Printf("Error invoking chaincode: %s\n", err)
	} else {
		fmt.Println("Invocation successful!")
	}

	return nil
}
